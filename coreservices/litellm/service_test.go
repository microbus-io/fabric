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

package litellm

import (
	"bufio"
	"context"
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

	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/coreservices/litellm/litellmapi"
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
	_ litellmapi.Client
	_ llmapi.Message
	_ bufio.Reader
	_ strings.Builder
	_ httpegress.Mock
)

// MARKER: Turn

func TestLiteLLM_Turn(t *testing.T) { // MARKER: Turn
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	httpEgressMock := httpegress.NewMock()

	tester := connector.New("tester.client")
	client := litellmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, httpEgressMock, tester)
	app.RunInTest(t)

	svc.SetAPIKey("test-key")

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if strings.Contains(req.URL.String(), "/v1/responses") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello from LiteLLM!"}]}],"status":"completed","model":"gpt-4o","usage":{"input_tokens":10,"output_tokens":5}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		out, stopReason, usage, err := client.Turn(ctx, "gpt-4o", items, nil, nil)
		if assert.NoError(err) {
			assert.Expect(llmapi.LastAssistantMessage(out), "Hello from LiteLLM!")
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
			if strings.Contains(req.URL.String(), "/v1/responses") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"output":[{"type":"function_call","call_id":"call_1","name":"Arithmetic","arguments":"{\"x\":10,\"op\":\"-\",\"y\":3}"}],"status":"completed","model":"gpt-4o","usage":{"input_tokens":15,"output_tokens":8}}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})
		defer httpEgressMock.MockMakeRequest(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is 10 - 3?").AsItem()}
		out, stopReason, _, err := client.Turn(ctx, "gpt-4o", items, nil, nil)
		if assert.NoError(err) {
			calls := llmapi.PendingToolCalls(out)
			assert.Expect(len(calls), 1)
			assert.Expect(calls[0].Name, "Arithmetic")
			assert.Expect(stopReason, llmapi.StopReasonToolUse)
		}
	})
}

func TestLiteLLM_OnResolveProvider(t *testing.T) { // MARKER: OnResolveProvider
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

func TestLiteLLM_RefreshModels(t *testing.T) { // MARKER: RefreshModels
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
