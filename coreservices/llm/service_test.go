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

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/chatgptllm"
	"github.com/microbus-io/fabric/coreservices/chatgptllm/chatgptllmapi"
	"github.com/microbus-io/fabric/coreservices/claudellm"
	"github.com/microbus-io/fabric/coreservices/claudellm/claudellmapi"
	"github.com/microbus-io/fabric/coreservices/foreman"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/coreservices/geminillm"
	"github.com/microbus-io/fabric/coreservices/geminillm/geminillmapi"
	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/coreservices/llm/llmapi"
	"github.com/microbus-io/fabric/exampleservices/calculator"
	"github.com/microbus-io/fabric/exampleservices/calculator/calculatorapi"
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
	_ llmapi.Client
	_ json.Encoder
	_ httpegress.Service
	_ claudellm.Mock
	_ calculator.Service
)

// MARKER: Chat

func TestLLM_Chat(t *testing.T) { // MARKER: Chat
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)
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
		HINT: Use the following pattern for each test case

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
			result, usage, _, err := client.Chat(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
			assert.Expect(
				result, expectedResult,
				usage.Turns, 1,
				err, nil,
			)
		})
	*/
}

// TestLLM_ChatAnyResolution verifies that an empty/"any" provider is resolved via OnResolveProvider to whichever
// provider answers ok, using a mocked Claude provider so no live API key is needed.
func TestLLM_ChatAnyResolution(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	svc := NewService()

	// A configured Claude provider: it answers the resolve event and returns a canned turn.
	claudeMock := claudellm.NewMock()
	claudeMock.MockOnResolveProvider(func(ctx context.Context, model string) (ok bool, err error) {
		return true, nil
	})
	claudeMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (itemsOut []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
		return []llmapi.Item{llmapi.NewMessage("assistant", "Resolved!").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{Turns: 1, Model: "claude-sonnet-4-6"}, nil
	})

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(
		svc,
		claudeMock,
		tester,
	)
	app.RunInTest(t)

	t.Run("any resolves to the configured provider", func(t *testing.T) {
		assert := testarossa.For(t)
		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		result, _, resolvedProvider, err := client.Chat(ctx, "any", "default", items, nil, nil)
		assert.NoError(err)
		assert.Expect(resolvedProvider, claudellmapi.Hostname)
		assert.Expect(llmapi.LastAssistantMessage(result), "Resolved!")
	})
}

// TestLLM_ChatNoProvider verifies that when no provider answers the resolve event ok, Chat returns an error rather
// than silently falling back to a simulated provider.
func TestLLM_ChatNoProvider(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	svc := NewService()

	claudeMock := claudellm.NewMock()
	claudeMock.MockOnResolveProvider(func(ctx context.Context, model string) (ok bool, err error) {
		return false, nil
	})

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(
		svc,
		claudeMock,
		tester,
	)
	app.RunInTest(t)

	t.Run("no configured provider errors", func(t *testing.T) {
		assert := testarossa.For(t)
		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		_, _, _, err := client.Chat(ctx, "any", "default", items, nil, nil)
		assert.Error(err)
	})
}

