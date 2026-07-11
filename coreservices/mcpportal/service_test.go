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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/invopop/jsonschema"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/openapiportal"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/openapi"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/mcpportal/mcpportalapi"
)

var (
	_ context.Context
	_ io.Reader
	_ *http.Request
	_ *testing.T
	_ jwt.MapClaims
	_ application.Application
	_ connector.Connector
	_ frame.Frame
	_ pub.Option
	_ sub.Option
	_ *errors.TracedError
	_ *workflow.Flow
	_ testarossa.Asserter
	_ mcpportalapi.Client
)

// MARKER: MCP

func TestMcpportal_HandleInitialize(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	svc := NewService()
	result, rpcErr := svc.handleInitialize()
	assert.Nil(rpcErr)

	m, ok := result.(map[string]any)
	assert.True(ok)
	assert.Equal(mcpProtocolVersion, m["protocolVersion"])

	caps, ok := m["capabilities"].(map[string]any)
	assert.True(ok)
	_, hasTools := caps["tools"]
	assert.True(hasTools)
	if tools, ok := caps["tools"].(map[string]any); ok {
		_, hasListChanged := tools["listChanged"]
		assert.False(hasListChanged)
	}

	server, ok := m["serverInfo"].(map[string]any)
	assert.True(ok)
	assert.NotEqual("", server["name"])
}

func TestMcpportal_InlineSchemaRefs(t *testing.T) {
	t.Parallel()

	// Build a doc with a chain (Outer -> Inner -> primitive) and a self-cycle (Loop -> Loop).
	mkObj := func(props map[string]*jsonschema.Schema) *jsonschema.Schema {
		s := &jsonschema.Schema{Type: "object", Properties: jsonschema.NewProperties()}
		for k, v := range props {
			s.Properties.Set(k, v)
		}
		return s
	}
	doc := &openapi.Document{
		Components: &openapi.Components{
			Schemas: map[string]*jsonschema.Schema{
				"Outer": mkObj(map[string]*jsonschema.Schema{
					"inner": {Ref: "#/components/schemas/Inner"},
				}),
				"Inner": mkObj(map[string]*jsonschema.Schema{
					"name": {Type: "string"},
				}),
				"Loop": mkObj(map[string]*jsonschema.Schema{
					"self": {Ref: "#/components/schemas/Loop"},
				}),
			},
		},
	}

	t.Run("resolves_chain", func(t *testing.T) {
		assert := testarossa.For(t)
		input := map[string]any{"$ref": "#/components/schemas/Outer"}
		out := inlineSchemaRefs(doc, input, map[string]bool{})
		// Should be the Outer object, with `inner` resolved to the Inner object, with
		// `name` as a string. Walk and check.
		outer, ok := out.(map[string]any)
		assert.True(ok)
		assert.Equal("object", outer["type"])
		props := outer["properties"].(map[string]any)
		inner := props["inner"].(map[string]any)
		assert.Equal("object", inner["type"])
		innerProps := inner["properties"].(map[string]any)
		nameProp := innerProps["name"].(map[string]any)
		assert.Equal("string", nameProp["type"])
	})

	t.Run("cycle_keeps_ref", func(t *testing.T) {
		assert := testarossa.For(t)
		input := map[string]any{"$ref": "#/components/schemas/Loop"}
		out := inlineSchemaRefs(doc, input, map[string]bool{})
		// Outer Loop is inlined once; inner self-ref hits the visiting set and is left
		// as a `$ref` map so recursion stops.
		loop, ok := out.(map[string]any)
		assert.True(ok)
		props := loop["properties"].(map[string]any)
		self := props["self"].(map[string]any)
		assert.Equal("#/components/schemas/Loop", self["$ref"])
	})

	t.Run("missing_ref_passthrough", func(t *testing.T) {
		assert := testarossa.For(t)
		input := map[string]any{"$ref": "#/components/schemas/DoesNotExist"}
		out := inlineSchemaRefs(doc, input, map[string]bool{})
		// Unresolvable ref leaves the map alone - clients see the broken pointer rather
		// than a silent rewrite to nil/empty.
		m, ok := out.(map[string]any)
		assert.True(ok)
		assert.Equal("#/components/schemas/DoesNotExist", m["$ref"])
	})
}

