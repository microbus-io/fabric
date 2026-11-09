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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/openapi"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
)

// subscribeControl creates subscriptions for control requests on the reserved port 888.
func (c *Connector) subscribeControl() (err error) {
	if c.controlSubs {
		// Already subscribed
		return nil
	}
	type ctrlSub struct {
		name    string
		route   string
		handler HTTPHandler
		options []sub.Option
	}
	subs := []*ctrlSub{
		{name: "Ping", route: ":888/ping", handler: c.handleControlPing, options: []sub.Option{sub.NoQueue()}},
		{name: "ConfigRefresh", route: ":888/config-refresh", handler: c.handleControlConfigRefresh, options: []sub.Option{sub.NoQueue()}},
		{name: "Metrics", route: ":888/metrics", handler: c.handleMetrics, options: []sub.Option{sub.NoQueue()}},
		{name: "Trace", route: ":888/trace", handler: c.handleTrace, options: []sub.Option{sub.NoQueue()}},
		{name: "OnNewSubs", route: ":888/on-new-subs", handler: c.handleOnNewSubs, options: []sub.Option{sub.NoQueue(), sub.NoTrace()}},
		{name: "OpenAPI", route: ":888/openapi.json", handler: c.handleOpenAPI, options: []sub.Option{sub.DefaultQueue(), sub.Method("GET")}},
	}
	var registered []string
	rollback := func() {
		for _, name := range registered {
			_ = c.Unsubscribe(name)
		}
	}
	for _, s := range subs {
		opts := append(s.options, sub.Route(s.route), sub.Web())
		name := "Ctrl888" + s.name
		err := c.Subscribe(name, s.handler, opts...)
		if err != nil {
			rollback()
			return errors.Trace(err)
		}
		registered = append(registered, name)

		opts = append(s.options, sub.Route("//all"+s.route), sub.Web())
		name = "Ctrl888" + s.name + "All"
		err = c.Subscribe(name, s.handler, opts...)
		if err != nil {
			rollback()
			return errors.Trace(err)
		}
		registered = append(registered, name)
	}

	c.controlSubs = true
	return nil
}

