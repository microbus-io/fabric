package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
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
	frame.Of(req.Header).SetOpCode(frame.OpCodeRequest)

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
	awaitCh := make(chan *http.Response, 2)
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
	acked := false
	budget := time.Until(req.Deadline)
	timeoutTimer := time.NewTimer(budget)
	defer timeoutTimer.Stop()
	ackTimer := time.NewTimer(c.networkHop)
	defer ackTimer.Stop()
	for {
		select {
		case response := <-awaitCh:
			opCode := frame.Of(response).OpCode()

			if opCode == frame.OpCodeAck {
				acked = true
			}

			if opCode == frame.OpCodeResponse || opCode == frame.OpCodeError {
				return response, nil
			}
		case <-timeoutTimer.C:
			return nil, errors.New("timeout")
		case <-ackTimer.C:
			if !acked {
				return nil, errors.New("ack timeout")
			}
		}
	}
}

// onResponse is called when a response to an outgoing request is received
func (c *Connector) onResponse(msg *nats.Msg) {
	// Parse the response
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(msg.Data)), nil)
	if err != nil {
		c.LogError(err)
		return
	}

	// Push it to the channel matching the message ID
	msgID := frame.Of(response).MessageID()
	c.reqsLock.Lock()
	ch, ok := c.reqs[msgID]
	c.reqsLock.Unlock()
	if !ok {
		c.LogInfo("Response received after timeout: %s", msgID)
		return
	}
	ch <- response
}