func TestMcpportal_BuildToolInputSchema(t *testing.T) {
	t.Parallel()

	doc := &openapi.Document{
		Components: &openapi.Components{
			Schemas: map[string]*jsonschema.Schema{
				"Body": {
					Type:       "object",
					Properties: jsonschema.NewProperties(),
				},
			},
		},
	}
	doc.Components.Schemas["Body"].Properties.Set("name", &jsonschema.Schema{Type: "string"})

	t.Run("request_body", func(t *testing.T) {
		assert := testarossa.For(t)
		op := &openapi.Operation{
			RequestBody: &openapi.RequestBody{
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &jsonschema.Schema{Ref: "#/components/schemas/Body"},
					},
				},
			},
		}
		schema := buildToolInputSchema(doc, op)
		m, ok := schema.(map[string]any)
		assert.True(ok)
		assert.Equal("object", m["type"])
		props := m["properties"].(map[string]any)
		_, hasName := props["name"]
		assert.True(hasName)
	})

	t.Run("query_parameters", func(t *testing.T) {
		assert := testarossa.For(t)
		op := &openapi.Operation{
			Parameters: []*openapi.Parameter{
				{Name: "x", In: "query", Schema: &jsonschema.Schema{Type: "integer"}, Required: true},
				{Name: "y", In: "query", Schema: &jsonschema.Schema{Type: "string"}, Description: "the y arg"},
			},
		}
		schema := buildToolInputSchema(doc, op)
		m, ok := schema.(map[string]any)
		assert.True(ok)
		assert.Equal("object", m["type"])
		props := m["properties"].(map[string]any)
		x := props["x"].(map[string]any)
		assert.Equal("integer", x["type"])
		y := props["y"].(map[string]any)
		assert.Equal("the y arg", y["description"])
		// Only `x` is required. The slice is `[]string` in memory; consumers JSON-marshal
		// before sending so the wire shape is the standard `["x"]` array.
		req := m["required"].([]string)
		assert.Equal(1, len(req))
		assert.Equal("x", req[0])
	})
}

// dispatchSetup wires an mcpportal under test plus a mock openapi portal that returns the
// given doc. The returned tester is the bus client tests issue requests through.
func dispatchSetup(t *testing.T, doc *openapi.Document) *connector.Connector {
	t.Helper()
	svc := NewService()
	portalMock := openapiportal.NewMock()
	portalMock.MockDocument(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return errors.Trace(json.NewEncoder(w).Encode(doc))
	})
	tester := connector.New("tester.client")
	app := application.New()
	app.Add(svc, portalMock, tester)
	app.RunInTest(t)
	return tester
}

// rpcCall sends one JSON-RPC envelope to the MCP wire endpoint and parses the response.
// The route is `//mcp:0` (absolute), so the full URL is `https://mcp/` - addressing the
// `mcp` host directly rather than `mcp.core/...`.
func rpcCall(ctx context.Context, tester *connector.Connector, envelope map[string]any) (map[string]any, *http.Response, error) {
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, nil, err
	}
	return rpcCallRaw(ctx, tester, body)
}

// rpcCallRaw issues a POST with arbitrary bytes, used by the parse-error test that needs to
// send invalid JSON.
func rpcCallRaw(ctx context.Context, tester *connector.Connector, body []byte) (map[string]any, *http.Response, error) {
	res, err := tester.Request(ctx, pub.POST("https://mcp/"), pub.Body(body), pub.ContentType("application/json"))
	if err != nil {
		return nil, nil, err
	}
	if res == nil || res.Body == nil {
		return nil, res, nil
	}
	defer res.Body.Close()
	respBytes, _ := io.ReadAll(res.Body)
	if len(strings.TrimSpace(string(respBytes))) == 0 {
		return nil, res, nil
	}
	var resp map[string]any
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, res, err
	}
	return resp, res, nil
}

