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

package geminillm

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

	"github.com/microbus-io/fabric/coreservices/geminillm/geminillmapi"
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
	_ geminillmapi.Client
	_ llmapi.Message
	_ bufio.Reader
	_ strings.Builder
	_ httpegress.Mock
)

// MARKER: Turn

func TestGeminiLLM_Turn(t *testing.T) { // MARKER: Turn
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	httpEgressMock := httpegress.NewMock()

	tester := connector.New("tester.client")
	client := geminillmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, httpEgressMock, tester)
	app.RunInTest(t)

	svc.SetAPIKey("test-key")

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello from Gemini!"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.0-flash","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		out, stopReason, usage, err := client.Turn(ctx, "gemini-3.5-flash", items, nil, nil)
		if assert.NoError(err) {
			assert.Expect(llmapi.LastAssistantMessage(out), "Hello from Gemini!")
			assert.Expect(len(llmapi.PendingToolCalls(out)), 0)
			assert.Expect(stopReason, llmapi.StopReasonEndTurn)
			assert.Expect(usage.OutputTokens, 5)
			assert.Expect(usage.Turns, 1)
		}
	})

	t.Run("tool_calling", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"Arithmetic","args":{"x":7,"op":"*","y":6}}}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.0-flash","usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":8}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is 7 * 6?").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, "gemini-3.5-flash", items, nil, nil)
		if assert.NoError(err) {
			calls := llmapi.PendingToolCalls(out)
			assert.Expect(len(calls), 1)
			assert.Expect(calls[0].Name, "Arithmetic")
			assert.Expect(stopReason, llmapi.StopReasonToolUse)
		}
	})

	t.Run("system_instruction_routing", func(t *testing.T) {
		assert := testarossa.For(t)

		var capturedBody []byte
		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				capturedBody, _ = io.ReadAll(req.Body)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.0-flash","usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{
			llmapi.NewMessage("system", "You are terse.").AsItem(),
			llmapi.NewMessage("system", "Reply in one word.").AsItem(),
			llmapi.NewMessage("user", "Hello").AsItem(),
		}
		_, _, _, err := client.Turn(ctx, "gemini-3.5-flash", items, nil, nil)
		if !assert.NoError(err) {
			return
		}
		var sent map[string]any
		assert.NoError(json.Unmarshal(capturedBody, &sent))
		// System messages hoisted into systemInstruction (concatenated with blank lines).
		si, ok := sent["systemInstruction"].(map[string]any)
		assert.Expect(ok, true)
		siParts, _ := si["parts"].([]any)
		if assert.Expect(len(siParts), 1) {
			text, _ := siParts[0].(map[string]any)["text"].(string)
			assert.Expect(text, "You are terse.\n\nReply in one word.")
		}
		// contents must NOT carry a system-as-user duplicate.
		contents, _ := sent["contents"].([]any)
		assert.Expect(len(contents), 1)
		first, _ := contents[0].(map[string]any)
		assert.Expect(first["role"], "user")
	})

	t.Run("thought_signature_round_trip", func(t *testing.T) {
		assert := testarossa.For(t)

		// Turn 1: the model returns a function call carrying a thoughtSignature, and a
		// preceding thought part that must be ignored from the visible content.
		var firstCall = true
		var secondCallBody []byte
		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if !strings.Contains(req.URL.String(), "generateContent") {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}
			w.Header().Set("Content-Type", "application/json")
			if firstCall {
				firstCall = false
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[` +
					`{"text":"deliberating...","thought":true},` +
					`{"functionCall":{"name":"Lookup","args":{"q":"foo"}},"thoughtSignature":"SIG-ABC"}` +
					`]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":7}}`))
			} else {
				secondCallBody, _ = io.ReadAll(req.Body)
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Final answer"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":3}}`))
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Look it up").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, "gemini-2.5-flash", items, nil, nil)
		if !assert.NoError(err) {
			return
		}
		// Thought text suppressed from the visible content; function call extracted, and the signature
		// split out into a reasoning item positioned immediately before the tool call.
		assert.Expect(llmapi.LastAssistantMessage(out), "")
		assert.Expect(stopReason, llmapi.StopReasonToolUse)
		calls := llmapi.PendingToolCalls(out)
		if assert.Expect(len(calls), 1) {
			assert.Expect(calls[0].Name, "Lookup")
		}
		var foundReasoningSig string
		for _, it := range out {
			if it.Type() == llmapi.ItemReasoning && it.Reasoning != nil && it.Reasoning.Signature != "" {
				foundReasoningSig = it.Reasoning.Signature
			}
		}
		assert.Expect(foundReasoningSig, "SIG-ABC")

		// Round 2: hand the assistant turn (reasoning items included) back plus a tool result. The
		// provider must re-glue the thoughtSignature onto the functionCall part it re-emits, otherwise
		// Gemini 2.5 loses thinking continuity.
		round2 := []llmapi.Item{llmapi.NewMessage("user", "Look it up").AsItem()}
		round2 = append(round2, out...)
		round2 = append(round2, llmapi.NewToolResult("Lookup", `{"result":"42"}`).AsItem())
		_, _, _, err = client.Turn(ctx, "gemini-2.5-flash", round2, nil, nil)
		if !assert.NoError(err) {
			return
		}
		var sent map[string]any
		assert.NoError(json.Unmarshal(secondCallBody, &sent))
		contents, _ := sent["contents"].([]any)
		// Find the model-role content and verify its functionCall part carries the signature.
		var foundSig string
		for _, c := range contents {
			m, _ := c.(map[string]any)
			if m["role"] != "model" {
				continue
			}
			parts, _ := m["parts"].([]any)
			for _, p := range parts {
				pm, _ := p.(map[string]any)
				if _, isFunc := pm["functionCall"]; !isFunc {
					continue
				}
				if sig, ok := pm["thoughtSignature"].(string); ok {
					foundSig = sig
				}
			}
		}
		assert.Expect(foundSig, "SIG-ABC")
	})

	t.Run("thinking_tokens_in_usage", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				w.Header().Set("Content-Type", "application/json")
				// 2.5 model returns thoughtsTokenCount separately from candidatesTokenCount.
				// OutputTokens must equal the sum (60+250); ReasoningTokens carries the breakdown.
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"answer"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":60,"thoughtsTokenCount":250}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "hi").AsItem()}
		_, _, usage, err := client.Turn(ctx, "gemini-2.5-flash", items, nil, nil)
		if assert.NoError(err) {
			assert.Expect(usage.InputTokens, 100)
			assert.Expect(usage.OutputTokens, 310) // candidates + thoughts
			assert.Expect(usage.ReasoningTokens, 250)
		}
	})

	t.Run("attachments_outbound", func(t *testing.T) {
		assert := testarossa.For(t)

		var capturedBody []byte
		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				capturedBody, _ = io.ReadAll(req.Body)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"saw it"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":2}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		imageBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a} // PNG magic
		items := []llmapi.Item{{
			Message: &llmapi.Message{
				Role:    "user",
				Content: "What do you see?",
				Attachments: []llmapi.Attachment{
					{MediaType: "image/png", Data: imageBytes},
					{MediaType: "application/pdf", URI: "https://generativelanguage.googleapis.com/v1beta/files/abc-123"},
				},
			},
		}}
		out, _, _, err := client.Turn(ctx, "gemini-2.5-flash", items, nil, nil)
		if !assert.NoError(err) {
			return
		}
		assert.Expect(llmapi.LastAssistantMessage(out), "saw it")

		var sent map[string]any
		assert.NoError(json.Unmarshal(capturedBody, &sent))
		contents, _ := sent["contents"].([]any)
		if !assert.Expect(len(contents), 1) {
			return
		}
		parts, _ := contents[0].(map[string]any)["parts"].([]any)
		// text + inlineData + fileData
		if !assert.Expect(len(parts), 3) {
			return
		}
		// inlineData: base64 of the PNG bytes
		inline, _ := parts[1].(map[string]any)["inlineData"].(map[string]any)
		assert.Expect(inline["mimeType"], "image/png")
		// fileData
		file, _ := parts[2].(map[string]any)["fileData"].(map[string]any)
		assert.Expect(file["mimeType"], "application/pdf")
		assert.Expect(file["fileUri"], "https://generativelanguage.googleapis.com/v1beta/files/abc-123")
	})

	t.Run("empty_response_warn_no_error", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "generateContent") {
				w.Header().Set("Content-Type", "application/json")
				// Empty parts array with finishReason STOP - the smoking-gun case.
				w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":0}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, "gemini-2.5-flash", items, nil, nil)
		if assert.NoError(err) {
			// Empty response surfaces as end_turn with empty content; caller decides what to do.
			// The warn log is the diagnostic; we just verify the function completes cleanly.
			assert.Expect(llmapi.LastAssistantMessage(out), "")
			assert.Expect(len(llmapi.PendingToolCalls(out)), 0)
			assert.Expect(stopReason, llmapi.StopReasonEndTurn)
		}
	})
}

// TestGeminiLLM_RealTurn exercises the real Gemini generateContent API end-to-end through the live HTTP
// egress proxy. It is skipped by default; drop an API key into realAPIKey and remove the skip to run
// it against production.
func TestGeminiLLM_RealTurn(t *testing.T) {
	const realAPIKey = ""
	if realAPIKey == "" {
		t.Skip("set realAPIKey to run against the live Gemini generateContent API")
	}
	const realModel = "gemini-2.5-flash"

	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	client := geminillmapi.NewClient(tester)

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

				// Round-trip: append the assistant turn items verbatim (thought-signature reasoning
				// items included), then the tool result, and confirm the functionCall /
				// functionResponse parts thread through and the thinking model accepts the replay.
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

func TestGeminiLLM_RefreshModels(t *testing.T) { // MARKER: RefreshModels
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

func TestGeminiLLM_OnResolveProvider(t *testing.T) { // MARKER: OnResolveProvider
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