func TestLLM_ChatLive(t *testing.T) {
	t.Skip("Manual test - set real API keys and comment this line to run")
	t.Parallel()
	ctx := t.Context()

	svc := NewService()

	claude := claudellm.NewService()
	claude.SetAPIKey("") // sk-...

	gemini := geminillm.NewService()
	gemini.SetAPIKey("") // AI...

	chatgpt := chatgptllm.NewService()
	chatgpt.SetAPIKey("") // sk-...

	egressSvc := httpegress.NewService()
	calcSvc := calculator.NewService()

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, claude, gemini, chatgpt, egressSvc, calcSvc, tester)
	app.RunInTest(t)

	t.Run("text_only_claude", func(t *testing.T) {
		assert := testarossa.For(t)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is the capital of France? Answer in one word.").AsItem()}
		result, _, _, err := client.Chat(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		if assert.NoError(err) {
			t.Log("Response:", result)
			assert.Expect(len(result) > 0, true)
		}
	})

	t.Run("text_only_gemini", func(t *testing.T) {
		assert := testarossa.For(t)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is the capital of Japan? Answer in one word.").AsItem()}
		result, _, _, err := client.Chat(ctx, geminillmapi.Hostname, "gemini-3.5-flash", items, nil, nil)
		if assert.NoError(err) {
			t.Log("Response:", result)
			assert.Expect(len(result) > 0, true)
		}
	})

	t.Run("text_only_chatgpt", func(t *testing.T) {
		assert := testarossa.For(t)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is the capital of Italy? Answer in one word.").AsItem()}
		result, _, _, err := client.Chat(ctx, chatgptllmapi.Hostname, "gpt-5.4-mini", items, nil, nil)
		if assert.NoError(err) {
			t.Log("Response:", result)
			assert.Expect(len(result) > 0, true)
		}
	})

	t.Run("with_tools", func(t *testing.T) {
		assert := testarossa.For(t)
		items := []llmapi.Item{llmapi.NewMessage("user", "What is 6 times 7? Use the calculator tool.").AsItem()}
		tools := []string{calculatorapi.Arithmetic.URL()}
		result, _, _, err := client.Chat(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, tools, nil)
		if assert.NoError(err) {
			t.Log("Response:", result)
			assert.Expect(len(result) > 0, true)
		}
	})
}

func TestLLM_ChatWithMockedProvider(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	providerMock := claudellm.NewMock()
	calcSvc := calculator.NewService()

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, providerMock, calcSvc, tester)
	app.RunInTest(t)

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			return []llmapi.Item{llmapi.NewMessage("assistant", "Hello from mocked provider!").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 5, OutputTokens: 5, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		result, usage, _, err := client.Chat(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		if assert.NoError(err) {
			// Chat returns the full conversation: the input user message plus the assistant reply.
			assert.Expect(len(result), 2)
			assert.Expect(result[0].Message.Role, "user")
			last := result[len(result)-1]
			assert.Expect(last.Message.Role, "assistant")
			assert.Expect(last.Message.Content, "Hello from mocked provider!")
			assert.Expect(usage.Turns, 1)
			assert.Expect(usage.OutputTokens, 5)
		}
	})

	t.Run("tool_calling", func(t *testing.T) {
		assert := testarossa.For(t)
		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			if callCount == 1 {
				return []llmapi.Item{llmapi.ToolCall{
					ID:        "call_1",
					Name:      "Arithmetic",
					Arguments: json.RawMessage(`{"x":3,"op":"+","y":5}`),
				}.AsItem()}, llmapi.StopReasonToolUse, llmapi.Usage{InputTokens: 10, OutputTokens: 5, Model: model, Turns: 1}, nil
			}
			return []llmapi.Item{llmapi.NewMessage("assistant", "3 + 5 = 8").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 15, OutputTokens: 8, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "What is 3 + 5?").AsItem()}
		tools := []string{calculatorapi.Arithmetic.URL()}
		result, usage, _, err := client.Chat(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, tools, nil)
		if assert.NoError(err) {
			assert.Expect(len(result) >= 2, true)
			last := result[len(result)-1]
			assert.Expect(last.Message.Role, "assistant")
			assert.Expect(last.Message.Content, "3 + 5 = 8")
			assert.Expect(usage.Turns, 2)
		}
	})
}

