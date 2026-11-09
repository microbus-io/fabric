## Design Rationale

### Single wire endpoint, internal dispatch

The service exposes one HTTP endpoint (`POST //mcp:0`) and dispatches on the JSON-RPC `method` field. This matches MCP clients' expectation of one URL per server (Claude Desktop, Claude Code, custom integrations). Splitting into three Microbus endpoints would not be MCP-compliant - real clients send JSON-RPC envelopes to a single URL.

### Capabilities: tools only, no `listChanged`

The server advertises the `tools` capability without `listChanged`. The bus's tool set is effectively static within a session - services come and go between deploys, not mid-session - so sending `notifications/tools/list_changed` would force MCP clients to re-fetch without delivering meaningful new state. If real use cases for in-session tool churn appear (auth-gated tools that surface after login, plugin-style runtime registration), `listChanged` is purely additive and can be added without breaking clients.

### Stateless dispatch, no SSE

Every supported method is request/response. We do not stream: no `notifications/*` are emitted, no `Last-Event-ID` resumption, no SSE bookkeeping. Plain `POST /mcp` with a JSON response works for every method. The "stateful edge / sticky sessions / DistribCache" machinery only matters once we adopt server-push, which we don't.

### `tools/list` derives from the OpenAPI portal

Tool definitions come from the openapi portal's aggregate document, not by walking the bus directly. This piggybacks on the portal's per-claim and per-port filtering - the caller's authorization is enforced once at the source, not duplicated here. The MCP service forwards its request port to the portal via a `pub.URL` override (otherwise the portal's `r.URL.Port()` would be the wildcard `0` from the route, which would zero-out the port filter).

### Functions and web handlers only

`tools/list` exposes only function and web operations. Tasks are not independent call units, and workflows are
asynchronous: a flow can run for longer than any request budget, or park on an interrupt waiting for a human. An MCP
`tools/call` is a synchronous request/response that the agent waits on within its turn, so running a workflow behind
it would silently turn an asynchronous unit into a synchronous one, bounded by the ingress time budget and leaving a
running flow behind when the call times out. Posting the workflow URL directly does not run it either; it returns the
graph. A workflow is therefore treated like any unknown name: absent from `tools/list`, and `[404] tool not found` on
`tools/call`.

The explicit way to reach workflows over MCP would be the MCP Tasks extension, where the server answers a long call
with a task handle that the client polls. It maps onto the foreman directly (the flow key as the task ID, `Create` to
start, `Poll` for `tasks/get`), but it needs both client and server to opt in and the portal does not implement it.

### `tools/call` re-fetches the aggregate

The handler re-fetches the aggregate on every call rather than caching tool→URL mappings. The portal's `DistribCache` already absorbs the cost, and a stateless handler avoids cache-invalidation logic when services come or go.

### Errors as content blocks, not JSON-RPC errors

A tool that doesn't exist or fails to invoke returns a `content` array with `isError: true` rather than a JSON-RPC error object. This surfaces the failure to the model so the conversation can continue, rather than failing the whole turn. JSON-RPC errors are reserved for protocol-level failures (parse error, method not found, malformed params, unsupported jsonrpc version).

The error message is plain text, prefixed with the HTTP status code when present (e.g. `[404] tool not found`). MCP clients pass the content directly to the model as natural language - JSON envelopes are unnecessary structure that the model would just have to parse out. The status code prefix is the one piece of structured information worth surfacing because models reason differently about retryable failures (5XX) versus permanent ones (4XX).

### `isError` only when `Request` returned an error

In Microbus, `svc.Request` returns a non-nil `error` for every real failure - transport problems, panics, or handlers that returned an error (status code is attached to the error, not delivered as a non-2XX response with `err == nil`). A nil error means the round-trip completed normally; the response status code is whatever the handler chose to write. Web handlers in particular can legitimately return 4XX with `err == nil` to express normal negative results (e.g. 404 for a lookup miss, 304 for not-modified). Treating those as `isError: true` would mislabel normal responses as failures and push the model to retry or apologize when it should just read the body and continue. So `isError: true` is set exactly when `Request` returned an error, never based on the response status code.

### Name disambiguation in `tools/list`

When two operations across services share an `x-name`, the second occurrence becomes `<name>_2`, the third `<name>_3`, etc. This lets the MCP client address each tool unambiguously without forcing service authors to negotiate globally unique names.

### Protocol version

The `protocolVersion` returned by `initialize` is bumped only when MCP spec revisions land *and* handlers are updated to match. The constant lives in `service.go` as `mcpProtocolVersion`.

