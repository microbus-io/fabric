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
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/boolexp"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/transport"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"

	"go.opentelemetry.io/otel/propagation"
)

// HTTPHandler extends the standard http.Handler to also return an error
type HTTPHandler = sub.HTTPHandler

// SubscriptionInfo is a read-only snapshot of one subscription as it appears in the connector.
type SubscriptionInfo struct {
	Name           string
	Description    string
	Host           string
	Port           string
	Method         string
	Path           string
	Queue          string
	Type           string
	RequiredClaims string
	Tags           []string
	Manual         bool
	NoTrace        bool
	Active         bool
}

/*
Listen subscribes a handler under a typed name. The name must be a Go-style upper-case
identifier (e.g. "MyEndpoint2") and unique within the connector. Exactly one feature option
([sub.Function], [sub.Web], [sub.InboundEvent], [sub.Task], [sub.Workflow]) must be supplied.

Defaults filled in after options are applied:
  - Method defaults to "ANY".
  - Route defaults to ":443/my-subscription" with the port determined by the feature type
    (443 for function/web, 417 for inbound events, 428 for tasks/graphs).
  - Queue defaults to the connector's hostname.

Returns an error if the name is invalid, already registered, or if the option set is malformed.
*/
func (c *Connector) Subscribe(name string, handler sub.HTTPHandler, options ...sub.Option) error {
	if c.hostname == "" {
		return c.captureInitErr(errors.New("hostname is not set"))
	}
	newSub, err := sub.NewSubscription(name, c.hostname, handler, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	if _, loaded := c.subs.LoadOrStore(name, newSub); loaded {
		return c.captureInitErr(errors.New("duplicate subscription name '%s'", name))
	}
	if c.isPhase(startedUp) && !newSub.Manual {
		if err := c.activateSub(newSub); err != nil {
			c.subs.Delete(name)
			return c.captureInitErr(errors.Trace(err))
		}
		c.notifyOnNewSubs(newSub)
		c.transportConn.WaitForSub()
	}
	return nil
}

// Subscriptions returns a read-only snapshot of every subscription registered with the connector,
// in registration order. Callers iterate the result and act on names of interest via
// [Connector.ActivateSubscription] and [Connector.DeactivateSubscription]. Filter by any
// combination of fields ([SubscriptionInfo.Tags], [SubscriptionInfo.Type], etc.) to express
// "all Python tasks" or "all manual subs" without a custom query API.
func (c *Connector) Subscriptions() []SubscriptionInfo {
	values := c.subs.Values()
	out := make([]SubscriptionInfo, 0, len(values))
	for _, s := range values {
		var tags []string
		if len(s.Tags) > 0 {
			tags = append([]string(nil), s.Tags...)
		}
		out = append(out, SubscriptionInfo{
			Name:           s.Name,
			Description:    s.Description,
			Host:           s.Host,
			Port:           s.Port,
			Method:         s.Method,
			Path:           s.Path,
			Queue:          s.Queue,
			Type:           s.Type,
			RequiredClaims: s.RequiredClaims,
			Tags:           tags,
			Manual:         s.Manual,
			NoTrace:        s.NoTrace,
			Active:         len(s.Subs) > 0,
		})
	}
	return out
}

// ActivateSubscription activates the named subscription if it is currently off-bus. The
// subscription joins the transport and a peer notification is broadcast so other microservices
// invalidate their cached known-responders entries. Already-active subscriptions are silently
// skipped, so the call is idempotent. Valid while the connector is starting up, started, or
// shutting down - this lets [sub.Manual] subscriptions come on/off bus alongside the lifecycle
// of their backing resource (e.g. a Python venv allocated inside OnStartup, or a distributed
// cache torn down inside OnShutdown). Returns an error if no subscription with that name is
// registered.
func (c *Connector) ActivateSubscription(name string) error {
	if !c.isPhase(startingUp, startedUp, shuttingDown) {
		return errors.New("not started")
	}
	s, ok := c.subs.Load(name)
	if !ok {
		return errors.New("unknown subscription name '%s'", name)
	}
	if len(s.Subs) > 0 {
		return nil
	}
	if err := c.activateSub(s); err != nil {
		return errors.Trace(err)
	}
	if c.isPhase(startedUp) {
		c.notifyOnNewSubs(s)
		c.transportConn.WaitForSub()
	}
	return nil
}

// DeactivateSubscription takes the named subscription off the bus while leaving it registered
// in the connector, so a subsequent [Connector.ActivateSubscription] brings it back online.
// Unicast callers see a clean 404 ack-timeout while it is deactivated; load-balancing routes
// around the cold replica. Already-deactivated subscriptions are silently skipped, so the call
// is idempotent. Valid while the connector is starting up, started, or shutting down. Returns
// an error if no subscription with that name is registered.
func (c *Connector) DeactivateSubscription(name string) error {
	if !c.isPhase(startingUp, startedUp, shuttingDown) {
		return errors.New("not started")
	}
	s, ok := c.subs.Load(name)
	if !ok {
		return errors.New("unknown subscription name '%s'", name)
	}
	if len(s.Subs) == 0 {
		return nil
	}
	if err := c.deactivateSub(s); err != nil {
		return errors.Trace(err)
	}
	if c.isPhase(startedUp) {
		c.transportConn.WaitForSub()
	}
	return nil
}

// Unsubscribe removes the subscription registered with [Connector.Subscribe] under the given name.
// Returns an error if no subscription with that name exists.
func (c *Connector) Unsubscribe(name string) error {
	s, ok := c.subs.Delete(name)
	if !ok {
		return errors.New("unknown subscription name '%s'", name)
	}
	if err := c.deactivateSub(s); err != nil {
		return errors.Trace(err)
	}
	if c.isPhase(startedUp) {
		c.transportConn.WaitForSub()
	}
	return nil
}

// validRequestMethods is the set of HTTP method tokens accepted on inbound wire-level
// requests. Per RFC 9110 §9. The framework's "ANY" wildcard is deliberately excluded -
// it is a subscription-side match-anything sentinel that should never appear on the wire.
var validRequestMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
	http.MethodPatch:   true,
}

