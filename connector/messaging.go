package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/nats-io/nats.go"
)

// GET makes a GET request
func (c *Connector) GET(ctx context.Context, url string) (*http.Response, error) {
	return c.Publish(ctx, []pub.Option{
		pub.GET(url),
	}...)
}

// POST makes a POST request.
// Body of type io.Reader, []byte and string is serialized in binary form.
// All other types are serialized as JSON
func (c *Connector) POST(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.Publish(ctx, []pub.Option{
		pub.POST(url),
		pub.Body(body),
	}...)
}

// Publish makes an HTTP request then awaits and returns the response
func (c *Connector) Publish(ctx context.Context, options ...pub.Option) (*http.Response, error) {
	// Build the request
	req, err := pub.NewRequest(options...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Restrict the time budget to the context deadline
	deadline, ok := ctx.Deadline()
	if ok {
		req.Apply(pub.Deadline(deadline))
	} else if req.Deadline.IsZero() {
		// If no budget is set, use the default
		req.Apply(pub.TimeBudget(c.defaultTimeBudget))
	}
	// Check if there's enough time budget
	if !req.Deadline.After(time.Now().Add(-c.networkHop)) {
		return nil, errors.New("timeout")
	}

	// Limit number of hops
	depth := frame.Of(ctx).CallDepth()
	if depth >= c.maxCallDepth {
		return nil, errors.New("call depth overflow")
	}
	frame.Of(req.Header).SetCallDepth(depth + 1)

	// Set return address
	frame.Of(req.Header).SetFromHost(c.hostName)
	frame.Of(req.Header).SetFromID(c.id)

	// Make the request
	httpRes, err := c.makeHTTPRequest(req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Reconstitute the error if an error op code is returned
	if frame.Of(httpRes).OpCode() == frame.OpCodeError {
		var tracedError errors.TracedError
		body, err := io.ReadAll(httpRes.Body)
		if err != nil {
			return nil, errors.Trace(err)
		}
		u, err := url.Parse(req.URL)
		if err != nil {
			return nil, errors.Trace(err)
		}
		err = json.Unmarshal(body, &tracedError)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(&tracedError, c.hostName+" -> "+u.Hostname())
	}

	return httpRes, nil
}

// makeHTTPRequest makes an HTTP request then awaits and returns the response
func (c *Connector) makeHTTPRequest(req *pub.Request) (*http.Response, error) {
	// Set a random message ID
	msgID := rand.AlphaNum64(8)
	frame.Of(req.Header).SetMessageID(msgID)

	// Prepare the HTTP request
	httpReq, err := req.ToHTTP()
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Create a channel to await on
	awaitCh := make(chan *http.Response)
	c.reqsLock.Lock()
	c.reqs[msgID] = awaitCh
	c.reqsLock.Unlock()
	defer func() {
		c.reqsLock.Lock()
		delete(c.reqs, msgID)
		c.reqsLock.Unlock()
	}()

	// Send the message
	port := 443
	if httpReq.URL.Scheme == "http" {
		port = 80
	}
	if httpReq.URL.Port() != "" {
		port64, err := strconv.ParseInt(httpReq.URL.Port(), 10, 32)
		if err != nil {
			return nil, errors.Trace(err)
		}
		port = int(port64)
	}
	subject := subjectOfRequest(c.plane, httpReq.URL.Hostname(), port, httpReq.URL.Path)

	var buf bytes.Buffer
	err = httpReq.Write(&buf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = c.natsConn.Publish(subject, buf.Bytes())
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Await and return the response
	budget := time.Until(req.Deadline)
	timeoutTimer := time.NewTimer(budget)
	defer timeoutTimer.Stop()
	select {
	case response := <-awaitCh:
		return response, nil
	case <-timeoutTimer.C:
		return nil, errors.New("timeout")
	}
}

// onReply is called when a reply to an outgoing request is received
func (c *Connector) onReply(msg *nats.Msg) {
	// Parse the response
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(msg.Data)), nil)
	if err != nil {
		c.LogError(context.Background(), "Failed to parse response", err)
		return
	}

	// Push it to the channel matching the message ID
	msgID := frame.Of(response).MessageID()
	c.reqsLock.Lock()
	ch, ok := c.reqs[msgID]
	c.reqsLock.Unlock()
	if !ok {
		c.LogInfo(context.Background(), "Response received after timeout", log.String("messageId", msgID))
		return
	}
	select {
	case ch <- response:
	default:
		c.LogInfo(context.Background(), "No listener on channel", log.String("messageId", msgID))
	}
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

	// Call the web handler
	handlerErr := catchPanic(func() error {
		return s.Handler(httpRecorder, httpReq)
	})

	if handlerErr != nil {
		handlerErr = errors.Trace(handlerErr, fmt.Sprintf("%s:%d%s", s.Host, s.Port, s.Path))
		c.LogError(context.Background(), "Handler error", handlerErr)

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
	c.subsLock.Lock()
	c.subs = append(c.subs, newSub)
	c.subsLock.Unlock()
	return nil
}

func (c *Connector) activateSub(s *sub.Subscription) error {
	var err error
	s.NATSSub, err = c.natsConn.QueueSubscribe(subjectOfSubscription(c.plane, c.hostName, s.Port, s.Path), c.hostName, func(msg *nats.Msg) {
		go func() {
			err := c.onRequest(msg, s)
			if err != nil {
				c.LogError(context.Background(), "Failed request", err)
			}
		}()
	})
	return errors.Trace(err)
}
