package connector

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/microbus-io/fabric/rand"
	"github.com/nats-io/nats.go"
)

// subscription holds the specs of the NATS subscription for a given port and path
type subscription struct {
	port             int
	path             string
	handler          func(w http.ResponseWriter, r *http.Request)
	natsSubscription *nats.Subscription
}

// GET makes a GET request
func (c *Connector) GET(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Request(request)
}

// POST makes a POST request
func (c *Connector) POST(url string, body []byte) (*http.Response, error) {
	request, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return c.Request(request)
}

// Request makes an HTTP request then awaits and returns the response
func (c *Connector) Request(req *http.Request) (*http.Response, error) {
	// Set a random message ID and the return address
	msgID := rand.AlphaNum64(8)
	req.Header.Set("Microbus-Msg-Id", msgID)
	req.Header.Set("Microbus-From-Host", c.hostName)
	req.Header.Set("Microbus-From-Id", c.id)

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
	var buf bytes.Buffer
	err := req.Write(&buf)
	if err != nil {
		return nil, err
	}
	port := 443
	if req.URL.Scheme == "http" {
		port = 80
	}
	if req.URL.Port() != "" {
		port64, err := strconv.ParseInt(req.URL.Port(), 10, 32)
		if err != nil {
			return nil, err
		}
		port = int(port64)
	}
	subject := subjectOfRequest(req.URL.Hostname(), port, req.URL.Path)
	err = c.natsConn.Publish(subject, buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Await and return the response
	timeoutTimer := time.NewTimer(time.Minute)
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
		c.LogError(err)
		return
	}

	// Push it to the channel matching the message ID
	msgID := response.Header.Get("Microbus-Msg-Id")
	c.reqsLock.Lock()
	ch, ok := c.reqs[msgID]
	c.reqsLock.Unlock()
	if !ok {
		c.LogInfo("response received after timeout: %s", msgID)
	}
	ch <- response
}

// onRequest is called when an incoming HTTP request is received.
// The message is dispatched to the appropriate web handler and the response is serialized and sent back to the reply channel of the sender
func (c *Connector) onRequest(msg *nats.Msg, handler func(w http.ResponseWriter, r *http.Request)) error {
	// Parse the request
	httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(msg.Data)))
	if err != nil {
		return err
	}

	// Get the sender host name and message ID
	fromHost := httpReq.Header.Get("Microbus-From-Host")
	fromId := httpReq.Header.Get("Microbus-From-Id")
	msgID := httpReq.Header.Get("Microbus-Msg-Id")

	// Prepare an HTTP recorder
	httpRecorder := httptest.NewRecorder()

	// Echo the message ID in the reply
	httpRecorder.Header().Set("Microbus-Msg-Id", msgID)
	httpRecorder.Header().Set("Microbus-From-Host", c.hostName)
	httpRecorder.Header().Set("Microbus-From-Id", c.id)

	// Call the web handler
	handler(httpRecorder, httpReq)

	// Serialize the response
	httpResponse := httpRecorder.Result()
	var buf bytes.Buffer
	err = httpResponse.Write(&buf)
	if err != nil {
		return err
	}

	// Send back the reply
	err = c.natsConn.Publish(subjectOfReply(fromHost, fromId), buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// Subscribe assigns a function to handle web requests to the given port and path.
// If the path ends with a / all sub-paths under the path are capture by the subscription
func (c *Connector) Subscribe(port int, path string, handler func(w http.ResponseWriter, r *http.Request)) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	newSub := &subscription{
		port:    port,
		path:    path,
		handler: handler,
	}
	if c.IsStarted() {
		err := c.activateSub(newSub)
		if err != nil {
			return err
		}
		time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS
	}
	c.subsLock.Lock()
	c.subs = append(c.subs, newSub)
	c.subsLock.Unlock()
	return nil
}

func (c *Connector) activateSub(sub *subscription) error {
	var err error
	sub.natsSubscription, err = c.natsConn.Subscribe(subjectOfSubscription(c.hostName, sub.port, sub.path), func(msg *nats.Msg) {
		go func() {
			err := c.onRequest(msg, sub.handler)
			if err != nil {
				c.LogError(err)
			}
		}()
	})
	if err != nil {
		return err
	}
	return nil
}