// onRequest handles an incoming request. It acks it, then calls the handler to process it and responds to the caller.
func (c *Connector) onRequest(msg *transport.Msg, s *sub.Subscription) {
	c.pendingOps.Add(1)

	if msg.Request == nil {
		// Parse the request
		httpReq, err := http.ReadRequest(bufio.NewReaderSize(bytes.NewReader(msg.Data), 64))
		if err != nil {
			c.pendingOps.Add(-1)
			err = errors.Trace(err)
			c.LogError(c.Lifetime(), "Parsing request", "error", err)
			return
		}
		msg.Request = httpReq
	}
	if !validRequestMethods[msg.Request.Method] {
		c.pendingOps.Add(-1)
		err := errors.New("invalid method",
			http.StatusMethodNotAllowed,
			"method", msg.Request.Method,
			"url", msg.Request.URL.String(),
		)
		c.LogError(c.Lifetime(), "Rejecting request", "error", err)
		return
	}
	if msg.Request.Body == nil {
		msg.Request.Body = http.NoBody
	}

	// Overwrite From-Host with the verified source from the subject.
	_, _, _, src, _, _ := splitSubject(msg.Subject)
	frame.Of(msg.Request).SetFromHost(src)

	err := c.ackRequest(msg, s)
	if err != nil {
		c.pendingOps.Add(-1)
		err = errors.Trace(err)
		c.LogError(c.Lifetime(), "Acking request", "error", err)
		return
	}
	go func() {
		defer c.pendingOps.Add(-1)
		err := c.handleRequest(msg, s)
		if err != nil {
			err = errors.Trace(err)
			c.LogError(c.Lifetime(), "Handling request", "error", err)
		}
	}()
}

