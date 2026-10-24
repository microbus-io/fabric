## LLM Core Service

Create a core microservice at hostname `llm.core` that bridges LLM tool-calling with Microbus endpoint invocations. It is provider-agnostic: each call names the provider microservice (e.g. `claude.llm.core`) and the model, and `llm.core` delegates the actual LLM call to that provider's `Turn` endpoint.

### Config Properties

- `MaxToolRounds` - maximum number of tool-call round-trips per invocation, default `10`, range `[1, 50]`. A caller can override per call via `ChatOptions.MaxToolRounds`.

### Functional Endpoints (both on `:444`)

**`Turn(model string, items []Item, tools []Tool) (itemsOut []Item, stopReason string, usage Usage)`** - a single LLM turn. On `llm.core` this is a stub returning 501; the real implementation lives in each provider microservice (`claudellm`, `chatgptllm`, `geminillm`, `litellm`). The conversation is an ordered `[]Item` (message / tool_call / tool_result / reasoning); `Turn` returns only the assistant turn's new items.

**`Chat(provider, model string, items []Item, toolURLs []string, options *ChatOptions) (itemsOut []Item, usage Usage)`** - the full multi-turn chat loop with tool execution. `toolURLs` are canonical Microbus endpoint URLs (e.g. `calculatorapi.Arithmetic.URL()`), resolved to `[]llmapi.Tool` via `fetchTools`. It iterates up to `MaxToolRounds` turns; after each turn that returns tool calls, it executes each via `executeTool` and appends `tool_result` items. When no tool calls remain it returns; after exhausting `MaxToolRounds` it makes one final call without tools to force a text answer. `itemsOut` is the full conversation - the input `items` plus every item the LLM produced - so a caller can resume by re-invoking with it.

### Tool Resolution (`fetchTools`)

Group tool URLs by `host:port` (default port `443`). Fetch each host's OpenAPI document in parallel via `controlapi.NewClient(svc).ForHost(host).OpenAPI(ctx)` (the `:888/openapi.json` endpoint). For each requested URL, canonicalize path arguments (strip `...` from greedy args, name anonymous args `path1`, `path2`, ...) to match the OpenAPI document's path keys, then resolve the operation. Only `FeatureFunction`, `FeatureWeb`, and `FeatureWorkflow` operations are accepted. Disambiguate name collisions with `_2`, `_3`, ... suffixes in argument order.

### Tool Execution (`executeTool`)

Invoke the endpoint over the bus using `svc.Request` with the tool's URL, method, and JSON arguments as the body. On transport error, return the error as a JSON tool result rather than failing the whole chat. On HTTP 4xx/5xx, return the response body as a JSON error result.

### ChatLoop Workflow (on `:428`)

The `ChatLoop` workflow orchestrates multi-turn LLM conversations as durable workflow steps:

```
InitChat -> FirstLLM -> ProcessResponse
                             |- (no tool calls, or round limit) -> END
                             |- forEach(pendingToolCalls, currentTool) -> ExecuteTool -> NextLLM -> ProcessResponse
```

`FirstLLM` and `NextLLM` are two graph positions dispatching to the same `CallLLM` task (the split lets `NextLLM` be the fan-in nexus). A `ReducerAppend` reducer on `toolResults` merges the parallel `ExecuteTool` branches at the fan-in, and a second one on `items` accumulates the conversation. `CallLLM` is the sole writer of `items`: it folds the assembled `toolResults` into the conversation, calls the provider, and contributes only the folded tool results and the new turn to `items`.

**Task endpoints (all on `:428`):**

- `InitChat(items []Item, toolURLs []string, options *ChatOptions)` - a pure setup step with no outputs; resolves `toolURLs` to tool schemas via each host's OpenAPI document and seeds the ambient flow state the rest of the loop reads - `toolSchemas`, `turnOptions`, `maxToolRounds`, and the `toolRounds` counter (0) - all via `flow.Set`.
- `CallLLM(items []Item, toolResults []ToolResult) (itemsOut []Item, pendingToolCalls []ToolCall, turnUsage Usage)` - folds `toolResults` (the fan-in's assembled results) into the conversation and clears the key, reads `toolSchemas`/`turnOptions`/`finalCall` ambiently, calls the provider, and contributes the folded tool results and the new turn to `items` (json:"items"). On a rate-limit retry it rewinds by returning nil `items`.
- `ProcessResponse(pendingToolCalls []ToolCall, turnUsage Usage, toolRounds int) (toolsRequested bool, toolRoundsOut int, usageOut Usage)` - accumulates usage and routes; it does not touch `items`. If no pending calls or the round limit was already reached, sets `toolsRequested=false` and `flow.Goto(workflow.End)`. Otherwise increments `toolRoundsOut`, sets `toolsRequested=true` to trigger the `forEach` over `pendingToolCalls`, and on the last permitted round sets the ambient `finalCall` flag so the next `CallLLM` omits tools and forces a text answer.
- `ExecuteTool(currentTool ToolCall) (toolResults []ToolResult)` - executes the single tool call named by the `currentTool` forEach variable. Workflow tools run as dynamic subgraphs via `flow.Subgraph` (parks on the first call, returns the child's final state on re-entry); other tools run via a direct bus call. Returns a single-element `toolResults` delta, which the append reducer merges into the `toolResults` state key.