// handleOpenAPI renders the OpenAPI 3.1 document for the subscriptions registered with this
// connector. Filters by subscription type (only function, web, workflow and task appear) and by the
// caller's actor claims - subscriptions the caller is not authorized to invoke are omitted. Ports are
// preserved on each rendered endpoint; consumers (portal/MCP) apply any port-based filtering
// at their own ingress boundary. The response is marked private/no-store because its content
// varies by the caller's claims.
func (c *Connector) handleOpenAPI(w http.ResponseWriter, r *http.Request) error {
	actor := r.Header.Get(frame.HeaderActor)

	oapiSvc := &openapi.Service{
		ServiceName: c.hostname,
		Description: c.description,
		Version:     c.version,
		RemoteURI:   frame.Of(r).XForwardedFullURL(),
	}
	for _, s := range c.subs.Values() {
		// Skip subs registered on the broadcast plane (//all/...): they are mirrors of an
		// endpoint already registered against the primary host, not separate operations.
		if s.Host == "all" || strings.HasSuffix(s.Host, ".all") {
			continue
		}
		// Skip subs that aren't on the bus. This excludes manual subs registered via
		// sub.Manual() that haven't yet been activated via ActivateSubscription -
		// otherwise the document advertises endpoints callers can't currently reach.
		if len(s.Subs) == 0 {
			continue
		}
		// sub.Type values intentionally match the openapi.Feature* string values 1:1
		// (function/web/workflow/task), so the type field passes through unchanged.
		// Tasks are included so internal inspector tools (e.g. agentstudio) can
		// discover task signatures; consumers that should not surface tasks
		// (openapiportal, mcpportal, llm.core) whitelist the feature types they
		// expose - the producer no longer pre-filters.
		switch s.Type {
		case sub.TypeFunction, sub.TypeWeb, sub.TypeWorkflow, sub.TypeTask:
		default:
			continue
		}
		if s.RequiredClaims != "" {
			if actor == "" || !utils.LooksLikeJWT(actor) {
				continue
			}
			if _, err := c.verifyToken(actor, s.RequiredClaims); err != nil {
				continue
			}
		}
		route := s.Path
		if !strings.HasPrefix(route, ":") && !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		oapiSvc.Endpoints = append(oapiSvc.Endpoints, &openapi.Endpoint{
			Type:        s.Type,
			Name:        s.Name,
			Hostname:    s.Host,
			Method:      s.Method,
			Route:       ":" + s.Port + route,
			Summary:     summarizeSubscription(s),
			Description: s.Description,
			InputArgs:   s.Inputs,
			OutputArgs:  s.Outputs,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	encoder := json.NewEncoder(w)
	if c.deployment == LOCAL {
		encoder.SetIndent("", "  ")
	}
	return errors.Trace(encoder.Encode(openapi.Render(oapiSvc)))
}

// summarizeSubscription builds a Go-style signature for an endpoint, e.g.
// "MyFunction(x int, y Point) (z map[string]Point)". Field names come from the json tag of
// each Input/Output struct field. Type names are reflected and stripped of any package
// qualifier (so "myserviceapi.Point" becomes "Point").
func summarizeSubscription(s *sub.Subscription) string {
	in := fieldList(s.Inputs)
	out := fieldList(s.Outputs)
	if out == "" {
		return fmt.Sprintf("%s(%s)", s.Name, in)
	}
	return fmt.Sprintf("%s(%s) (%s)", s.Name, in, out)
}

// pkgQualifier matches a package qualifier (lowercase identifier followed by a dot) anywhere
// inside a reflected type string, e.g. "errors." in "*errors.TracedError" or "myserviceapi."
// in "map[string]myserviceapi.Point".
var pkgQualifier = regexp.MustCompile(`\b[a-z][a-zA-Z0-9_]*\.`)

func fieldList(v any) string {
	if v == nil {
		return ""
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	var parts []string
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			continue
		}
		if name == "" {
			// Lowercase the first rune to keep the camelCase convention used by the rest
			// of the framework.
			if len(f.Name) > 0 {
				name = strings.ToLower(f.Name[:1]) + f.Name[1:]
			} else {
				name = f.Name
			}
		}
		parts = append(parts, name+" "+pkgQualifier.ReplaceAllString(f.Type.String(), ""))
	}
	return strings.Join(parts, ", ")
}

// handleControlPing responds to the :888/ping control request with a pong.
func (c *Connector) handleControlPing(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"pong":0}`))
	return nil
}

// handleControlConfigRefresh responds to the :888/config-refresh control request
// by pulling the latest config values from the configurator service.
func (c *Connector) handleControlConfigRefresh(w http.ResponseWriter, r *http.Request) error {
	err := c.refreshConfig(r.Context(), true)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}

// handleMetrics responds to the :888/metrics control request with collected metrics.
func (c *Connector) handleMetrics(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	c.observeMetricsJustInTime(ctx)
	if c.metricsHandler != nil {
		if c.Deployment() == LOCAL {
			// Do not compress the response on local to avoid special characters when running NATS is debug mode
			r.Header.Del("Accept-Encoding")
		}
		c.metricsHandler.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotImplemented)
	}
	return nil
}

// handleTrace responds to the :888/trace control request to force exporting the indicated tracing span.
func (c *Connector) handleTrace(w http.ResponseWriter, r *http.Request) error {
	if c.traceProcessor != nil {
		traceID := r.URL.Query().Get("id")
		c.traceProcessor.Select(traceID)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}

// handleOnNewSubs responds to the :888/on-new-subs control request to update the known responders cache.
func (c *Connector) handleOnNewSubs(w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Hosts []string `json:"hosts"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.invalidateKnownRespondersCache(payload.Hosts)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}