// activateSubs subscribes all non-[sub.Manual] subscriptions with the transport.
// It is called during startup after the user's OnStartup callback returns. Manual
// subscriptions stay off-bus and are brought online by user code (or by a framework
// activate-by-tag pass) once their backing resource is ready.
//
// In TESTING deployment the [sub.Manual] flag is ignored and every registered
// subscription is activated. Tests typically replace the backing resource with a
// mock, so the gating that real OnStartup code would do (wait for the resource,
// then call [Connector.ActivateSubscription]) doesn't run - activating manual
// subs here keeps mocked endpoints reachable on the bus without per-test setup.
func (c *Connector) activateSubs() (err error) {
	subs := c.subs.Values()
	var activated []*sub.Subscription
	for _, s := range subs {
		if s.Manual && c.deployment != TESTING {
			continue
		}
		if err = c.activateSub(s); err != nil {
			return errors.Trace(err)
		}
		activated = append(activated, s)
	}
	if c.isPhase(startingUp) && len(activated) > 0 {
		c.notifyOnNewSubs(activated...)
	}
	c.transportConn.WaitForSub()
	return nil
}

// activateSub subscribes a single subscription with the transport.
func (c *Connector) activateSub(s *sub.Subscription) (err error) {
	if len(s.Subs) > 0 {
		return nil
	}
	// Recalculate the subscription path in case the hostname changed since it was created
	err = s.RefreshHostname(c.hostname)
	if err != nil {
		return errors.Trace(err)
	}
	// Create the subscriptions
	handler := func(msg *transport.Msg) {
		c.onRequest(msg, s)
	}
	prefixes := []string{
		"", // bare - encoded as the `_` placeholder
		c.id,
	}
	if c.locality != "" {
		loc := strings.Split(c.locality, "-")
		for i := 1; i <= len(loc); i++ {
			prefixes = append(prefixes, escapeLocality(strings.Join(loc[:i], "-")))
		}
	}
	for _, prefix := range prefixes {
		subject := SubjectOfRequestSub(c.plane, s.Port, s.Host, prefix, s.Method, s.Path)
		if len(subject) > maxSubjectLength {
			err = errors.New("subject too long", http.StatusRequestURITooLong)
			break
		}
		var transportSub *transport.Subscription
		if s.Queue != "" {
			transportSub, err = c.transportConn.QueueSubscribe(subject, s.Queue, handler)
		} else {
			transportSub, err = c.transportConn.Subscribe(subject, handler)
		}
		if err != nil {
			break
		}
		s.Subs = append(s.Subs, transportSub)
	}
	if err != nil {
		c.LogError(c.Lifetime(), "Activating sub",
			"error", err,
			"url", s.Canonical(),
			"method", s.Method,
		)
		c.deactivateSub(s)
		return errors.Trace(err)
	}
	// c.LogDebug(c.Lifetime(), "Sub activated",
	// 	"url", s.Canonical(),
	// 	"method", req.Method,
	// )
	return nil
}

// deactivateSubs unsubscribes all subscriptions with the transport.
// It is called during shutdown.
func (c *Connector) deactivateSubs() error {
	var lastErr error
	for _, sub := range c.subs.Values() {
		err := c.deactivateSub(sub)
		if err != nil {
			lastErr = errors.Trace(err)
		}
	}
	c.transportConn.WaitForSub()
	return lastErr // No trace
}

// deactivateAutoSubs unsubscribes all non-[sub.Manual] subscriptions. It is called during
// shutdown before OnShutdown runs, so user code stops receiving routine traffic while any
// manual subscriptions (the distributed cache, Python venv handlers, etc.) remain reachable
// to OnShutdown. Manual subscriptions are the caller's responsibility - either user code
// deactivates them inside OnShutdown, or the connector deactivates a framework-owned tag
// group via [Connector.deactivateSubsByTag] after OnShutdown returns.
//
// In TESTING deployment the [sub.Manual] flag is ignored, mirroring [Connector.activateSubs]:
// manual subs were brought online during startup, so they come off the bus here.
func (c *Connector) deactivateAutoSubs() error {
	var lastErr error
	deactivated := false
	for _, s := range c.subs.Values() {
		if s.Manual && c.deployment != TESTING {
			continue
		}
		if err := c.deactivateSub(s); err != nil {
			lastErr = errors.Trace(err)
		}
		deactivated = true
	}
	if deactivated {
		c.transportConn.WaitForSub()
	}
	return lastErr // No trace
}

