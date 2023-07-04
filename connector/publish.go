/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/utils"
	"github.com/nats-io/nats.go"
)

const AckTimeout = 250 * time.Millisecond

// GET makes a GET request.
func (c *Connector) GET(ctx context.Context, url string) (*http.Response, error) {
	return c.Request(ctx, []pub.Option{
		pub.GET(url),
	}...)
}

// POST makes a POST request.
// Body of type io.Reader, []byte and string is serialized in binary form.
// url.Values is serialized as form data.
// All other types are serialized as JSON.
func (c *Connector) POST(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.Request(ctx, []pub.Option{
		pub.POST(url),
		pub.Body(body),
	}...)
}

// Request makes an HTTP request then awaits and returns a single response synchronously.
// If no response is received, an ack timeout (404) error is returned.
func (c *Connector) Request(ctx context.Context, options ...pub.Option) (*http.Response, error) {
	options = append(options, pub.Unicast())
	ch := c.Publish(ctx, options...)
	res, err := (<-ch).Get()
	return res, err // No trace
}

// Publish makes an HTTP request then awaits and returns the responses asynchronously.
// By default, publish performs a multicast and multiple responses (or none at all) may be returned.
// Use the Request method or pass in pub.Unicast() to Publish to perform a unicast.
func (c *Connector) Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	errOutput := make(chan *pub.Response, 1)
	defer close(errOutput)

	// Build the request
	req, err := pub.NewRequest(options...)
	if err != nil {
		errOutput <- pub.NewErrorResponse(errors.Trace(err))
		return errOutput
	}

	// Check if there's enough time budget
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.networkHop {
		err = errors.Newc(http.StatusRequestTimeout, "timeout", req.Canonical())
		errOutput <- pub.NewErrorResponse(err)
		return errOutput
	}

	// Limit number of hops
	inboundFrame := frame.Of(ctx)
	outboundFrame := frame.Of(req.Header)
	depth := inboundFrame.CallDepth()
	if depth >= c.maxCallDepth {
		err = errors.Newc(http.StatusLoopDetected, "call depth overflow", req.Canonical())
		errOutput <- pub.NewErrorResponse(err)
		return errOutput
	}
	outboundFrame.SetCallDepth(depth + 1)

	// Set return address
	outboundFrame.SetFromHost(c.hostName)
	outboundFrame.SetFromID(c.id)
	outboundFrame.SetFromVersion(c.version)
	outboundFrame.SetOpCode(frame.OpCodeRequest)

	// Copy X-Forwarded headers (set by ingress proxy)
	for _, fwdHdr := range []string{"X-Forwarded-Host", "X-Forwarded-For", "X-Forwarded-Proto", "X-Forwarded-Prefix"} {
		v := inboundFrame.Get(fwdHdr)
		if v != "" && outboundFrame.Get(fwdHdr) == "" {
			outboundFrame.Set(fwdHdr, v)
		}
	}

	// Copy baggage headers
	for k := range inboundFrame.Header() {
		if strings.HasPrefix(k, frame.HeaderBaggagePrefix) {
			outboundFrame.Set(k, inboundFrame.Get(k))
		}
	}

	// Make the request
	var output *utils.InfiniteChan[*pub.Response]
	if req.Multicast {
		output = utils.MakeInfiniteChan[*pub.Response](c.multicastChanCap)
	} else {
		output = utils.MakeInfiniteChan[*pub.Response](2)
	}
	go func() {
		c.makeHTTPRequest(ctx, req, output)
		fullyDrained := output.Close(time.Second)
		if !fullyDrained {
			c.LogDebug(ctx, "Unconsumed responses dropped", log.String("url", req.Canonical()))
		}
	}()
	return output.C()
}

