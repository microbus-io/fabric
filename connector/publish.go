package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	return c.Request(ctx, []pub.Option{
		pub.GET(url),
		pub.Unicast(),
	}...)
}

// POST makes a POST request.
// Body of type io.Reader, []byte and string is serialized in binary form.
// All other types are serialized as JSON
func (c *Connector) POST(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.Request(ctx, []pub.Option{
		pub.POST(url),
		pub.Body(body),
		pub.Unicast(),
	}...)
}

// Request makes an HTTP request then awaits and returns a single response synchronously
func (c *Connector) Request(ctx context.Context, options ...pub.Option) (*http.Response, error) {
	options = append(options, pub.Unicast())
	ch := c.Publish(ctx, options...)
	return (<-ch).Get()
}

// Publish makes an HTTP request then awaits and returns the responses asynchronously
func (c *Connector) Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	errOutput := make(chan *pub.Response, 1)
	defer close(errOutput)

	// Build the request
	req, err := pub.NewRequest(options...)
	if err != nil {
		errOutput <- pub.NewErrorResponse(errors.Trace(err))
		return errOutput
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
		errOutput <- pub.NewErrorResponse(errors.New("timeout"))
		return errOutput
	}

	// Limit number of hops
	depth := frame.Of(ctx).CallDepth()
	if depth >= c.maxCallDepth {
		errOutput <- pub.NewErrorResponse(errors.New("call depth overflow"))
		return errOutput
	}
	frame.Of(req.Header).SetCallDepth(depth + 1)

	// Set return address
	frame.Of(req.Header).SetFromHost(c.hostName)
	frame.Of(req.Header).SetFromID(c.id)
	frame.Of(req.Header).SetOpCode(frame.OpCodeRequest)

	// Make the request
	var output chan *pub.Response
	if req.Multicast {
		output = make(chan *pub.Response, 64)
	} else {
		output = make(chan *pub.Response, 2)
	}
	go func() {
		c.makeHTTPRequest(req, output)
		close(output)
	}()
	return output
}

// makeHTTPRequest makes an HTTP request then awaits and pushes the responses to the output channel
func (c *Connector) makeHTTPRequest(req *pub.Request, output chan *pub.Response) {
	// Set a random message ID
	msgID := rand.AlphaNum64(8)
	frame.Of(req.Header).SetMessageID(msgID)

	// Prepare the HTTP request
	httpReq, err := req.ToHTTP()
	if err != nil {
		output <- pub.NewErrorResponse(errors.Trace(err))
		return
	}

	// Create a channel to await on
	awaitCh := make(chan *http.Response, 64)
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
			output <- pub.NewErrorResponse(errors.Trace(err))
			return
		}
		port = int(port64)
	}
	subject := subjectOfRequest(c.plane, httpReq.URL.Hostname(), port, httpReq.URL.Path)

	var buf bytes.Buffer
	err = httpReq.Write(&buf)
	if err != nil {
		output <- pub.NewErrorResponse(errors.Trace(err))
		return
	}
	err = c.natsConn.Publish(subject, buf.Bytes())
	if err != nil {
		output <- pub.NewErrorResponse(errors.Trace(err))
		return
	}

	// Await and return the responses
	var expectedResponders map[string]bool
	if req.Multicast {
		c.knownRespondersLock.Lock()
		expectedResponders = c.knownResponders[subject]
		c.knownRespondersLock.Unlock()
	}
	countResponses := 0
	seenIDs := map[string]string{} // FromID -> OpCode
	seenQueues := map[string]bool{}
	doneWaitingForAcks := false
	budget := time.Until(req.Deadline)
	timeoutTimer := time.NewTimer(budget)
	defer timeoutTimer.Stop()
	ackTimer := time.NewTimer(c.networkHop)
	defer ackTimer.Stop()
	for {
		select {
		case response := <-awaitCh:
			opCode := frame.Of(response).OpCode()
			fromID := frame.Of(response).FromID()
			queue := frame.Of(response).Queue()

			// Known responders optimization
			if req.Multicast {
				seenQueues[queue] = true
				if !doneWaitingForAcks && len(seenQueues) == len(expectedResponders) {
					match := true
					for k := range seenQueues {
						if !expectedResponders[k] {
							match = false
							break
						}
					}
					if match {
						doneWaitingForAcks = true
					}
				}
			}

			// Ack
			if opCode == frame.OpCodeAck {
				if seenIDs[fromID] == "" {
					seenIDs[fromID] = frame.OpCodeAck
				}
			}

			// Response
			if opCode == frame.OpCodeResponse {
				seenIDs[fromID] = frame.OpCodeResponse
				countResponses++
				output <- pub.NewHTTPResponse(response)
				if !req.Multicast {
					// Return the first result found
					return
				}
				if doneWaitingForAcks && countResponses == len(seenIDs) {
					// All responses have been received
					c.knownRespondersLock.Lock()
					c.knownResponders[subject] = seenQueues
					c.knownRespondersLock.Unlock()
					return
				}
			}

			// Error
			if opCode == frame.OpCodeError {
				// Reconstitute the error if an error op code is returned
				var tracedError *errors.TracedError
				body, err := io.ReadAll(response.Body)
				if err == nil {
					json.Unmarshal(body, &tracedError)
				}
				if tracedError == nil {
					tracedError = errors.New("unparsable error response").(*errors.TracedError)
				}

				seenIDs[fromID] = frame.OpCodeError
				countResponses++
				output <- pub.NewErrorResponse(errors.Trace(tracedError, c.hostName+" -> "+httpReq.URL.Hostname()))
				if !req.Multicast {
					// Return the first result found
					return
				}
				if doneWaitingForAcks && countResponses == len(seenIDs) {
					// All responses have been received
					c.knownRespondersLock.Lock()
					c.knownResponders[subject] = seenQueues
					c.knownRespondersLock.Unlock()
					return
				}
			}

		// Timeout timer
		case <-timeoutTimer.C:
			output <- pub.NewErrorResponse(errors.New("timeout"))
			// Known responders optimization
			if req.Multicast {
				c.knownRespondersLock.Lock()
				delete(c.knownResponders, subject)
				c.knownRespondersLock.Unlock()
			}
			return

		// Ack timer
		case <-ackTimer.C:
			doneWaitingForAcks = true
			if len(seenIDs) == 0 {
				output <- pub.NewErrorResponse(errors.New("ack timeout"))
				// Known responders optimization
				if req.Multicast {
					c.knownRespondersLock.Lock()
					delete(c.knownResponders, subject)
					c.knownRespondersLock.Unlock()
				}
				return
			}
			if countResponses == len(seenIDs) {
				// All responses have been received
				// Known responders optimization
				if req.Multicast {
					c.knownRespondersLock.Lock()
					c.knownResponders[subject] = seenQueues
					c.knownRespondersLock.Unlock()
				}
				return
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