// deactivateSub unsubscribes a single subscription with the transport.
func (c *Connector) deactivateSub(s *sub.Subscription) error {
	var lastErr error
	for _, sub := range s.Subs {
		err := sub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
		}
	}
	if lastErr != nil {
		c.LogError(c.Lifetime(), "Deactivating sub",
			"error", lastErr,
			"url", s.Canonical(),
			"method", s.Method,
		)
	}
	s.Subs = nil
	// c.LogDebug(c.Lifetime(), "Sub deactivated",
	// 	"url", s.Canonical(),
	// 	"method", s.Method,
	// )
	return lastErr
}

// ackRequest sends an ack response back to the caller.
// Acks are sent as soon as a request is received to let the caller know it is being processed.
func (c *Connector) ackRequest(msg *transport.Msg, s *sub.Subscription) (err error) {
	httpReq := msg.Request
	if httpReq == nil {
		// Parse only the headers of the request
		headerData := msg.Data
		eoh := bytes.Index(headerData, []byte("\r\n\r\n"))
		if eoh >= 0 {
			headerData = headerData[:eoh+4]
		}
		httpReq, err = http.ReadRequest(bufio.NewReaderSize(bytes.NewReader(headerData), 64))
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Get return address
	frm := frame.Of(httpReq)
	fromHost := frm.FromHost()
	if fromHost == "" {
		return errors.New("empty " + frame.HeaderFromHost + " header")
	}
	fromID := frm.FromID()
	if fromID == "" {
		return errors.New("empty " + frame.HeaderFromId + " header")
	}
	msgID := frm.MessageID()
	if msgID == "" {
		return errors.New("empty " + frame.HeaderMsgId + " header")
	}
	queue := s.Queue
	_, fragmentMax := frm.Fragment()

	// Prepare and send the ack
	httpRes := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Connection": []string{"close"},
		},
	}
	frm = frame.Of(httpRes)
	frm.SetOpCode(frame.OpCodeAck)
	frm.SetFromHost(c.hostname)
	frm.SetFromID(c.id)
	frm.SetMessageID(msgID)
	frm.SetQueue(queue)
	frm.SetLocality(c.locality)
	if fragmentMax > 1 {
		httpRes.StatusCode = http.StatusContinue
		httpRes.Status = "100 Continue"
	} else {
		httpRes.StatusCode = http.StatusAccepted
		httpRes.Status = "202 Accepted"
	}
	err = c.transportConn.Response(subjectOfResponse(c.plane, c.hostname, fromHost, fromID), httpRes)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// handleRequest is called when an incoming HTTP request is received.