// makeHTTPRequest makes an HTTP request then awaits and pushes the responses to the output channel.
func (c *Connector) makeHTTPRequest(ctx context.Context, req *pub.Request, output *utils.InfiniteChan[*pub.Response]) {
	// Set a random message ID
	msgID := rand.AlphaNum64(8)
	frame.Of(req.Header).SetMessageID(msgID)

	// Prepare the HTTP request (first fragment only)
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		err = errors.Trace(err, req.Canonical())
		output.Push(pub.NewErrorResponse(err))
		return
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	deadline, deadlineOK := ctx.Deadline()
	if deadlineOK {
		frame.Of(httpReq).SetTimeBudget(time.Until(deadline))
	}

	c.LogDebug(ctx, "Request", log.String("msg", msgID), log.String("url", req.Canonical()))

	// Fragment large requests
	fragger, err := httpx.NewFragRequest(httpReq, c.maxFragmentSize)
	if err != nil {
		err = errors.Trace(err, req.Canonical())
		output.Push(pub.NewErrorResponse(err))
		return
	}
	httpReq, err = fragger.Fragment(1)
	if err != nil {
		err = errors.Trace(err, req.Canonical())
		output.Push(pub.NewErrorResponse(err))
		return
	}

	// Create a channel to await on
	awaitCh := utils.MakeInfiniteChan[*http.Response](c.multicastChanCap)
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
			err = errors.Trace(err, req.Canonical())
			output.Push(pub.NewErrorResponse(err))
			return
		}
		port = int(port64)
	}
	subject := subjectOfRequest(c.plane, httpReq.URL.Hostname(), port, httpReq.URL.Path)

	var buf bytes.Buffer
	err = httpReq.Write(&buf)
	if err != nil {
		err = errors.Trace(err, req.Canonical())
		output.Push(pub.NewErrorResponse(err))
		return
	}

	publishTime := time.Now()
	err = c.natsConn.Publish(subject, buf.Bytes())
	if err != nil {
		err = errors.Trace(err, req.Canonical())
		output.Push(pub.NewErrorResponse(err))
		return
	}

	enumResponders := func(responders map[string]bool) string {
		var b strings.Builder
		for k := range responders {
			if b.Len() != 0 {
				b.WriteString(", ")
			}
			b.WriteString(k)
		}
		return b.String()
	}

	// Await and return the responses
	var expectedResponders map[string]bool
	if req.Multicast {
		expectedResponders, _ = c.knownResponders.Load(subject)
		if len(expectedResponders) > 0 {
			c.LogDebug(ctx, "Expecting responders", log.String("msg", msgID), log.String("subject", subject), log.String("responders", enumResponders(expectedResponders)))
		}
		c.postRequestData.Store("multicast:"+msgID, subject)
	}
	countResponses := 0
	seenIDs := map[string]string{} // FromID -> OpCode
	seenQueues := map[string]bool{}
	doneWaitingForAcks := false
	var timeoutTimer *time.Timer
	if deadlineOK {
		timeoutTimer = time.NewTimer(time.Until(deadline))
		defer timeoutTimer.Stop()
	} else {
		// No op timer
		timeoutTimer = &time.Timer{
			C: make(<-chan time.Time),
		}
	}
	ackTimer := time.NewTimer(AckTimeout)
	defer ackTimer.Stop()
	ackTimerStart := time.Now()
	for {
		select {
		case response := <-awaitCh.C():
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
					_ = c.ObserveMetric(
						"microbus_ack_duration_seconds",
						time.Since(publishTime).Seconds(),
						httpReq.URL.Hostname(),
					)
					seenIDs[fromID] = frame.OpCodeAck
				}

				// Send additional fragments (if there are any) in a goroutine
				if fragger.N() > 1 {
					go func() {
						for f := 2; f <= fragger.N(); f++ {
							fragment, err := fragger.Fragment(f)
							if err != nil {
								err = errors.Trace(err)
								c.LogError(ctx, "Sending fragments", log.Error(err), log.String("url", req.Canonical()))
								break
							}

							// Direct addressing
							subject := subjectOfRequest(c.plane, fromID+"."+fragment.URL.Hostname(), port, fragment.URL.Path)

							var buf bytes.Buffer
							err = fragment.Write(&buf)
							if err != nil {
								err = errors.Trace(err)
								c.LogError(ctx, "Sending fragments", log.Error(err), log.String("url", req.Canonical()))
								break
							}
							err = c.natsConn.Publish(subject, buf.Bytes())
							if err != nil {
								err = errors.Trace(err)
								c.LogError(ctx, "Sending fragments", log.Error(err), log.String("url", req.Canonical()))
								break
							}
						}
					}()
				}
			}

			// Response
			if opCode == frame.OpCodeResponse {
				output.Push(pub.NewHTTPResponse(response))
				_ = c.IncrementMetric(
					"microbus_request_count_total",
					1,
					httpReq.Method,
					httpReq.URL.Hostname(),
					strconv.Itoa(port),
					strconv.Itoa(response.StatusCode),
					"OK",
				)
			}

			// Error
			if opCode == frame.OpCodeError {
				// Reconstitute the error
				var reconstitutedError *errors.TracedError
				body, err := io.ReadAll(response.Body)
				if err == nil {
					json.Unmarshal(body, &reconstitutedError)
				}
				if reconstitutedError == nil {
					err = errors.New("unparsable error response")
				} else {
					err = errors.Convert(reconstitutedError)
				}
				// err = errors.Convert(err).Annotate(c.hostName + " -> " + httpReq.URL.Host + httpReq.URL.Path)
				output.Push(pub.NewErrorResponse(err))
				statusCode := reconstitutedError.StatusCode
				if statusCode == 0 {
					statusCode = http.StatusInternalServerError
				}
				_ = c.IncrementMetric(
					"microbus_request_count_total",
					1,
					httpReq.Method,
					httpReq.URL.Hostname(),
					strconv.Itoa(port),
					strconv.Itoa(statusCode),
					func() string {
						if statusCode == http.StatusNotFound {
							return "OK"
						}
						return "ERROR"
					}(),
				)
			}

			// Response or error (i.e. not an ack)
			if opCode == frame.OpCodeResponse || opCode == frame.OpCodeError {
				if !req.Multicast {
					// Return the first result found immediately
					return
				}
				seenIDs[fromID] = opCode
				countResponses++
				if doneWaitingForAcks && countResponses == len(seenIDs) {
					// All responses have been received
					// Known responders optimization
					c.knownResponders.Store(subject, seenQueues)
					c.LogDebug(ctx, "Caching responders", log.String("msg", msgID), log.String("subject", subject), log.String("responders", enumResponders(seenQueues)))
					return
				}
			}

		// Timeout timer
		case <-timeoutTimer.C:
			c.LogDebug(ctx, "Request timeout", log.String("msg", msgID), log.String("subject", subject))
			err = errors.Newc(http.StatusRequestTimeout, "timeout", req.Canonical())
			output.Push(pub.NewErrorResponse(err))
			c.postRequestData.Store("timeout:"+msgID, subject)
			_ = c.IncrementMetric(
				"microbus_request_count_total",
				1,
				httpReq.Method,
				httpReq.URL.Hostname(),
				strconv.Itoa(port),
				strconv.Itoa(http.StatusRequestTimeout),
				"OK",
			)

			// Known responders optimization
			if req.Multicast {
				c.knownResponders.Delete(subject)
				c.LogDebug(ctx, "Clearing responders", log.String("msg", msgID), log.String("subject", subject))
			}
			return

		// Ack timer
		case <-ackTimer.C:
			if c.deployment == LOCAL && time.Since(ackTimerStart) >= AckTimeout*8 {
				// Likely resuming from a breakpoint that prevented the ack from arriving in time.
				// Reset the ack timer to allow the ack to arrive.
				ackTimer.Reset(AckTimeout)
				ackTimerStart = time.Now()
				c.LogDebug(ctx, "Resetting ack timeout", log.String("msg", msgID), log.String("subject", subject))
				continue
			}
			doneWaitingForAcks = true
			if len(seenIDs) == 0 {
				if req.Multicast {
					// Known responders optimization
					c.knownResponders.Delete(subject)
					c.LogDebug(ctx, "Clearing responders", log.String("msg", msgID), log.String("subject", subject))
				} else {
					err = errors.Newc(http.StatusNotFound, "ack timeout", req.Canonical())
					output.Push(pub.NewErrorResponse(err))
					_ = c.IncrementMetric(
						"microbus_request_count_total",
						1,
						httpReq.Method,
						httpReq.URL.Hostname(),
						strconv.Itoa(port),
						strconv.Itoa(http.StatusNotFound),
						"OK",
					)
				}
				return
			}
			if countResponses == len(seenIDs) {
				// All responses have been received
				// Known responders optimization
				if req.Multicast {
					c.knownResponders.Store(subject, seenQueues)
					c.LogDebug(ctx, "Caching responders", log.String("msg", msgID), log.String("subject", subject), log.String("responders", enumResponders(seenQueues)))
				}
				return
			}
		}
	}
}

