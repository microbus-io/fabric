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

package llmapi

import (
	"github.com/microbus-io/fabric/define"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "llm.core"

// Name is the decorative PascalCase name of the microservice.
const Name = "LLM"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 16

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `The LLM microservice bridges LLM tool-calling protocols with Microbus endpoint invocations.`

// ProviderAny is the provider sentinel that auto-selects any configured provider. Pass it (equivalent to the
// empty string) as the provider argument to Chat or ChatLoop to let llm.core resolve a provider that holds a key.
const ProviderAny = "any"

// Generic capability-tier model aliases. A caller passes one of these (or a provider family alias like opus, or a
// concrete model name) as the model argument; the resolved provider maps it to a concrete model. ModelDefault is
// the fallback when no model is given.
const (
	ModelFast    = "fast"
	ModelDefault = "default"
	ModelSmart   = "smart"
)

// MaxToolRounds is the maximum number of tool call round-trips per invocation.
var MaxToolRounds = define.Config{ // MARKER: MaxToolRounds
	Value:      int(0),
	Default:    "10",
	Validation: "int [1,50]",
}

// LLMTokens counts tokens consumed per LLM turn
var LLMTokens = define.Metric{ // MARKER: LLMTokens
	Kind: define.Counter, Value: int(0), Labels: []string{"provider", "model", "direction"},
	OTelName: "microbus_llm_tokens",
}

// ToolCalls counts tool invocations from the chat loop, labeled by the tool's URL, its feature type (function/web/workflow), and the outcome (ok/error)
var ToolCalls = define.Metric{ // MARKER: ToolCalls
	Kind: define.Counter, Value: int(0), Labels: []string{"tool_url", "tool_type", "outcome"},
	OTelName: "microbus_llm_tool_calls",
}

// OnResolveProvider is fired by llm.core to resolve which provider serves a given model alias or name. Each provider microservice sinks it and answers ok=true when it is configured (holds an API key) and its catalog recognizes the model. llm.core reads the responder's hostname from the response frame and routes the turn there; the simulated chatbox provider does not subscribe, so it never participates in resolution.
var OnResolveProvider = define.OutboundEvent{ // MARKER: OnResolveProvider
	Host: Hostname, Method: "POST", Route: ":417/on-resolve-provider",
	In: OnResolveProviderIn{}, Out: OnResolveProviderOut{},
}

// OnResolveProviderIn are the input arguments of OnResolveProvider.
type OnResolveProviderIn struct { // MARKER: OnResolveProvider
	Model string `json:"model,omitzero"`
}

// OnResolveProviderOut are the output arguments of OnResolveProvider.
type OnResolveProviderOut struct { // MARKER: OnResolveProvider
	OK bool `json:"ok,omitzero"`
}

// Chat sends a conversation to an LLM with optional tools, looping through tool calls until the LLM returns a final answer. Provider is a provider microservice hostname (e.g. claude.llm.core); pass it empty or "any" to auto-select a configured provider. Model is a capability-tier alias (fast, default, smart), a provider family alias (e.g. opus), or a concrete model name; empty means default. Each toolURL is the canonical URL of a Microbus Function, Web, or Workflow endpoint exposed to the LLM; Chat fetches each host's OpenAPI document and resolves the URL to a callable tool. On error it still returns the items accumulated before the failure, so a caller running its own retry can resume from them (e.g. wait llmapi.RetryAfter(err) and re-call with the returned items) instead of restarting the conversation.
var Chat = define.Function{ // MARKER: Chat
	Host: Hostname, Method: "POST", Route: ":444/chat",
	In: ChatIn{}, Out: ChatOut{},
}

// ChatIn are the input arguments of Chat.
type ChatIn struct { // MARKER: Chat
	Provider string       `json:"provider,omitzero" jsonschema_description:"provider is the LLM provider microservice hostname; empty or any auto-selects a configured provider"`
	Model    string       `json:"model,omitzero" jsonschema_description:"model is a tier alias (fast, default, smart), a provider family alias, or a concrete model name; empty means default"`
	Items    []Item       `json:"items,omitzero" jsonschema_description:"items is the conversation history to send to the LLM"`
	ToolURLs []string     `json:"toolURLs,omitzero" jsonschema_description:"toolURLs is the list of Microbus endpoint URLs exposed to the LLM"`
	Options  *ChatOptions `json:"options,omitzero" jsonschema_description:"options configures tool-call rounds, max tokens, temperature (nil = defaults)"`
}

// ChatOut are the output arguments of Chat.
type ChatOut struct { // MARKER: Chat
	ItemsOut         []Item `json:"items,omitzero" jsonschema_description:"items is the full conversation including new items produced by the LLM"`
	Usage            Usage  `json:"usage,omitzero" jsonschema_description:"usage is the aggregate token consumption across all turns"`
	ResolvedProvider string `json:"resolvedProvider,omitzero" jsonschema_description:"resolvedProvider is the provider hostname that served the conversation, resolved when the caller passed an empty or any provider"`
}

// Turn executes a single LLM turn. On llm.core it is a stub returning 501 Not Implemented; the actual implementation lives in each provider microservice (claudellm, chatgptllm, geminillm). Call ForHost(<providerHostname>).Turn to reach a specific provider directly, or use Chat for the full conversation loop.
var Turn = define.Function{ // MARKER: Turn
	Host: Hostname, Method: "POST", Route: ":444/turn",
	In: TurnIn{}, Out: TurnOut{},
}

// TurnIn are the input arguments of Turn.
type TurnIn struct { // MARKER: Turn
	Model   string       `json:"model,omitzero" jsonschema_description:"model is the provider-specific model identifier"`
	Items   []Item       `json:"items,omitzero" jsonschema_description:"items is the conversation history to send to the LLM"`
	Tools   []Tool       `json:"tools,omitzero" jsonschema_description:"tools is the resolved tool definitions with schemas"`
	Options *TurnOptions `json:"options,omitzero" jsonschema_description:"options configures max tokens and temperature (nil = provider defaults)"`
}

// TurnOut are the output arguments of Turn.
type TurnOut struct { // MARKER: Turn
	ItemsOut   []Item `json:"items,omitzero" jsonschema_description:"items is the LLM's response turn: reasoning, message, and tool_call items in order"`
	StopReason string `json:"stopReason,omitzero"`
	Usage      Usage  `json:"usage,omitzero" jsonschema_description:"usage is the token consumption for this single turn"`
}

// InitChat resolves caller-supplied tool URLs into LLM tool schemas via each host's OpenAPI document and stores them, along with chat options, in flow state for use by the rest of the chat loop.
var InitChat = define.Task{ // MARKER: InitChat
	Host: Hostname, Method: "POST", Route: ":428/init-chat",
	In: InitChatIn{}, Out: InitChatOut{},
}

// InitChatIn are the input arguments of InitChat.
type InitChatIn struct { // MARKER: InitChat
	Provider string       `json:"provider,omitzero" jsonschema_description:"provider is the LLM provider microservice hostname; empty or any auto-selects a configured provider"`
	Model    string       `json:"model,omitzero" jsonschema_description:"model is a tier alias (fast, default, smart), a provider family alias, or a concrete model name; empty means default"`
	Items    []Item       `json:"items,omitzero" jsonschema_description:"items is the initial conversation history sent to the LLM"`
	ToolURLs []string     `json:"toolURLs,omitzero" jsonschema_description:"toolURLs is the list of Microbus endpoint URLs exposed to the LLM"`
	Options  *ChatOptions `json:"options,omitzero" jsonschema_description:"options configures tool-call rounds, max tokens, temperature (nil = defaults)"`
}

// InitChatOut are the output arguments of InitChat. InitChat is a pure setup step - it seeds ambient
// flow state (toolSchemas, turnOptions, maxToolRounds, toolRounds) via flow.Set and declares no outputs.
type InitChatOut struct { // MARKER: InitChat
}

// CallLLM is the sole writer of the items conversation state key. It folds the prior round's tool
// results (accumulated in the toolResults key by the fan-in) into the conversation, calls the provider,
// and contributes the folded tool results and the new turn to items, whose append reducer adds them to the
// conversation.
var CallLLM = define.Task{ // MARKER: CallLLM
	Host: Hostname, Method: "POST", Route: ":428/call-llm",
	In: CallLLMIn{}, Out: CallLLMOut{},
}

// CallLLMIn are the input arguments of CallLLM.
type CallLLMIn struct { // MARKER: CallLLM
	Provider    string       `json:"provider,omitzero"`
	Model       string       `json:"model,omitzero"`
	Items       []Item       `json:"items,omitzero"`
	ToolResults []ToolResult `json:"toolResults,omitzero"`
}

// CallLLMOut are the output arguments of CallLLM. ItemsOut contributes the folded tool results and the new
// turn to the Append-reduced items state key; PendingToolCalls drives the forEach fan-out.
type CallLLMOut struct { // MARKER: CallLLM
	ItemsOut         []Item     `json:"items,omitzero"`
	PendingToolCalls []ToolCall `json:"pendingToolCalls,omitzero"`
	TurnUsage        Usage      `json:"turnUsage,omitzero"`
}

// ProcessResponse accumulates usage and routes the loop: it fans out one ExecuteTool per pending tool
// call, or ends the loop. It does not write the conversation - CallLLM owns the items state key.
var ProcessResponse = define.Task{ // MARKER: ProcessResponse
	Host: Hostname, Method: "POST", Route: ":428/process-response",
	In: ProcessResponseIn{}, Out: ProcessResponseOut{},
}

// ProcessResponseIn are the input arguments of ProcessResponse.
type ProcessResponseIn struct { // MARKER: ProcessResponse
	PendingToolCalls []ToolCall `json:"pendingToolCalls,omitzero"`
	TurnUsage        Usage      `json:"turnUsage,omitzero"`
	ToolRounds       int        `json:"toolRounds,omitzero"`
}

// ProcessResponseOut are the output arguments of ProcessResponse. ProcessResponse routes the loop and
// accumulates usage; it does not touch the items state key, which CallLLM owns. The conversation
// therefore survives to ChatLoop's terminal output (ChatLoopOut.ItemsOut) as CallLLM last wrote it.
type ProcessResponseOut struct { // MARKER: ProcessResponse
	ToolsRequested bool  `json:"toolsRequested,omitzero"`
	ToolRoundsOut  int   `json:"toolRounds,omitzero"`
	UsageOut       Usage `json:"usage,omitzero"`
}

// ExecuteTool executes a single tool call, identified by the currentTool forEach variable, and returns
// its result. The results of all fanned-out ExecuteTool branches are Append-reduced into the
// toolResults state key, which the next CallLLM folds into the conversation.
var ExecuteTool = define.Task{ // MARKER: ExecuteTool
	Host: Hostname, Method: "POST", Route: ":428/execute-tool",
	In: ExecuteToolIn{}, Out: ExecuteToolOut{},
}

// ExecuteToolIn are the input arguments of ExecuteTool.
type ExecuteToolIn struct { // MARKER: ExecuteTool
	CurrentTool ToolCall `json:"currentTool,omitzero"`
}

// ExecuteToolOut are the output arguments of ExecuteTool. ToolResults is a single-element delta that the
// fan-in appends into the toolResults state key (Append-reduced).
type ExecuteToolOut struct { // MARKER: ExecuteTool
	ToolResults []ToolResult `json:"toolResults,omitzero"`
}

// ChatLoop defines the workflow graph for multi-turn LLM conversations with tool calling.
var ChatLoop = define.Workflow{ // MARKER: ChatLoop
	Host: Hostname, Method: "GET", Route: ":428/chat-loop",
	In: ChatLoopIn{}, Out: ChatLoopOut{},
}

// ChatLoopIn are the input arguments of ChatLoop.
type ChatLoopIn struct { // MARKER: ChatLoop
	Provider string       `json:"provider,omitzero"`
	Model    string       `json:"model,omitzero"`
	Items    []Item       `json:"items,omitzero"`
	ToolURLs []string     `json:"toolURLs,omitzero"`
	Options  *ChatOptions `json:"options,omitzero"`
}

// ChatLoopOut are the output arguments of ChatLoop.
type ChatLoopOut struct { // MARKER: ChatLoop
	ItemsOut []Item `json:"items,omitzero"`
	Usage    Usage  `json:"usage,omitzero"`
}