// The message is dispatched to the appropriate web handler and the response is serialized and sent back to the response channel of the sender.
func (c *Connector) handleRequest(msg *transport.Msg, s *sub.Subscription) (err error) {
	ctx := c.Lifetime()

	httpReq := msg.Request
	if httpReq == nil {
		// Parse the request
		httpReq, err = http.ReadRequest(bufio.NewReaderSize(bytes.NewReader(msg.Data), 64))
		if err != nil {
			return errors.Trace(err)
		}
	}
	if httpReq.Header.Get("User-Agent") == "Go-http-client/1.1" {
		// Remove the default user-agent set by the http package
		httpReq.Header.Del("User-Agent")
	}
	httpx.SetPathValues(httpReq, s.Path)
	if br, ok := httpReq.Body.(*httpx.BodyReader); ok {
		br.Reset()
	}

	// Get the sender hostname and message ID
	fromHost := frame.Of(httpReq).FromHost()
	fromId := frame.Of(httpReq).FromID()
	msgID := frame.Of(httpReq).MessageID()
	queue := s.Queue

	c.LogDebug(c.Lifetime(), "Handling",
		"msg", msgID,
		"url", s.Canonical(),
		"method", s.Method,
	)

	// Time budget
	budget := frame.Of(httpReq).TimeBudget()
	if budget <= 0 {
		budget = c.defaultTimeBudget
	}
	budget = min(budget, c.maxTimeBudget)
	if s.TimeBudget > 0 {
		budget = min(budget, s.TimeBudget)
	}
	frame.Of(httpReq).SetTimeBudget(budget)
	budgetExhausted := budget <= c.networkRoundtrip

	// Integrate fragments together
	httpReq, err = c.defragRequest(httpReq)
	if err != nil {
		return errors.Trace(err)
	}
	if httpReq == nil {
		// Not all fragments arrived yet
		return nil
	}

	// OpenTelemetry: create a child span
	ctx = propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(httpReq.Header))
	var span trc.Span
	if s.NoTrace {
		span = trc.NewSpan(nil)
	} else {
		ctx, span = c.StartSpan(ctx, fmt.Sprintf(":%s%s", s.Port, s.Path),
			trc.Server(),
			trc.Request(httpReq),
			trc.String("http.route", s.Path),
		)
	}
	spanEnded := false
	defer func() {
		if !spanEnded {
			span.End()
		}
	}()

	// Execute the request
	handlerStartTime := time.Now()
	httpRecorder := httpx.NewResponseRecorder()
	var handlerErr error

	// Prepare the context with a timeout set to the time budget reduced by a network hop
	ctx = frame.ContextWithClonedFrameOf(ctx, httpReq.Header)
	ctx, cancel := context.WithTimeout(ctx, budget-c.networkRoundtrip)
	httpReq = httpReq.WithContext(ctx)
	httpReq.Header = frame.Of(ctx).Header()

	// A budget too small to dispatch fails fast as 408, routed through the error
	// response below so the caller is told rather than left to its own pub.Timeout.
	if budgetExhausted {
		handlerErr = errors.New("timeout", http.StatusRequestTimeout)
	}

	// Check actor constraints. Whenever an actor token is present it is verified,
	// even if the endpoint declares no requiredClaims, so a handler that reads claims
	// via IfActor/ParseActor can trust them. A missing token fails only when the
	// endpoint declares requiredClaims.
	if handlerErr == nil {
		actor := httpReq.Header.Get(frame.HeaderActor)
		if actor == "" || !utils.LooksLikeJWT(actor) {
			if s.RequiredClaims != "" {
				handlerErr = errors.New("", http.StatusUnauthorized)
			}
		} else {
			// An empty requiredClaims still verifies the token's signature; "true"
			// satisfies the claim evaluation unconditionally.
			requiredClaims := s.RequiredClaims
			if requiredClaims == "" {
				requiredClaims = "true"
			}
			var claims jwt.MapClaims
			claims, handlerErr = c.verifyToken(actor, requiredClaims)
			if handlerErr == nil {
				ctx = logActorClaims(ctx, claims)
				httpReq = httpReq.WithContext(ctx)
			}
		}
	}

	// Call the handler
	if handlerErr == nil {
		handlerErr = errors.CatchPanic(func() error {
			return s.Handler.(HTTPHandler)(httpRecorder, httpReq)
		})
	}
	cancel()

	if handlerErr != nil {
		convertedErr := errors.Convert(handlerErr)
		handlerErr = convertedErr
		if convertedErr.Error() == "http: request body too large" || // https://go.dev/src/net/http/request.go#L1150
			convertedErr.Error() == "http: POST too large" { // https://go.dev/src/net/http/request.go#L1240
			convertedErr.StatusCode = http.StatusRequestEntityTooLarge
		}
		statusCode := convertedErr.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.LogError(ctx, "Handling request",
			"error", convertedErr,
			"path", s.Path,
			"depth", frame.Of(httpReq).CallDepth(),
			"code", statusCode,
		)

		// OpenTelemetry: record the error
		span.SetError(convertedErr)
		c.ForceTrace(ctx)

		// Enrich error with trace ID
		convertedErr.Trace = span.TraceID()

		// Prepare an error response instead
		httpRecorder = httpx.NewResponseRecorder()
		httpRecorder.Header().Set("Content-Type", "application/json")
		httpRecorder.WriteHeader(statusCode)
		encoder := json.NewEncoder(httpRecorder)
		if c.Deployment() == LOCAL {
			encoder.SetIndent("", "  ")
		}
		serializedErr := struct {
			Err error `json:"err"`
		}{
			Err: convertedErr,
		}
		err = encoder.Encode(serializedErr)
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Meter
	_ = c.RecordHistogram(
		ctx,
		"microbus_server_request_duration_seconds",
		time.Since(handlerStartTime).Seconds(),
		"name", s.Name,
		"port", s.Port,
		"method", httpReq.Method,
		"code", httpRecorder.StatusCode(),
		"type", s.Type,
		"error", func() string {
			if handlerErr != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)
	_ = c.RecordHistogram(
		ctx,
		"microbus_server_response_body_bytes",
		float64(httpRecorder.ContentLength()),
		"name", s.Name,
		"port", s.Port,
		"method", httpReq.Method,
		"code", httpRecorder.StatusCode(),
		"type", s.Type,
		"error", func() string {
			if handlerErr != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)

	// Set control headers on the response
	httpResponse := httpRecorder.Result()
	frame.Of(httpResponse).SetMessageID(msgID)
	frame.Of(httpResponse).SetFromHost(c.hostname)
	frame.Of(httpResponse).SetFromID(c.id)
	frame.Of(httpResponse).SetFromVersion(c.version)
	frame.Of(httpResponse).SetQueue(queue)
	if handlerErr != nil {
		frame.Of(httpResponse).SetOpCode(frame.OpCodeError)
	} else {
		frame.Of(httpResponse).SetOpCode(frame.OpCodeResponse)
	}
	frame.Of(httpResponse).SetLocality(c.locality)

	// OpenTelemetry: record the status code
	if handlerErr == nil {
		span.SetOK(httpResponse.StatusCode)
	}
	span.End()
	spanEnded = true

	// Send back the response, in fragments if needed
	fragger, err := httpx.NewFragResponse(httpResponse, c.maxFragmentSize)
	if err != nil {
		return errors.Trace(err)
	}
	for f := 1; f <= fragger.N(); f++ {
		fragment, err := fragger.Fragment(f)
		if err != nil {
			return errors.Trace(err)
		}
		err = c.transportConn.Response(subjectOfResponse(c.plane, c.hostname, fromHost, fromId), fragment)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// notifyOnNewSubs notifies all microservices of this microservice's new subscriptions,
// allowing them to invalidate their known responders cache appropriately.
func (c *Connector) notifyOnNewSubs(subs ...*sub.Subscription) error {
	if len(subs) == 0 {
		return nil
	}
	hosts := make([]string, 0, len(subs))
	added := map[string]bool{}
	for _, s := range subs {
		if added[s.Host] {
			continue
		}
		hosts = append(hosts, s.Host)
		added[s.Host] = true
	}
	if len(hosts) == 0 {
		return nil
	}
	payload := struct {
		Hosts []string `json:"hosts"`
	}{
		Hosts: hosts,
	}
	ch := c.Publish(c.Lifetime(),
		pub.POST("https://all:888/on-new-subs"),
		pub.Body(payload),
		pub.Multicast(),
	)
	for range ch {
	}
	return nil
}

// verifyToken verifies the signature of an actor JWT and evaluates requiredClaims against
// its payload. On success it returns the verified claims, which the caller may cache for
// downstream non-security uses such as logging.
// If the kid is unknown, it fetches JWKS from the issuer host.
func (c *Connector) verifyToken(token string, requiredClaims string) (jwt.MapClaims, error) {
	// Get the alg and kid from the token's header
	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, errors.New("", http.StatusUnauthorized)
	}
	alg, _ := parsed.Header["alg"].(string)

	// Unsigned tokens (e.g. those minted by pub.Actor) are accepted only in TESTING,
	// where signing is impractical. Signature verification is skipped, but the claim
	// evaluation below still runs - an unsigned token's payload must satisfy
	// requiredClaims just like a signed one.
	if alg == "none" {
		if c.deployment != TESTING {
			return nil, errors.New("", http.StatusUnauthorized)
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("", http.StatusUnauthorized)
		}
		satisfy, err := boolexp.Eval(requiredClaims, map[string]any(claims))
		if err != nil {
			return nil, errors.Trace(err)
		}
		if !satisfy {
			return nil, errors.New("", http.StatusForbidden)
		}
		return claims, nil
	}

	kid, _ := parsed.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("", http.StatusUnauthorized)
	}

	// Verify the token with the issuer
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("", http.StatusUnauthorized)
	}
	issStr, _ := claims["iss"].(string)
	_, issuerHost, ok := strings.Cut(issStr, "://")
	if !ok {
		issuerHost = issStr
	}
	// Pin issuer to the framework's known token services. The iss claim is a sanity gate,
	// not a routing hint - keys are only ever fetched from the pinned hostnames.
	switch issuerHost {
	case "access.token.core", "bearer.token.core":
	default:
		return nil, errors.New("", http.StatusUnauthorized)
	}

	// Look up the public key, refresh cache if needed
	key, found := c.lookupActorKey(kid)
	if !found {
		err = c.fetchActorKeys(issuerHost)
		if err != nil {
			return nil, errors.New("", http.StatusUnauthorized, err)
		}
		key, found = c.lookupActorKey(kid)
		if !found {
			return nil, errors.New("", http.StatusUnauthorized)
		}
	}

	// Verify the JWT signature
	_, err = jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}))
	if err != nil {
		return nil, errors.New("", http.StatusUnauthorized)
	}

	// Check the required claims against the access token's claims
	satisfy, err := boolexp.Eval(requiredClaims, map[string]any(claims))
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !satisfy {
		return nil, errors.New("", http.StatusForbidden)
	}

	return claims, nil
}