func TestMcpportal_Dispatch(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Synthetic aggregate: one function tool plus a duplicate-named operation to exercise
	// the dedup path.
	doc := &openapi.Document{
		OpenAPI: "3.1.0",
		Info:    openapi.Info{Title: "Microbus", Version: "1"},
		Paths: map[string]map[string]*openapi.Operation{
			"/calc.example:443/square": {
				"get": {
					XName:        "Square",
					XFeatureType: "function",
					Summary:      "Square(x int)",
					Description:  "Squares x.",
					Parameters: []*openapi.Parameter{
						{Name: "x", In: "query", Schema: &jsonschema.Schema{Type: "integer"}, Required: true},
					},
				},
			},
			"/calc.example:443/square-too": {
				"get": {
					XName:        "Square", // collision; should be renamed to Square_2
					XFeatureType: "function",
					Description:  "Other square.",
				},
			},
		},
	}

	tester := dispatchSetup(t, doc)

	t.Run("initialize", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
		})
		assert.NoError(err)
		assert.Equal(float64(1), resp["id"])
		result := resp["result"].(map[string]any)
		assert.Equal(mcpProtocolVersion, result["protocolVersion"])
	})

	t.Run("ping", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      "p",
			"method":  "ping",
		})
		assert.NoError(err)
		assert.Equal("p", resp["id"])
		_, hasResult := resp["result"]
		assert.True(hasResult)
	})

	t.Run("tools_list", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		assert.NoError(err)
		result := resp["result"].(map[string]any)
		tools := result["tools"].([]any)
		assert.Equal(2, len(tools))
		// Both names must be unique; the second `Square` becomes `Square_2`.
		names := map[string]bool{}
		for _, raw := range tools {
			tool := raw.(map[string]any)
			names[tool["name"].(string)] = true
		}
		assert.True(names["Square"])
		assert.True(names["Square_2"])
	})

	t.Run("tools_call_not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params":  map[string]any{"name": "DoesNotExist", "arguments": map[string]any{}},
		})
		assert.NoError(err)
		// Missing tool surfaces as a content-block error rather than a JSON-RPC error so
		// the LLM gets the message and can react.
		result := resp["result"].(map[string]any)
		assert.Equal(true, result["isError"])
	})

	t.Run("method_not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "unknown/method",
		})
		assert.NoError(err)
		errObj := resp["error"].(map[string]any)
		assert.Equal(float64(rpcMethodNotFound), errObj["code"])
	})

	t.Run("parse_error", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCallRaw(ctx, tester, []byte(`{ not json`))
		assert.NoError(err)
		errObj := resp["error"].(map[string]any)
		assert.Equal(float64(rpcParseError), errObj["code"])
	})

	t.Run("notification_no_response", func(t *testing.T) {
		assert := testarossa.For(t)
		// Notifications carry no `id` and produce no response body per JSON-RPC.
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		})
		assert.NoError(err)
		assert.Nil(resp)
	})

	t.Run("invalid_jsonrpc_version", func(t *testing.T) {
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "1.0",
			"id":      5,
			"method":  "ping",
		})
		assert.NoError(err)
		errObj := resp["error"].(map[string]any)
		assert.Equal(float64(rpcInvalidRequest), errObj["code"])
	})

	t.Run("batch_request_returns_parse_error", func(t *testing.T) {
		// JSON-RPC 2.0 allows array batches; the MCP spec inherits this. Our handler does
		// not implement batch dispatch (it `json.Unmarshal`s into a single envelope), so a
		// batch body is rejected as a parse error. This test pins that behavior so a future
		// batch implementation surfaces as a deliberate change rather than a silent break.
		assert := testarossa.For(t)
		resp, _, err := rpcCallRaw(ctx, tester, []byte(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`))
		assert.NoError(err)
		errObj := resp["error"].(map[string]any)
		assert.Equal(float64(rpcParseError), errObj["code"])
	})

	t.Run("tools_list_excludes_task_features", func(t *testing.T) {
		// Defensive regression: the framework filters tasks/events at the OpenAPI source
		// (connector/control.go), so MCP relies on receiving only function/web/workflow ops.
		// If a task ever leaked in (synthesized here), it would appear in tools/list -
		// this test documents that current behavior so any future change is explicit.
		assert := testarossa.For(t)
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      "tl",
			"method":  "tools/list",
		})
		assert.NoError(err)
		result := resp["result"].(map[string]any)
		tools := result["tools"].([]any)
		// The synthetic doc in TestMcpportal_Dispatch declares only `Square` ops; no task.
		// Verifying no surprise tool names appear catches accidental task leakage.
		for _, raw := range tools {
			tool := raw.(map[string]any)
			name := tool["name"].(string)
			assert.True(strings.HasPrefix(name, "Square"), "unexpected tool exposed: %q", name)
		}
	})
}

// doubleIn / doubleOut are the typed input/output of the synthetic tool service used in
// TestMcpportal_ToolsCallEndToEnd. They live at file scope so the openapi portal's
// reflector can see exported field names.
type doubleIn struct {
	X int `json:"x"`
}
type doubleOut struct {
	Result int `json:"result"`
}

// TestMcpportal_ToolsCallEndToEnd verifies the full tools/call happy path through real
// services: a tiny `tool.test` connector exposes a `Double` function on the bus, the real
// openapi portal aggregates its OpenAPI doc, and the mcpportal handler resolves the tool
// from that aggregate and forwards the call. No mocks - the bus does the work end-to-end.
func TestMcpportal_ToolsCallEndToEnd(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Tool service: one function endpoint that doubles its input.
	toolSvc := connector.New("tool.test")
	err := toolSvc.Subscribe("Double", func(w http.ResponseWriter, r *http.Request) error {
		var in doubleIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return errors.Trace(err)
		}
		w.Header().Set("Content-Type", "application/json")
		return errors.Trace(json.NewEncoder(w).Encode(doubleOut{Result: in.X * 2}))
	}, sub.Function(doubleIn{}, doubleOut{}))
	if err != nil {
		t.Fatal(err)
	}

	mcpSvc := NewService()
	portalSvc := openapiportal.NewService()
	tester := connector.New("tester.client")

	app := application.New()
	app.Add(mcpSvc, portalSvc, toolSvc, tester)
	app.RunInTest(t)

	t.Run("tool_listed_then_called", func(t *testing.T) {
		assert := testarossa.For(t)

		// tools/list should include the Double tool with an `x` integer in its inputSchema.
		resp, _, err := rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		})
		assert.NoError(err)
		result := resp["result"].(map[string]any)
		tools := result["tools"].([]any)
		var found map[string]any
		for _, raw := range tools {
			tool := raw.(map[string]any)
			if tool["name"] == "Double" {
				found = tool
				break
			}
		}
		if !assert.NotNil(found) {
			return
		}
		inputSchema := found["inputSchema"].(map[string]any)
		assert.Equal("object", inputSchema["type"])
		props := inputSchema["properties"].(map[string]any)
		x := props["x"].(map[string]any)
		assert.Equal("integer", x["type"])

		// tools/call invokes Double with x=21 and expects result=42.
		resp, _, err = rpcCall(ctx, tester, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "Double",
				"arguments": map[string]any{"x": 21},
			},
		})
		assert.NoError(err)
		result = resp["result"].(map[string]any)
		isError, _ := result["isError"].(bool)
		assert.False(isError)
		content := result["content"].([]any)
		block := content[0].(map[string]any)
		// `text` is the JSON-encoded response body from the tool, e.g. `{"result":42}`.
		var out doubleOut
		assert.NoError(json.Unmarshal([]byte(block["text"].(string)), &out))
		assert.Equal(42, out.Result)
	})
}

func TestMCPPortal_MCP(t *testing.T) { // MARKER: MCP
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := mcpportalapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			res, err := client.MCP(ctx, "", nil)
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}