func TestLLM_ChatLoop(t *testing.T) { // MARKER: ChatLoop
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	providerMock := claudellm.NewMock()
	calcSvc := calculator.NewService()

	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)
	exec := llmapi.NewExecutor(tester).WithWorkflowRunner(foremanClient)

	app := application.New()
	app.Add(svc, providerMock, calcSvc, foreman.NewService(), tester)
	app.RunInTest(t)

	t.Run("text_response", func(t *testing.T) {
		assert := testarossa.For(t)

		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			return []llmapi.Item{llmapi.NewMessage("assistant", "Hello from the workflow!").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 5, OutputTokens: 5, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		out, usage, status, err := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		assert.Expect(
			err, nil,
			status, workflow.StatusCompleted,
			usage.Turns, 1,
			usage.OutputTokens, 5,
		)
		// Output messages = input history + assistant reply (the messages field is Append-reduced).
		if assert.Expect(len(out) >= 1, true) {
			last := out[len(out)-1]
			assert.Expect(last.Message.Role, "assistant")
			assert.Expect(last.Message.Content, "Hello from the workflow!")
		}
	})

	t.Run("tool_calling_loop", func(t *testing.T) {
		assert := testarossa.For(t)

		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			if callCount == 1 {
				// First turn: ask for a tool call.
				return []llmapi.Item{llmapi.ToolCall{
					ID:        "call_1",
					Name:      "Arithmetic",
					Arguments: json.RawMessage(`{"x":3,"op":"+","y":5}`),
				}.AsItem()}, llmapi.StopReasonToolUse, llmapi.Usage{InputTokens: 10, OutputTokens: 5, Model: model, Turns: 1}, nil
			}
			// Second turn (after fan-in at nextLLM): finalize.
			return []llmapi.Item{llmapi.NewMessage("assistant", "3 + 5 = 8").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 15, OutputTokens: 8, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		// ChatLoop's InitChat resolves URLs to tool schemas via the host's OpenAPI document,
		// so we pass the calculator's canonical URL and let InitChat fetch the schema.
		toolURLs := []string{calculatorapi.Arithmetic.URL()}
		items := []llmapi.Item{llmapi.NewMessage("user", "What is 3 + 5?").AsItem()}
		out, usage, status, err := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, toolURLs, nil)
		assert.Expect(
			err, nil,
			status, workflow.StatusCompleted,
			usage.Turns, 2,
		)
		// After two turns (with a fan-in round in between), the final assistant reply lands.
		if assert.Expect(len(out) >= 1, true) {
			last := out[len(out)-1]
			assert.Expect(last.Message.Role, "assistant")
			assert.Expect(last.Message.Content, "3 + 5 = 8")
		}
	})

	t.Run("round_limit_forces_final_toolless_call", func(t *testing.T) {
		assert := testarossa.For(t)

		// A model that always asks for another tool call while tools are offered. With MaxToolRounds=1
		// the loop must not end on a dangling tool_use: it executes the one round, then makes a final
		// call with no tools, forcing a text answer (mirroring the live Chat loop).
		var toollessCalls int
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			if len(tools) == 0 {
				toollessCalls++
				return []llmapi.Item{llmapi.NewMessage("assistant", "Final answer after limit").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 5, OutputTokens: 3, Model: model, Turns: 1}, nil
			}
			return []llmapi.Item{llmapi.ToolCall{
				ID:        "call_x",
				Name:      "Arithmetic",
				Arguments: json.RawMessage(`{"x":1,"op":"+","y":1}`),
			}.AsItem()}, llmapi.StopReasonToolUse, llmapi.Usage{InputTokens: 10, OutputTokens: 5, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		toolURLs := []string{calculatorapi.Arithmetic.URL()}
		items := []llmapi.Item{llmapi.NewMessage("user", "Keep calling tools").AsItem()}
		opts := &llmapi.ChatOptions{MaxToolRounds: 1}
		out, _, status, err := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, toolURLs, opts)
		assert.Expect(
			err, nil,
			status, workflow.StatusCompleted,
			toollessCalls, 1,
		)
		// The conversation ends on a text answer, not an unexecuted tool_use.
		if assert.Expect(len(out) >= 1, true) {
			last := out[len(out)-1]
			if assert.Expect(last.Type(), llmapi.ItemMessage) {
				assert.Expect(last.Message.Role, "assistant")
				assert.Expect(last.Message.Content, "Final answer after limit")
			}
		}
	})
}

