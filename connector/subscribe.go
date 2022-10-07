package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/sub"
	"github.com/nats-io/nats.go"
)

/*
Subscribe assigns a function to handle HTTP requests to the given path.
If the path ends with a / all sub-paths under the path are capture by the subscription

If the path does not include a host name, the default host is used.
If a port is not specified, 443 is used by default.

Examples of valid paths:

	(empty)
	/
	:1080
	:1080/
	:1080/path
	/path/with/slash
	path/with/no/slash
	https://www.example.com/path
	https://www.example.com:1080/path
*/
func (c *Connector) Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option) error {
	if c.hostName == "" {
		return errors.New("host name is not set")
	}
	newSub, err := sub.NewSub(c.hostName, path, options...)
	if err != nil {
		return errors.Trace(err)
	}
	newSub.Handler = handler
	if c.IsStarted() {
		err := c.activateSub(newSub)
		if err != nil {
			return errors.Trace(err)
		}
		time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS
	}
	key := newSub.Canonical()
	c.subsLock.Lock()
	c.subs[key] = newSub
	c.subsLock.Unlock()
	return nil
}

// Unsubscribe removes the handler for the specified path
func (c *Connector) Unsubscribe(path string) error {
	newSub, err := sub.NewSub(c.hostName, path)
	if err != nil {
		return errors.Trace(err)
	}
	key := newSub.Canonical()
	c.subsLock.Lock()
	if sub, ok := c.subs[key]; ok {
		err = c.deactivateSub(sub)
		if err == nil {
			delete(c.subs, key)
		}
	}
	c.subsLock.Unlock()
	return errors.Trace(err)
}

// UnsubscribeAll removes all handlers
func (c *Connector) UnsubscribeAll() error {
	c.subsLock.Lock()
	defer c.subsLock.Unlock()

	var lastErr error
	for _, sub := range c.subs {
		lastErr = c.deactivateSub(sub)
	}
	c.subs = map[string]*sub.Subscription{}
	return errors.Trace(lastErr)
}

// activateSub will subscribe to NATS
func (c *Connector) activateSub(s *sub.Subscription) error {
	handler := func(msg *nats.Msg) {
		err := c.ackRequest(msg, s)
		if err != nil {
			c.LogError(err)
			return
		}
		go func() {
			err := c.onRequest(msg, s)
			if err != nil {
				c.LogError(err)
			}
		}()
	}

	var err error
	if s.HostSub == nil {
		if s.Queue != "" {
			s.HostSub, err = c.natsConn.QueueSubscribe(subjectOfSubscription(c.plane, s.Host, s.Port, s.Path), s.Queue, handler)
		} else {
			s.HostSub, err = c.natsConn.Subscribe(subjectOfSubscription(c.plane, s.Host, s.Port, s.Path), handler)
		}
		if err != nil {
			return errors.Trace(err)
		}
	}
	if s.DirectSub == nil {
		if s.Queue != "" {
			s.DirectSub, err = c.natsConn.QueueSubscribe(subjectOfSubscription(c.plane, c.id+"."+s.Host, s.Port, s.Path), s.Queue, handler)
		} else {
			s.DirectSub, err = c.natsConn.Subscribe(subjectOfSubscription(c.plane, c.id+"."+s.Host, s.Port, s.Path), handler)
		}
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// deactivateSub will unsubscribe from NATS
func (c *Connector) deactivateSub(s *sub.Subscription) error {
	var lastErr error
	if s.HostSub != nil {
		err := s.HostSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err, s.Canonical())
			c.LogError(err)
		} else {
			s.HostSub = nil
		}
	}
	if s.DirectSub != nil {
		err := s.DirectSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err, s.Canonical())
			c.LogError(err)
		} else {
			s.DirectSub = nil
		}
	}
	return lastErr
}