// lookupActorKey returns the cached Ed25519 public key for the given kid.
func (c *Connector) lookupActorKey(kid string) (ed25519.PublicKey, bool) {
	c.actorKeysLock.RLock()
	defer c.actorKeysLock.RUnlock()
	key, ok := c.actorKeys[kid]
	return key, ok
}

// fetchActorKeys fetches JWKS from the given host at :888/jwks and updates the key cache.
// It is a no-op when the last fetch for the host was within 1s.
// Concurrent calls for the same host share a single fetch.
func (c *Connector) fetchActorKeys(host string) error {
	_, err, _ := c.jwksFlight.Do(host, func() (any, error) {
		c.actorKeysLock.Lock()
		if c.lastJWKSFetch == nil {
			c.lastJWKSFetch = map[string]time.Time{}
		}
		if last, ok := c.lastJWKSFetch[host]; ok && time.Since(last) < time.Second {
			c.actorKeysLock.Unlock()
			return nil, nil
		}
		c.actorKeysLock.Unlock()

		resp, err := c.Request(
			c.Lifetime(),
			pub.GET("https://"+host+":888/jwks"),
		)
		if err != nil {
			return nil, errors.Trace(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Trace(err)
		}
		var jwksResp struct {
			Keys []struct {
				KID string `json:"kid"`
				X   string `json:"x"`
			} `json:"keys"`
		}
		err = json.Unmarshal(body, &jwksResp)
		if err != nil {
			return nil, errors.Trace(err)
		}

		c.actorKeysLock.Lock()
		defer c.actorKeysLock.Unlock()
		c.lastJWKSFetch[host] = time.Now()
		if c.actorKeys == nil {
			c.actorKeys = make(map[string]ed25519.PublicKey)
		}
		for _, jwk := range jwksResp.Keys {
			pubBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
			if err != nil {
				continue
			}
			c.actorKeys[jwk.KID] = ed25519.PublicKey(pubBytes)
		}
		return nil, nil
	})
	return errors.Trace(err)
}

// invalidateKnownRespondersCache clears known responder cache for any of the indicated hosts.
// It is called in response to a notification from other microservices on new subscriptions.
func (c *Connector) invalidateKnownRespondersCache(hosts []string) error {
	if len(hosts) == 0 {
		return nil
	}
	hostSet := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		hostSet[host] = true
	}
	c.knownResponders.DeletePredicate(func(key string) bool {
		_, _, _, _, dest, _ := splitSubject(key)
		if !hostSet[dest] {
			return false
		}
		c.LogDebug(c.Lifetime(), "Invalidating known responders cache", "host", dest)
		return true
	})
	return nil
}
