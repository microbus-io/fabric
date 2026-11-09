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

package mcpportal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/coreservices/mcpportal/mcpportalapi"
	"github.com/microbus-io/fabric/coreservices/openapiportal/openapiportalapi"
	"github.com/microbus-io/fabric/openapi"
	"github.com/microbus-io/fabric/pub"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ mcpportalapi.Client
)

// mcpProtocolVersion is the MCP spec revision the server supports.
const mcpProtocolVersion = "2024-11-05"

// JSON-RPC 2.0 error codes (https://www.jsonrpc.org/specification#error_object).
const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcInternalError  = -32603
)

/*
Service implements mcpportal.core, an MCP-protocol facade in front of the bus.

The microservice exposes one HTTP wire endpoint (POST //mcp:0) that accepts JSON-RPC 2.0
envelopes and dispatches on the method field to handlers for initialize, tools/list,
and tools/call.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

// jsonrpcRequest is one JSON-RPC 2.0 envelope. `ID` is left as raw JSON so we can echo
// numeric or string IDs back unchanged. `Params` is also raw - each method handler decodes
// the shape it expects.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

// jsonrpcError is the inner error object of a JSON-RPC failure response.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

/*
MCP is the JSON-RPC 2.0 wire endpoint for Model Context Protocol clients. It reads one envelope and
dispatches on the JSON-RPC method to internal handlers for initialize, tools/list, and tools/call,
writing one envelope back. Notifications (envelopes without an id) are acknowledged with 200 and no body
per spec.
*/
func (svc *Service) MCP(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: MCP
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return writeRPCError(w, nil, rpcParseError, "failed to read request body")
	}
	var req jsonrpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return writeRPCError(w, nil, rpcParseError, "invalid JSON envelope")
	}
	if req.JSONRPC != "2.0" {
		return writeRPCError(w, req.ID, rpcInvalidRequest, "missing or invalid jsonrpc version")
	}

	if len(req.ID) == 0 && strings.HasPrefix(req.Method, "notifications/") {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	var result any
	var rpcErr *jsonrpcError
	switch req.Method {
	case "initialize":
		result, rpcErr = svc.handleInitialize()
	case "tools/list":
		result, rpcErr = svc.handleToolsList(r)
	case "tools/call":
		result, rpcErr = svc.handleToolsCall(r, req.Params)
	case "ping":
		result = map[string]any{}
	default:
		rpcErr = &jsonrpcError{Code: rpcMethodNotFound, Message: "method not found: " + req.Method}
	}

	if rpcErr != nil {
		return writeRPCError(w, req.ID, rpcErr.Code, rpcErr.Message)
	}
	return writeRPCResult(w, req.ID, result)
}

// handleInitialize answers an MCP `initialize` request.
func (svc *Service) handleInitialize() (any, *jsonrpcError) {
	return map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "Microbus MCP Portal",
			"version": strconv.Itoa(int(svc.Version())),
		},
	}, nil
}

// handleToolsList converts each operation in the OpenAPI aggregate into an MCP tool
// descriptor. The `inputSchema` is the operation's request body schema for body-bearing
// methods, otherwise an object schema synthesized from the operation's parameters.
func (svc *Service) handleToolsList(r *http.Request) (any, *jsonrpcError) {
	doc, err := svc.fetchAggregate(r)
	if err != nil {
		return nil, &jsonrpcError{Code: rpcInternalError, Message: err.Error()}
	}
	tools := make([]map[string]any, 0)
	usedNames := map[string]bool{}
	// Iterate paths and methods in sorted order so name disambiguation (the `_2`, `_3`
	// suffix scheme) is deterministic across calls. Path keys begin with `/host:port/...`
	// so lexical order also groups by hostname.
	pathKeys := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)
	for _, path := range pathKeys {
		methods := doc.Paths[path]
		methodKeys := make([]string, 0, len(methods))
		for m := range methods {
			methodKeys = append(methodKeys, m)
		}
		sort.Strings(methodKeys)
		for _, method := range methodKeys {
			op := methods[method]
			if op == nil || op.XName == "" {
				continue
			}
			if !exposedFeatureType(op.XFeatureType) {
				continue
			}
			name := op.XName
			// Disambiguate name collisions: first occurrence keeps the bare name, later
			// ones get `_2`, `_3`, ... suffixes so the LLM can address them distinctly.
			if usedNames[name] {
				for i := 2; ; i++ {
					candidate := name + "_" + strconv.Itoa(i)
					if !usedNames[candidate] {
						name = candidate
						break
					}
				}
			}
			usedNames[name] = true
			tools = append(tools, map[string]any{
				"name":        name,
				"description": op.Description,
				"inputSchema": buildToolInputSchema(doc, op),
			})
		}
	}
	return map[string]any{"tools": tools}, nil
}