// onResponse is called when a response to an outgoing request is received.
func (c *Connector) onResponse(msg *nats.Msg) {
	// Parse the response
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(msg.Data)), nil)
	if err != nil {
		err = errors.Trace(err)
		c.LogError(c.lifetimeCtx, "Parsing response", log.Error(err))
		return
	}

	// Integrate fragments together
	response, err = c.defragResponse(response)
	if err != nil {
		err = errors.Trace(err)
		c.LogError(c.lifetimeCtx, "Defragging response", log.Error(err))
		return
	}
	if response == nil {
		// Not all fragments arrived yet
		return
	}

	// Push it to the channel matching the message ID
	msgID := frame.Of(response).MessageID()
	c.reqsLock.Lock()
	ch, ok := c.reqs[msgID]
	c.reqsLock.Unlock()
	if ok {
		ch.Push(response)
		return
	}

	// Handle message that arrive after the request is done
	opCode := frame.Of(response).OpCode()
	if opCode != frame.OpCodeAck {
		subject, ok := c.postRequestData.Load("multicast:"+msgID, lru.NoBump())
		if ok {
			c.knownResponders.Delete(subject)
			c.postRequestData.Delete("multicast:" + msgID)
		}
		subject, ok = c.postRequestData.Load("timeout:"+msgID, lru.NoBump())
		if ok {
			c.LogInfo(
				c.lifetimeCtx,
				"Response received after timeout",
				log.String("msg", msgID),
				log.String("fromID", frame.Of(response).FromID()),
				log.String("fromHost", frame.Of(response).FromHost()),
				log.String("queue", frame.Of(response).Queue()),
				log.String("subject", subject),
			)
		}
	}
}
