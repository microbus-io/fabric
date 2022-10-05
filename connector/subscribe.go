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
	newSub, err := sub.NewSub(c.hostName, path)
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
	key := fmt.Sprintf("%s:%d%s", newSub.Host, newSub.Port, newSub.Path)
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
	key := fmt.Sprintf("%s:%d%s", newSub.Host, newSub.Port, newSub.Path)
	c.subsLock.Lock()
	if sub, ok := c.subs[key]; ok {
		if sub.NATSSub != nil {
			err = errors.Trace(sub.NATSSub.Unsubscribe())
		}
		delete(c.subs, key)
	}
	c.subsLock.Unlock()
	return err
}

// UnsubscribeAll removes all handlers
func (c *Connector) UnsubscribeAll() error {
	c.subsLock.Lock()
	defer c.subsLock.Unlock()

	var lastErr error
	for _, sub := range c.subs {
		if sub.NATSSub != nil {
			err := sub.NATSSub.Unsubscribe()
			if err != nil {
				lastErr = errors.Trace(err)
				c.LogError(err)
			}
		}
	}
	c.subs = map[string]*sub.Subscription{}
	return lastErr
}

func (c *Connector) activateSub(s *sub.Subscription) error {
	var err error
	s.NATSSub, err = c.natsConn.QueueSubscribe(subjectOfSubscription(c.plane, c.hostName, s.Port, s.Path), c.hostName, func(msg *nats.Msg) {
		err := c.ackRequest(msg, s)
		if err != nil {
			c.LogError(err)
		}
		go func() {
			err := c.onRequest(msg, s)
			if err != nil {
				c.LogError(err)
			}
		}()
	})
	return errors.Trace(err)
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
	fromID := frame.Of(httpReq).FromID()
	msgID := frame.Of(httpReq).MessageID()

	// Prepare and send the reply
	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 202 Accepted\r\n")
	buf.WriteString("Connection: close\r\n")
	buf.WriteString("Microbus-Op-Code: " + frame.OpCodeAck + "\r\n")
	buf.WriteString("Microbus-From-Host: " + c.hostName + "\r\n")
	buf.WriteString("Microbus-From-Id: " + c.id + "\r\n")
	buf.WriteString("Microbus-Msg-Id: " + msgID + "\r\n")
	buf.WriteString("\r\n")

	err = c.natsConn.Publish(subjectOfReply(c.plane, fromHost, fromID), buf.Bytes())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// onRequest is called when an incoming HTTP request is received.
// The message is dispatched to the appropriate web handler and the response is serialized and sent back to the reply channel of the sender
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

	// Prepare an HTTP recorder
	httpRecorder := httptest.NewRecorder()

	// Echo the message ID in the reply
	frame.Of(httpRecorder).SetMessageID(msgID)
	frame.Of(httpRecorder).SetFromHost(c.hostName)
	frame.Of(httpRecorder).SetFromID(c.id)
	frame.Of(httpRecorder).SetOpCode(frame.OpCodeResponse)

	// Call the web handler
	handlerErr := catchPanic(func() error {
		return s.Handler(httpRecorder, httpReq)
	})

	if handlerErr != nil {
		handlerErr = errors.Trace(handlerErr, fmt.Sprintf("%s:%d%s", s.Host, s.Port, s.Path))
		c.LogError(handlerErr)

		// Prepare an error response instead
		httpRecorder = httptest.NewRecorder()
		frame.Of(httpRecorder).SetMessageID(msgID)
		frame.Of(httpRecorder).SetFromHost(c.hostName)
		frame.Of(httpRecorder).SetFromID(c.id)
		frame.Of(httpRecorder).SetOpCode(frame.OpCodeError)
		httpRecorder.Header().Set("Content-Type", "application/json")
		body, err := json.MarshalIndent(handlerErr, "", "\t")
		if err != nil {
			return errors.Trace(err)
		}
		httpRecorder.WriteHeader(http.StatusInternalServerError)
		httpRecorder.Write(body)
	}

	// Send back the reply
	var buf bytes.Buffer
	err = httpRecorder.Result().Write(&buf)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.natsConn.Publish(subjectOfReply(c.plane, fromHost, fromId), buf.Bytes())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