// handleToolsCall invokes the named tool with the supplied arguments and returns the
// response body wrapped in MCP's `content` array. Non-2xx responses and transport errors
// surface as `isError: true` content blocks rather than JSON-RPC errors.
func (svc *Service) handleToolsCall(r *http.Request, params json.RawMessage) (any, *jsonrpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &jsonrpcError{Code: rpcInvalidParams, Message: "invalid params"}
	}
	if p.Name == "" {
		return nil, &jsonrpcError{Code: rpcInvalidParams, Message: "missing tool name"}
	}

	doc, err := svc.fetchAggregate(r)
	if err != nil {
		return nil, &jsonrpcError{Code: rpcInternalError, Message: err.Error()}
	}

	var (
		toolMethod string
		toolPath   string
		toolOp     *openapi.Operation
	)
	// Iterate in sorted order so a tool name that maps to multiple operations resolves
	// to the same one across calls, matching handleToolsList's disambiguation order.
	pathKeys := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)
	for _, path := range pathKeys {
		methods := doc.Paths[path]
		methodKeys := make([]string, 0, len(methods))
		for m := range methods {
			methodKeys = append(methodKeys, m)
		}
		sort.Strings(methodKeys)
		for _, method := range methodKeys {
			op := methods[method]
			if op == nil || !exposedFeatureType(op.XFeatureType) {
				continue
			}
			if op.XName == p.Name {
				toolMethod = strings.ToUpper(method)
				toolPath = path
				toolOp = op
				break
			}
		}
		if toolOp != nil {
			break
		}
	}
	if toolOp == nil {
		return errorContent(errors.New("tool not found", "name", p.Name, http.StatusNotFound)), nil
	}

	// Path keys are `/host:port/route`; reconstruct the full bus URL.
	toolURL := "https:/" + toolPath
	body := []byte(p.Arguments)
	if len(body) == 0 {
		body = []byte("{}")
	}
	res, err := svc.Request(r.Context(),
		pub.Method(toolMethod),
		pub.URL(toolURL),
		pub.Body(body),
		pub.ContentType("application/json"),
	)
	if err != nil {
		return errorContent(err), nil
	}
	respBody, _ := io.ReadAll(res.Body)
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(respBody)}},
	}, nil
}

// fetchAggregate retrieves the OpenAPI portal's aggregate doc with a port override on the
// URL. Without the override, the portal handler sees `r.URL.Port() == "0"` (the wildcard
// from its route) and zeroes out the port filter; the override forces it to filter to the
// desired port. A missing request port defaults to 443.
func (svc *Service) fetchAggregate(r *http.Request) (*openapi.Document, error) {
	port := r.URL.Port()
	if port == "" {
		port = "443"
	}
	overrideURL := strings.Replace(openapiportalapi.Document.URL(), ":0", ":"+port, 1)
	res, err := openapiportalapi.NewClient(svc).
		WithOptions(pub.URL(overrideURL)).
		Document(r.Context(), "")
	if err != nil {
		return nil, errors.Trace(err)
	}
	var doc openapi.Document
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return nil, errors.Trace(err)
	}
	return &doc, nil
}

// exposedFeatureType is the whitelist of x-feature-type values mcpportal turns into MCP tools: functions and
// web handlers. Tasks and workflows are not exposed.
func exposedFeatureType(t string) bool {
	switch t {
	case openapi.FeatureFunction, openapi.FeatureWeb:
		return true
	}
	return false
}

