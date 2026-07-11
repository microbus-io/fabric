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

package claudellm

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/claudellm/claudellmapi"
	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/coreservices/llm/llmapi"
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
	_ *workflow.Flow
	_ testarossa.Asserter
	_ claudellmapi.Client
	_ llmapi.Message
	_ json.Encoder
	_ strings.Builder
	_ bufio.Reader
	_ httpegress.Mock
)

// MARKER: Turn

func TestClaudeLLM_Turn(t *testing.T) { // MARKER: Turn
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	httpEgressMock := httpegress.NewMock()

	tester := connector.New("tester.client")
	client := claudellmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, httpEgressMock, tester)
	app.RunInTest(t)

	svc.SetAPIKey("test-key")

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "/v1/messages") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"content":[{"type":"text","text":"Hello from Claude!"}],"stop_reason":"end_turn","model":"claude-haiku-4-5","usage":{"input_tokens":10,"output_tokens":5}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		out, stopReason, usage, err := client.Turn(ctx, "claude-haiku-4-5", items, nil, nil)
		if assert.NoError(err) {
			assert.Expect(llmapi.LastAssistantMessage(out), "Hello from Claude!")
			assert.Expect(len(llmapi.PendingToolCalls(out)), 0)
			assert.Expect(stopReason, llmapi.StopReasonEndTurn)
			assert.Expect(usage.InputTokens, 10)
			assert.Expect(usage.OutputTokens, 5)
			assert.Expect(usage.Turns, 1)
		}
	})

	t.Run("tool_calling", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "/v1/messages") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"content":[{"type":"tool_use","id":"toolu_1","name":"Arithmetic","input":{"x":3,"op":"+","y":5}}],"stop_reason":"tool_use","model":"claude-haiku-4-5","usage":{"input_tokens":15,"output_tokens":8}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is 3 + 5?").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, "claude-haiku-4-5", items, nil, nil)
		if assert.NoError(err) {
			calls := llmapi.PendingToolCalls(out)
			assert.Expect(len(calls), 1)
			assert.Expect(calls[0].Name, "Arithmetic")
			assert.Expect(stopReason, llmapi.StopReasonToolUse)
		}
	})
}

// TestClaudeLLM_RealTurn exercises the real Anthropic Messages API end-to-end through the live HTTP
// egress proxy. It is skipped by default; drop an API key into realAPIKey and remove the skip to run
// it against production.
func TestClaudeLLM_RealTurn(t *testing.T) {
	const realAPIKey = ""
	if realAPIKey == "" {
		t.Skip("set realAPIKey to run against the live Anthropic Messages API")
	}
	const realModel = "claude-haiku-4-5"

	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	client := claudellmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, httpegress.NewService(), tester)
	app.RunInTest(t)

	svc.SetAPIKey(realAPIKey)

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		items := []llmapi.Item{
			llmapi.NewMessage("system", "You are a terse assistant. Answer in one word.").AsItem(),
			llmapi.NewMessage("user", "What is the capital of France?").AsItem(),
		}
		out, stopReason, usage, err := client.Turn(ctx, realModel, items, nil, nil)
		if assert.NoError(err) {
			assert.True(strings.Contains(strings.ToLower(llmapi.LastAssistantMessage(out)), "paris"))
			assert.Expect(len(llmapi.PendingToolCalls(out)), 0)
			assert.Expect(stopReason, llmapi.StopReasonEndTurn)
			assert.True(usage.InputTokens > 0)
			assert.True(usage.OutputTokens > 0)
			assert.Expect(usage.Turns, 1)
		}
	})

	t.Run("tool_calling", func(t *testing.T) {
		assert := testarossa.For(t)

		tools := []llmapi.Tool{{
			Name:        "Arithmetic",
			Description: "Computes the result of an arithmetic operation.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"number"},"op":{"type":"string"},"y":{"type":"number"}},"required":["x","op","y"]}`),
		}}
		items := []llmapi.Item{llmapi.NewMessage("user", "Use the Arithmetic tool to compute 10 - 3.").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, realModel, items, tools, nil)
		if assert.NoError(err) {
			assert.Expect(stopReason, llmapi.StopReasonToolUse)
			calls := llmapi.PendingToolCalls(out)
			if assert.True(len(calls) > 0) {
				assert.Expect(calls[0].Name, "Arithmetic")

				// Round-trip: append the assistant turn items verbatim, then the tool result, and
				// confirm the tool_use / tool_result blocks thread through correctly.
				items = append(items, out...)
				items = append(items, llmapi.NewToolResult(calls[0].ID, `{"result":7}`).AsItem())
				out, stopReason, _, err = client.Turn(ctx, realModel, items, tools, nil)
				if assert.NoError(err) {
					assert.Expect(stopReason, llmapi.StopReasonEndTurn)
					assert.True(strings.Contains(llmapi.LastAssistantMessage(out), "7"))
				}
			}
		}
	})
}

func TestClaudeLLM_RefreshModels(t *testing.T) { // MARKER: RefreshModels
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.RefreshModels(ctx)
			assert.NoError(err)
		})
	*/
}

func TestClaudeLLM_OnResolveProvider(t *testing.T) { // MARKER: OnResolveProvider
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := llmapi.NewMulticastTrigger(tester)
	_ = trigger

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

			for e := range trigger.OnResolveProvider(ctx, model) {
				ok, err := e.Get()
				if frame.Of(e.HTTPResponse).FromHost() == svc.Hostname() {
					assert.Expect(
						ok, expectedOk,
						err, nil,
					)
				}
			}
		})
	*/
}
