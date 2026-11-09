## MCP Portal Core Service

Create a core microservice at hostname `mcp.core` that exposes the bus's tools to LLM clients via the Model Context Protocol (MCP). The directory is named `mcpportal` to avoid clashing with a potential `mcp` package name.

Expose one HTTP wire endpoint:

- `MCP` on `POST //mcp:0` - accepts JSON-RPC 2.0 envelopes and dispatches on `method`.

The `:0` wildcard means the MCP server accepts connections on any ingress port. The `//` prefix makes the path root-relative, so the external URL is `https://ingress.host/mcp` regardless of which port the client connects on.

### JSON-RPC Dispatch

Read the entire request body, unmarshal as `jsonrpcRequest` (preserving `id` and `params` as `json.RawMessage` to echo IDs back unchanged and defer param decoding to each handler). Dispatch on `method`:

- `initialize` - respond with `protocolVersion` (`"2024-11-05"`), `capabilities: {tools: {}}`, and `serverInfo: {name: "Microbus MCP Portal", version: <connector version>}`.
- `tools/list` - fetch the OpenAPI portal aggregate and convert each function and web operation to an MCP tool descriptor (name, description, inputSchema). Tasks and workflows are not exposed: an MCP tool call is synchronous and a workflow is not.
- `tools/call` - re-fetch the aggregate, locate the named tool, reconstruct its bus URL as `"https:/" + pathKey`, POST the arguments as JSON, return the response body as `content[0].text`.
- `ping` - respond with `{}`.
- Notifications (no `id`, method prefix `notifications/`) - respond with HTTP 200 and no body.
- Unknown methods - return JSON-RPC error code `-32601`.

### Tool List

Fetch the OpenAPI portal aggregate via `openapiportalapi.NewClient(svc).WithOptions(pub.URL(overrideURL)).Document(...)` where `overrideURL` replaces the `:0` port in `openapiportalapi.Document.URL()` with the request's actual port (defaulting to `443`). This forces the portal to filter tools to the caller's port.

Iterate paths and methods in sorted order (for deterministic name disambiguation). Skip operations with an empty `x-name`. Disambiguate name collisions by appending `_2`, `_3`, ... suffixes in order. Build `inputSchema` by inlining `$ref` entries from `#/components/schemas/` recursively (cycle-protected via a `visiting` set), falling back to a synthesized object schema from operation parameters when no request body is present.

### Tool Call

Re-fetch the aggregate on every `tools/call` (the portal's distributed cache absorbs the cost). Match the named tool by iterating in the same sorted order as `tools/list`. Return errors as content blocks with `isError: true` rather than JSON-RPC errors, so the LLM can continue the conversation. Prefix the error message with the HTTP status code when present (e.g. `[404] tool not found`). Set `isError: true` only when `svc.Request` returns an error, not on non-2xx responses with a nil error.

### Design Notes

- The service is stateless per request - no session state, no SSE, no `listChanged` notifications.
- Tool discovery is claim-filtered at the source (the OpenAPI portal); no authorization logic lives here.
- Name disambiguation order is deterministic across calls because both `tools/list` and `tools/call` iterate in the same sorted order.