// ackRequest sends an ack response back to the caller.
// Acks are sent as soon as a request is received to let the caller know it is
// being processed
func (c *Connector) ackRequest(msg *nats.Msg, s *sub.Subscription) error {
	// Parse only the headers of the request
	headerData := msg.Data
	eoh := bytes.Index(headerData, []byte("\r\n\r\n"))
	if eoh >= 0 {
		headerData = headerData[:eoh+4]
	}
	httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(headerData)))
	if err != nil {
		return errors.Trace(err)
	}

	// Get return address
	fromHost := frame.Of(httpReq).FromHost()
	if fromHost == "" {
		return errors.New("empty " + frame.HeaderFromHost + " header")
	}
	fromID := frame.Of(httpReq).FromID()
	if fromID == "" {
		return errors.New("empty " + frame.HeaderFromId + " header")
	}
	msgID := frame.Of(httpReq).MessageID()
	if msgID == "" {
		return errors.New("empty " + frame.HeaderMsgId + " header")
	}
	queue := s.Queue
	if queue == "" {
		queue = c.id + "." + c.hostName
	}

	// Prepare and send the ack
	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 202 Accepted\r\nConnection: close")
	header := map[string]string{
		frame.HeaderOpCode:   frame.OpCodeAck,
		frame.HeaderFromHost: c.hostName,
		frame.HeaderFromId:   c.id,
		frame.HeaderMsgId:    msgID,
		frame.HeaderQueue:    queue,
	}
	for k, v := range header {
		buf.WriteString("\r\n")
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v)
	}
	buf.WriteString("\r\n\r\n")

	err = c.natsConn.Publish(subjectOfResponses(c.plane, fromHost, fromID), buf.Bytes())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// onRequest is called when an incoming HTTP request is received.
// The message is dispatched to the appropriate web handler and the response is serialized and sent back to the response channel of the sender
func (c *Connector) onRequest(msg *nats.Msg, s *sub.Subscription) error {
	// Parse the request
	httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(msg.Data)))
	if err != nil {
		return errors.Trace(err)
	}

	// Fill in the gaps
	httpReq.URL.Host = fmt.Sprintf("%s:%d", s.Host, s.Port)
	httpReq.URL.Scheme = "https"

	// Get the sender host name and message ID
	fromHost := frame.Of(httpReq).FromHost()
	fromId := frame.Of(httpReq).FromID()
	msgID := frame.Of(httpReq).MessageID()
	queue := s.Queue
	if queue == "" {
		queue = c.id + "." + c.hostName
	}

	// Time budget
	budget := frame.Of(httpReq).TimeBudget()
	if budget <= c.networkHop {
		return errors.New("timeout")
	}

	// Prepare the context
	// Set the context's timeout to the time budget reduced by a network hop
	var cancel context.CancelFunc
	ctx := context.WithValue(context.Background(), frame.ContextKey, httpReq.Header)
	ctx, cancel = context.WithTimeout(ctx, budget-c.networkHop)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)

	// Call the web handler
	httpRecorder := httptest.NewRecorder()
	handlerErr := catchPanic(func() error {
		return s.Handler(httpRecorder, httpReq)
	})

	if handlerErr != nil {
		handlerErr = errors.Trace(handlerErr, s.Canonical())
		c.LogError(handlerErr)

		// Prepare an error response instead
		httpRecorder = httptest.NewRecorder()
		httpRecorder.Header().Set("Content-Type", "application/json")
		body, err := json.MarshalIndent(handlerErr, "", "\t")
		if err != nil {
			return errors.Trace(err)
		}
		httpRecorder.WriteHeader(http.StatusInternalServerError)
		httpRecorder.Write(body)
	}

	// Set control headers on the response
	httpResponse := httpRecorder.Result()
	frame.Of(httpResponse).SetMessageID(msgID)
	frame.Of(httpResponse).SetFromHost(c.hostName)
	frame.Of(httpResponse).SetFromID(c.id)
	frame.Of(httpResponse).SetQueue(queue)
	frame.Of(httpResponse).SetOpCode(frame.OpCodeResponse)
	if handlerErr != nil {
		frame.Of(httpResponse).SetOpCode(frame.OpCodeError)
	}

	// Send back the response
	var buf bytes.Buffer
	err = httpResponse.Write(&buf)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.natsConn.Publish(subjectOfResponses(c.plane, fromHost, fromId), buf.Bytes())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