// buildToolInputSchema returns a self-contained JSON Schema describing the tool's input.
// For body-bearing operations, the request body schema is used (with refs inlined). For
// query/path-arg operations, an object schema is synthesized from the parameters.
func buildToolInputSchema(doc *openapi.Document, op *openapi.Operation) any {
	if op.RequestBody != nil {
		if media := op.RequestBody.Content["application/json"]; media != nil && media.Schema != nil {
			b, err := json.Marshal(media.Schema)
			if err == nil {
				var v any
				if err := json.Unmarshal(b, &v); err == nil {
					return inlineSchemaRefs(doc, v, map[string]bool{})
				}
			}
		}
	}
	props := map[string]any{}
	var required []string
	for _, p := range op.Parameters {
		if p == nil || p.Name == "" {
			continue
		}
		var paramSchema any = map[string]any{"type": "string"}
		if p.Schema != nil {
			b, err := json.Marshal(p.Schema)
			if err == nil {
				var v any
				if err := json.Unmarshal(b, &v); err == nil {
					paramSchema = inlineSchemaRefs(doc, v, map[string]bool{})
				}
			}
		}
		if p.Description != "" {
			if m, ok := paramSchema.(map[string]any); ok {
				if _, has := m["description"]; !has {
					m["description"] = p.Description
				}
			}
		}
		props[p.Name] = paramSchema
		if p.Required {
			required = append(required, p.Name)
		}
	}
	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

// inlineSchemaRefs walks v and replaces every `{"$ref": "#/components/schemas/<key>"}` map
// with the resolved schema's content (recursively). Cycles short-circuit via `visiting`: a
// ref already on the resolution path keeps its `$ref` string so recursion terminates.
func inlineSchemaRefs(doc *openapi.Document, v any, visiting map[string]bool) any {
	switch t := v.(type) {
	case map[string]any:
		if ref, ok := t["$ref"].(string); ok {
			const prefix = "#/components/schemas/"
			if strings.HasPrefix(ref, prefix) && doc != nil && doc.Components != nil {
				key := ref[len(prefix):]
				if !visiting[key] {
					if target, exists := doc.Components.Schemas[key]; exists && target != nil {
						b, err := json.Marshal(target)
						if err == nil {
							var inner any
							if err := json.Unmarshal(b, &inner); err == nil {
								visiting[key] = true
								inlined := inlineSchemaRefs(doc, inner, visiting)
								delete(visiting, key)
								return inlined
							}
						}
					}
				}
			}
			return t
		}
		for k, val := range t {
			t[k] = inlineSchemaRefs(doc, val, visiting)
		}
		return t
	case []any:
		for i, el := range t {
			t[i] = inlineSchemaRefs(doc, el, visiting)
		}
		return t
	default:
		return v
	}
}

// errorContent wraps an error in MCP's content-block shape with `isError: true`. The
// message is prefixed with the HTTP status code when the error carries one, so the model can
// distinguish retryable failures (5XX) from permanent ones (4XX) without parsing structure.
func errorContent(err error) map[string]any {
	msg := err.Error()
	if code := errors.StatusCode(err); code > 0 {
		msg = fmt.Sprintf("[%d] %s", code, msg)
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
		"isError": true,
	}
}

// writeRPCResult writes a successful JSON-RPC response. `id` is echoed back as raw JSON so
// numeric and string IDs round-trip unchanged.
func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result any) error {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"jsonrpc": "2.0",
		"result":  result,
	}
	if len(id) > 0 {
		resp["id"] = id
	}
	return errors.Trace(json.NewEncoder(w).Encode(resp))
}

// writeRPCError writes a JSON-RPC failure envelope. A nil/empty `id` is rendered as
// `"id": null` per spec.
func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) error {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"jsonrpc": "2.0",
		"error":   map[string]any{"code": code, "message": message},
	}
	if len(id) > 0 {
		resp["id"] = id
	} else {
		resp["id"] = nil
	}
	return errors.Trace(json.NewEncoder(w).Encode(resp))
}