// TestLLM_CallLLMRateLimitRetry verifies CallLLM's reactive backoff: a provider error carrying a retryAfter is
// retried (the property survives the bus round-trip and drives flow.Retry), bounded by the task's time budget -
// once the next wait would overshoot it the flow fails rather than looping forever. An error without a retryAfter
// is permanent. The horizon is the foreman's TimeBudget (min 1s), set small here so the test resolves quickly.
func TestLLM_CallLLMRateLimitRetry(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	providerMock := claudellm.NewMock()

	// The CallLLM retry horizon is the task's time budget. Pin it to the 1s floor so the exhaust path resolves
	// in ~1s instead of the production default.
	fmn := foreman.NewService()
	fmn.SetTimeBudget(time.Second)

	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)
	exec := llmapi.NewExecutor(tester).WithWorkflowRunner(foremanClient)

	app := application.New()
	app.Add(svc, providerMock, fmn, tester)
	app.RunInTest(t)

	t.Run("retries_then_succeeds", func(t *testing.T) {
		assert := testarossa.For(t)

		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			if callCount == 1 {
				return nil, "", llmapi.Usage{}, errors.New("rate limited", http.StatusTooManyRequests, "retryAfter", "200ms")
			}
			return []llmapi.Item{llmapi.NewMessage("assistant", "Recovered after backoff").AsItem()}, llmapi.StopReasonEndTurn, llmapi.Usage{InputTokens: 5, OutputTokens: 5, Model: model, Turns: 1}, nil
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		started := time.Now()
		out, _, status, err := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		elapsed := time.Since(started)
		assert.Expect(
			err, nil,
			status, workflow.StatusCompleted,
			callCount, 2,
		)
		// The retry must actually wait ~retryAfter before re-dispatching (guards against the engine dropping the
		// delay - the wait is carried in flow.Retry's initialDelay precisely so it is honored).
		assert.True(elapsed >= 150*time.Millisecond, "retry should wait ~retryAfter (200ms), waited %v", elapsed)
		if assert.Expect(len(out) >= 1, true) {
			last := out[len(out)-1]
			assert.Expect(last.Message.Content, "Recovered after backoff")
		}
	})

	t.Run("exhausts_and_fails", func(t *testing.T) {
		assert := testarossa.For(t)
		// retryAfter (200ms) is well under the 1s budget, so the call retries repeatedly and then fails once the
		// budget horizon is reached. It must have retried at least once before giving up.
		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			return nil, "", llmapi.Usage{}, errors.New("rate limited", http.StatusTooManyRequests, "retryAfter", "200ms")
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		_, _, status, _ := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		assert.Expect(status, workflow.StatusFailed)
		assert.True(callCount >= 2, "expected at least one retry before giving up, got %d calls", callCount)
	})

	t.Run("retry_after_exceeds_budget_fails_fast", func(t *testing.T) {
		assert := testarossa.For(t)
		// retryAfter (5s) exceeds the 1s budget, so the very first wait would overshoot the horizon: flow.Retry
		// gives up immediately rather than parking a doomed wait. One call, fast failure (no 5s park).
		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			return nil, "", llmapi.Usage{}, errors.New("rate limited", http.StatusTooManyRequests, "retryAfter", "5s")
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		started := time.Now()
		_, _, status, _ := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		elapsed := time.Since(started)
		assert.Expect(status, workflow.StatusFailed)
		assert.Expect(callCount, 1)
		assert.True(elapsed < 2*time.Second, "should fail fast without parking the 5s retryAfter, took %v", elapsed)
	})

	t.Run("no_retry_after_is_permanent", func(t *testing.T) {
		assert := testarossa.For(t)

		callCount := 0
		providerMock.MockTurn(func(ctx context.Context, model string, items []llmapi.Item, tools []llmapi.Tool, options *llmapi.TurnOptions) (outItems []llmapi.Item, stopReason string, usage llmapi.Usage, err error) {
			callCount++
			return nil, "", llmapi.Usage{}, errors.New("bad request", http.StatusBadRequest)
		})
		defer providerMock.MockTurn(nil)

		items := []llmapi.Item{llmapi.NewMessage("user", "Hello").AsItem()}
		_, _, status, _ := exec.ChatLoop(ctx, claudellmapi.Hostname, "claude-haiku-4-5", items, nil, nil)
		assert.Expect(status, workflow.StatusFailed)
		// No retryAfter => permanent => a single attempt, no retries.
		assert.Expect(callCount, 1)
	})
}

