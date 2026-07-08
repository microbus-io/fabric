/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/transport"
	"github.com/microbus-io/fabric/utils"
	"go.opentelemetry.io/otel/propagation"
)

// transferChan is intermediating between the publisher and the responses it receives.
type transferChan struct {
	C    chan *http.Response
	Done chan bool
}

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
	for qi := range c.Publish(ctx, options...) {
		return qi.Get() // No trace
	}
	return nil, errors.New("no response")
}

// Publish makes an HTTP request then awaits and returns the responses asynchronously.
// By default, publish performs a multicast and multiple responses (or none at all) may be returned.
// Use the Request method or pass in pub.Unicast() to Publish to perform a unicast.
func (c *Connector) Publish(ctx context.Context, options ...pub.Option) iter.Seq[*pub.Response] {
	// Check if there's enough time budget
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.networkRoundtrip {
		err := errors.New("timeout", http.StatusRequestTimeout, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	// Limit number of hops
	inboundFrame := frame.Of(ctx)
	depth := inboundFrame.CallDepth()
	if depth >= c.maxCallDepth {
		err := errors.New("call depth overflow", http.StatusLoopDetected, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	// Build the request
	req, err := pub.NewRequest()
	if err != nil {
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}
	outboundFrame := frame.Of(req.Header)

	// Copy X-Forwarded headers (set by ingress proxy), baggage, actor, and Accept-Language headers
	for k, vv := range inboundFrame.Header() {
		if strings.HasPrefix(k, "X-Forwarded-") ||
			strings.HasPrefix(k, frame.HeaderBaggagePrefix) ||
			k == "Accept-Language" ||
			k == frame.HeaderActor {
			for _, v := range vv {
				outboundFrame.Header().Add(k, v)
			}
		}
	}

	// Options can override anything above
	err = req.Apply(options...)
	if err != nil {
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	// Set depth
	outboundFrame.SetCallDepth(depth + 1)

	// Set return address
	outboundFrame.SetFromHost(c.hostname)
	outboundFrame.SetFromID(c.id)
	outboundFrame.SetFromVersion(c.version)
	outboundFrame.SetOpCode(frame.OpCodeRequest)

	// OpenTelemetry: pass the span in headers
	carrier := make(propagation.HeaderCarrier)
	propagation.TraceContext{}.Inject(ctx, carrier)
	for k, v := range carrier {
		outboundFrame.Set(k, v[0])
	}

	// Locality-aware routing.
	optimizeLocality := !req.Multicast && c.locality != ""
	origURL := req.URL
	localityCacheKey := ""
	lastKnownLocality := ""
	if optimizeLocality {
		localityCacheKey, _, _ = strings.Cut(origURL, "?")
		lastKnownLocality, _ = c.localResponder.Load(localityCacheKey, lru.Bump(true))
		if lastKnownLocality != "" {
			// Adjust the hostname to include the best known locality,
			// e.g. example.com -> loc-us-west.example.com
			before, after, _ := strings.Cut(origURL, "://")
			req.URL = before + "://" + escapeLocality(lastKnownLocality) + "." + after
		}
	}

	// Make the request
	queue := c.makeRequest(ctx, req)

	// Locality-aware routing
	if optimizeLocality {
		firstResponse := func(q iter.Seq[*pub.Response]) (rr *pub.Response) {
			q(func(r *pub.Response) bool {
				rr = r
				return false
			})
			return rr
		}
		res, err := firstResponse(queue).Get()
		if lastKnownLocality != "" && errors.StatusCode(err) == http.StatusNotFound {
			// No response from the localized URL so retry at the original URL
			c.localResponder.Delete(localityCacheKey)
			lastKnownLocality = ""
			req.URL = origURL
			queue = c.makeRequest(ctx, req)
			res, _ = firstResponse(queue).Get()
		}
		responseLocality := frame.Of(res).Locality()
		if len(responseLocality) > len(lastKnownLocality) {
			_, after, _ := strings.Cut(origURL, "://")
			if !strings.HasPrefix(after, frame.Of(res).FromID()+".") { // Do not optimize when addressing a service by its ID
				longestCommonPrefix := ""
				parts := strings.Split(responseLocality, "-")
				for i := 1; i <= len(parts); i++ {
					l := strings.Join(parts[:i], "-")
					if c.locality == l || strings.HasPrefix(c.locality, l+"-") {
						longestCommonPrefix = l
					} else {
						break
					}
				}
				if len(longestCommonPrefix) > len(lastKnownLocality) {
					c.localResponder.Store(localityCacheKey, longestCommonPrefix)
				}
			}
		}
	}

	// Return the iterator
	return queue
}

// makeRequest makes an HTTP request over NATS, then awaits and pushes the responses to the output channel.
func (c *Connector) makeRequest(ctx context.Context, req *pub.Request) iter.Seq[*pub.Response] {
	// Prepare the HTTP request (first fragment only)
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}
	if req.Body == nil {
		req.Body = http.NoBody
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	httpReq.ContentLength = int64(req.ContentLength)
	// Stop the http package from setting Go-http-client/1.1 as the user-agent
	if len(httpReq.Header.Values("User-Agent")) == 0 {
		httpReq.Header.Set("User-Agent", "")
	}
	var timeout time.Duration
	deadline, deadlineOK := ctx.Deadline()
	if deadlineOK {
		timeout = time.Until(deadline)
	}
	if req.Timeout > 0 && (timeout == 0 || req.Timeout < timeout) {
		timeout = req.Timeout
	}
	if timeout <= 0 {
		timeout = c.defaultTimeBudget
	}
	timeout = min(timeout, c.maxTimeBudget)
	frame.Of(httpReq).SetTimeBudget(timeout)

	// Fragment large requests
	fragger, err := httpx.NewFragRequest(httpReq, c.maxFragmentSize)
	if err != nil {
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}
	httpReq, err = fragger.Fragment(1)
	if err != nil {
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	// Create a channel to await on
	awaitCh := &transferChan{
		C:    make(chan *http.Response, c.multicastChanCap),
		Done: make(chan bool),
	}
	msgID := ""
	for {
		msgID = utils.RandomIdentifier(8)               // 2.8e+14
		_, exists := c.reqs.LoadOrStore(msgID, awaitCh) // Avoid hash clash because it has severe repercussions
		if !exists {
			break
		}
	}
	// Must close the await chan after all responses are counted for
	releaseAwaitCh := func() {
		c.reqs.Delete(msgID)
		close(awaitCh.Done)
	}

	// Send the message
	port := "443"
	if httpReq.URL.Scheme == "http" {
		port = "80"
	}
	if httpReq.URL.Port() != "" {
		port = httpReq.URL.Port()
	}
	host := httpReq.URL.Hostname()
	host, idOrLocality := cutIDOrLocality(host)
	subject := SubjectOfRequest(c.plane, port, c.hostname, host, idOrLocality, httpReq.Method, httpReq.URL.Path)
	if len(subject) > maxSubjectLength {
		releaseAwaitCh()
		err := errors.New("subject too long", http.StatusRequestURITooLong, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	frame.Of(httpReq).SetMessageID(msgID)

	c.LogDebug(ctx, "Request",
		"msg", msgID,
		"url", req.Canonical(),
		"method", req.Method,
	)

	publishTime := time.Now()
	if req.Multicast {
		err = c.transportConn.Publish(subject, httpReq)
	} else {
		err = c.transportConn.Request(subject, httpReq)
	}
	if err != nil {
		releaseAwaitCh()
		err = errors.Trace(err, c.Span(ctx).TraceID())
		return pub.NewSoloResponseQueue(pub.NewErrorResponse(err))
	}

	// Await and return the responses
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

	var expectedResponders map[string]bool
	if req.Multicast {
		// Known responders optimization
		expectedResponders, _ = c.knownResponders.Load(subject, lru.Bump(true))
		if len(expectedResponders) > 0 {
			c.LogDebug(ctx, "Expecting responders",
				"msg", msgID,
				"subject", subject,
				"responders", enumResponders(expectedResponders),
			)
		}
		c.postRequestData.Store("multicast:"+msgID, subject)
	}

	// Wait for the responses in a separate goroutine
	var output *pub.ResponseQueue
	var soloResponse *pub.Response
	if req.Multicast {
		output = pub.NewResponseQueue(c.multicastChanCap)
	}
	awaitResponses := func() {
		defer func() {
			releaseAwaitCh()
			if output != nil {
				output.Close()
			}
		}()
		countResponses := 0
		seenIDs := map[string]string{} // FromID -> OpCode
		seenQueues := map[string]bool{}
		doneWaitingForAcks := false
		var timeoutTimer *time.Timer
		if timeout > 0 {
			timeoutTimer = time.NewTimer(timeout)
			defer timeoutTimer.Stop()
		} else {
			// No op timer
			timeoutTimer = &time.Timer{
				C: make(<-chan time.Time),
			}
		}

		ackTimer := time.NewTimer(c.ackTimeout)
		defer ackTimer.Stop()
		ackTimerStart := time.Now()
		fragmentsSent := map[string]bool{}
		for {
			select {
			case response := <-awaitCh.C:
				opCode := frame.Of(response).OpCode()
				fromID := frame.Of(response).FromID()
				queue := frame.Of(response).Queue()
				if queue == "" {
					queue = fromID + "." + frame.Of(response).FromHost()
				}

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
						_ = c.RecordHistogram(
							ctx,
							"microbus_client_ack_roundtrip_latency_seconds",
							time.Since(publishTime).Seconds(),
							"host", host,
							"port", port,
						)
						seenIDs[fromID] = frame.OpCodeAck
					}

					// Send additional fragments (if there are any) to all those who ack'ed
					if fragger.N() > 1 && !fragmentsSent[fromID] {
						fragmentsSent[fromID] = true
						go func() {
							for f := 2; f <= fragger.N(); f++ {
								fragment, err := fragger.Fragment(f)
								if err != nil {
									err = errors.Trace(err)
									c.LogError(ctx, "Sending fragments",
										"error", err,
										"url", req.Canonical(),
										"method", req.Method,
									)
									break
								}

								// Direct addressing - pin subsequent fragments to the exact replica that ack'd the first fragment.
								fragmentHost, _ := cutIDOrLocality(fragment.URL.Hostname())
								subject := SubjectOfRequest(c.plane, port, c.hostname, fragmentHost, fromID, fragment.Method, fragment.URL.Path)

								frame.Of(fragment).SetMessageID(msgID)
								if req.Multicast {
									err = c.transportConn.Publish(subject, fragment)
								} else {
									err = c.transportConn.Request(subject, fragment)
								}
								if err != nil {
									err = errors.Trace(err)
									c.LogError(ctx, "Sending fragments",
										"error", err,
										"url", req.Canonical(),
										"method", req.Method,
									)
									break
								}
							}
						}()
					}
				}

				// Response
				if opCode == frame.OpCodeResponse {
					soloResponse = pub.NewHTTPResponse(response)
					if output != nil {
						output.Push(soloResponse)
					}
				}

				// Error
				if opCode == frame.OpCodeError {
					// Reconstitute the error
					var reconstitutedError struct {
						Err *errors.TracedError `json:"err"`
					}
					err = json.NewDecoder(response.Body).Decode(&reconstitutedError)
					if err != nil || reconstitutedError.Err == nil {
						err = errors.New("unparsable error response: %s", req.Canonical(), c.Span(ctx).TraceID())
					} else {
						reconstitutedError.Err.Trace = c.Span(ctx).TraceID()
						err = errors.Convert(reconstitutedError.Err)
					}
					if reconstitutedError.Err.StatusCode == 0 {
						reconstitutedError.Err.StatusCode = http.StatusInternalServerError
					}
					soloResponse = pub.NewErrorResponse(err)
					if output != nil {
						output.Push(soloResponse)
					}
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
						c.LogDebug(ctx, "Caching responders",
							"msg", msgID,
							"subject", subject,
							"responders", enumResponders(seenQueues),
						)
						return
					}
				}

			// Timeout timer
			case <-timeoutTimer.C:
				c.LogDebug(ctx, "Request timeout",
					"msg", msgID,
					"subject", subject,
				)
				err = errors.New(
					"timeout: %s", req.Canonical(),
					http.StatusRequestTimeout,
					c.Span(ctx).TraceID(),
				)
				soloResponse = pub.NewErrorResponse(err)
				if output != nil {
					output.Push(soloResponse)
				}
				c.postRequestData.Store("timeout:"+msgID, subject)
				_ = c.IncrementCounter(
					ctx,
					"microbus_client_timeout_requests",
					1,
					"code", http.StatusRequestTimeout,
				)

				// Known responders optimization
				if req.Multicast {
					c.knownResponders.Delete(subject)
					c.LogDebug(ctx, "Clearing responders",
						"msg", msgID,
						"subject", subject,
					)
				}
				return

			// Ack timer
			case <-ackTimer.C:
				if c.deployment == LOCAL && time.Since(ackTimerStart) >= c.ackTimeout*8 {
					// Likely resuming from a breakpoint that prevented the ack from arriving in time.
					// Reset the ack timer to allow the ack to arrive.
					ackTimer.Reset(c.ackTimeout)
					ackTimerStart = time.Now()
					c.LogDebug(ctx, "Resetting ack timeout",
						"msg", msgID,
						"subject", subject,
					)
					continue
				}
				doneWaitingForAcks = true
				if len(seenIDs) == 0 {
					if req.Multicast {
						// Known responders optimization
						c.knownResponders.Delete(subject)
						c.LogDebug(ctx, "Clearing responders",
							"msg", msgID,
							"subject", subject,
						)
					} else {
						err = errors.New(
							"ack timeout: %s", req.Canonical(),
							http.StatusNotFound,
							c.Span(ctx).TraceID(),
						)
						soloResponse = pub.NewErrorResponse(err)
						if output != nil {
							output.Push(soloResponse)
						}
						_ = c.IncrementCounter(
							ctx,
							"microbus_client_timeout_requests",
							1,
							"code", http.StatusNotFound,
						)
					}
					return
				}
				if countResponses == len(seenIDs) {
					// All responses have been received
					// Known responders optimization
					if req.Multicast {
						c.knownResponders.Store(subject, seenQueues)
						c.LogDebug(ctx, "Caching responders",
							"msg", msgID,
							"subject", subject,
							"responders", enumResponders(seenQueues),
						)
					}
					return
				}
			}
		}
	}

	if output != nil {
		// Return the output queue instantly as a future
		// It will be closed by the goroutine when all responses come in
		go awaitResponses()
		return output.Q()
	} else {
		// Return the single response after it was received
		awaitResponses()
		return pub.NewSoloResponseQueue(soloResponse)
	}
}

// onResponse is called when a response to an outgoing request is received.
func (c *Connector) onResponse(msg *transport.Msg) {
	err := c.handleResponse(msg)
	if err != nil {
		err = errors.Trace(err)
		c.LogError(c.Lifetime(), "Handling response", "error", err)
	}
}

// onResponse is called when a response to an outgoing request is received.
func (c *Connector) handleResponse(msg *transport.Msg) error {
	var err error
	response := msg.Response
	if response == nil {
		// Parse the response
		response, err = http.ReadResponse(bufio.NewReaderSize(bytes.NewReader(msg.Data), 64), nil)
		if err != nil {
			return errors.Trace(err)
		}
	}
	if br, ok := response.Body.(*httpx.BodyReader); ok {
		br.Reset()
	}

	// Overwrite From-Host with the verified source from the subject.
	_, _, _, src, _, _ := splitSubject(msg.Subject)
	frame.Of(response).SetFromHost(src)

	// Integrate fragments together
	response, err = c.defragResponse(response)
	if err != nil {
		return errors.Trace(err)
	}
	if response == nil {
		// Not all fragments arrived yet
		return nil
	}

	// Push it to the channel matching the message ID
	msgID := frame.Of(response).MessageID()
	ch, ok := c.reqs.Load(msgID)
	if !ok {
		return nil
	}

	// Try to push inline for when the channel has enough capacity
	select {
	case ch.C <- response:
		return nil
	default:
	}
	// If not enough capacity, spin up a non-blocking goroutine to push the message
	// More messages can block, so need to listen to the Done channel
	go func() {
		select {
		case ch.C <- response:
			return
		case <-ch.Done:
			// Handle message that arrive after the request is done.
			frm := frame.Of(response)
			opCode := frm.OpCode()
			if opCode != frame.OpCodeAck {
				subject, ok := c.postRequestData.Load("multicast:"+msgID, lru.NoBump())
				if ok {
					c.knownResponders.Delete(subject)
					c.postRequestData.Delete("multicast:" + msgID)
				}
				subject, ok = c.postRequestData.Load("timeout:"+msgID, lru.NoBump())
				if ok {
					c.LogInfo(c.Lifetime(), "Response received after timeout",
						"msg", msgID,
						"fromID", frm.FromID(),
						"fromHost", frm.FromHost(),
						"queue", frm.Queue(),
						"subject", subject,
					)
				}
			}
		}
	}()
	return nil
}