// TestLLM_ErrorProbe sends a ~1MB system prompt (well over 200k tokens) to each provider once and
// catalogs the shape of the resulting error. It is a manual investigation aid for the 429 / rate-limit
// classification work: paste real provider API keys below and enable with LLM_PROBE=1. The providers also
// log the raw status, headers, and body of every non-OK response, so run with -v to see both the
// normalized TracedError (statusCode / message / properties) and the raw upstream response.
//
// Do not commit real keys.
func TestLLM_ErrorProbe(t *testing.T) {
	if os.Getenv("LLM_PROBE") == "" {
		t.Skip("Live error-shape probe; paste provider API keys and set LLM_PROBE=1 to run")
	}
	ctx := t.Context()

	const (
		claudeKey  = "" // sk-ant-...
		geminiKey  = "" // AQ...
		chatgptKey = "" // sk-proj-...
	)

	svc := NewService()
	claude := claudellm.NewService()
	claude.SetAPIKey(claudeKey)
	gemini := geminillm.NewService()
	gemini.SetAPIKey(geminiKey)
	chatgpt := chatgptllm.NewService()
	chatgpt.SetAPIKey(chatgptKey)
	egressSvc := httpegress.NewService()

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, claude, gemini, chatgpt, egressSvc, tester)
	app.RunInTest(t)

	// System prompt sized per provider at ~4 chars/token. unit is 20 chars; 50k units = ~1MB = ~250k tokens
	// (over the 200k context of claude/gpt-4o). Gemini's window is ~1M tokens, so it needs ~6MB (~1.5M tokens).
	prompt := func(units int) string { return strings.Repeat("word word word word ", units) }

	providers := []struct {
		name, host, model, key string
		promptUnits            int
	}{
		{"claude", claudellmapi.Hostname, "claude-haiku-4-5", claudeKey, 50_000},
		{"gemini", geminillmapi.Hostname, "gemini-2.5-flash", geminiKey, 300_000},
		{"chatgpt", chatgptllmapi.Hostname, "gpt-5.4-mini", chatgptKey, 50_000},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			if p.key == "" {
				t.Skipf("no API key pasted for %s", p.name)
			}
			items := []llmapi.Item{
				llmapi.NewMessage("system", prompt(p.promptUnits)).AsItem(),
				llmapi.NewMessage("user", "Summarize the above in one word.").AsItem(),
			}
			_, usage, _, err := client.Chat(ctx, p.host, p.model, items, nil, nil)
			if err == nil {
				t.Logf("[%s] unexpected success; usage=%+v", p.name, usage)
				return
			}
			te := errors.Convert(err)
			t.Logf("\n===== %s =====\nstatusCode: %d\nmessage:    %s\nproperties: %#v",
				p.name, te.StatusCode, te.Error(), te.Properties)
		})
	}
}

// TestLLM_ErrorProbeGeminiBurst sends several large Gemini requests back-to-back to trip a genuine (transient)
// RESOURCE_EXHAUSTED: the burst overflows the per-minute window even though no single request exceeds the quota, so
// the geminillm parser must keep it retryable (a retryAfter is attached) rather than treat it as the single-oversized
// poison case. Each request is sized so its conservative estimate (~2 chars/token) still fits the quota; only the
// accumulated burst overflows. Enable with LLM_PROBE=1 and a pasted Gemini key. Run with -v.
func TestLLM_ErrorProbeGeminiBurst(t *testing.T) {
	if os.Getenv("LLM_PROBE") == "" {
		t.Skip("Live error-shape probe; paste a Gemini API key and set LLM_PROBE=1 to run")
	}
	ctx := t.Context()

	const geminiKey = "" // AQ...
	if geminiKey == "" {
		t.Skip("no Gemini API key pasted")
	}

	svc := NewService()
	gemini := geminillm.NewService()
	gemini.SetAPIKey(geminiKey)
	egressSvc := httpegress.NewService()

	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)

	app := application.New()
	app.Add(svc, gemini, egressSvc, tester)
	app.RunInTest(t)

	// At ~4 tokens per 20-char unit, 90k units is ~360k tokens (~1.8M chars). The conservative ~2 chars/token estimate
	// is ~900k < the 1M/min quota, so each request alone is classified transient (not poison); three back-to-back in
	// the same minute total ~1.08M > 1M, so the overflowing one trips a retryable RESOURCE_EXHAUSTED.
	burstPrompt := strings.Repeat("word word word word ", 90_000)

	for i := 1; i <= 3; i++ {
		items := []llmapi.Item{
			llmapi.NewMessage("system", burstPrompt).AsItem(),
			llmapi.NewMessage("user", "Summarize the above in one word.").AsItem(),
		}
		_, usage, _, err := client.Chat(ctx, geminillmapi.Hostname, "gemini-2.5-flash", items, nil, nil)
		if err == nil {
			t.Logf("\n===== request %d: SUCCESS =====\nusage=%+v", i, usage)
			continue
		}
		te := errors.Convert(err)
		t.Logf("\n===== request %d: ERROR =====\nstatusCode: %d\nmessage:    %s\nproperties: %#v",
			i, te.StatusCode, te.Error(), te.Properties)
	}
}

func TestLLM_OnResolveProvider(t *testing.T) { // MARKER: OnResolveProvider
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	trigger := llmapi.NewMulticastTrigger(tester)
	hook := llmapi.NewHook(tester)
	_ = trigger
	_ = hook

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Enter distinct alphanumeric queue names in sub.Queue when hooking multiple times to simulate multiple clients.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			unsub, err := hook.WithOptions(sub.Queue("UniqueQueueName")).OnResolveProvider(
				func(ctx context.Context, model string) (ok bool, err error) {
					// Implement event sink here...
					return ok, nil
				},
			)
			if assert.NoError(err) {
				defer unsub()
			}
			for e := range trigger.OnResolveProvider(ctx, model) {
				if frame.Of(e.HTTPResponse).FromHost() == tester.Hostname() {
					ok, err := e.Get()
					assert.Expect(
						ok, expectedOk,
						err, nil,
					)
				}
			}
		})
	*/
}

func TestLLM_Turn(t *testing.T) { // MARKER: Turn
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := llmapi.NewClient(tester)
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

			itemsOut, stopReason, usage, err := client.Turn(ctx, model, items, tools, options)
			assert.Expect(
				itemsOut, expectedItemsOut,
				stopReason, expectedStopReason,
				usage, expectedUsage,
				err, nil,
			)
		})
	*/
}

func TestLLM_InitChat(t *testing.T) { // MARKER: InitChat
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := llmapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := exec.InitChat(ctx, provider, model, items, toolURLs, options)
			assert.NoError(err)
		})
	*/
}

func TestLLM_CallLLM(t *testing.T) { // MARKER: CallLLM
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := llmapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			itemsOut, pendingToolCalls, turnUsage, err := exec.CallLLM(ctx, provider, model, items, toolResults)
			assert.Expect(
				itemsOut, expectedItemsOut,
				pendingToolCalls, expectedPendingToolCalls,
				turnUsage, expectedTurnUsage,
				err, nil,
			)
		})
	*/
}

func TestLLM_ProcessResponse(t *testing.T) { // MARKER: ProcessResponse
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := llmapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			toolsRequested, toolRoundsOut, usageOut, err := exec.ProcessResponse(ctx, pendingToolCalls, turnUsage, toolRounds)
			assert.Expect(
				toolsRequested, expectedToolsRequested,
				toolRoundsOut, expectedToolRoundsOut,
				usageOut, expectedUsageOut,
				err, nil,
			)
		})
	*/
}

func TestLLM_ExecuteTool(t *testing.T) { // MARKER: ExecuteTool
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := llmapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			toolResults, err := exec.ExecuteTool(ctx, currentTool)
			assert.Expect(
				toolResults, expectedToolResults,
				err, nil,
			)
		})
	*/
}
