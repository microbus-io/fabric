# Developing With the Microbus Framework

## Instructions for Agents

**CRITICAL**: This project follows a microservices architecture based on the Microbus framework. Always follow the patterns and conventions in this file.

**CRITICAL**: Before performing any task, check for pertinent skills in `.claude/skills/` and its subdirectories. Follow the workflow of the most relevant skill.

**CRITICAL**: After completing any change to a microservice, always follow the `housekeeping` skill.

**CRITICAL**: After compaction of the context window, re-read this file.

**CRITICAL**: Comments in the code that include the word `HINT` or `MARKER` in all-caps are there to guide you. Do not remove them.

**CRITICAL**: Never edit a file with a comment that includes `Code generated` and `DO NOT EDIT`.

### Reading Framework Design Notes

When you need to understand *why* a Microbus framework package behaves a certain way (not just *how* to use it), read the `CLAUDE.md` file inside that package in the local Go module cache. These files capture design rationale that godoc does not, and they are not bundled into this project. Substitute the package (e.g. `connector`) for `<package>`:

```bash
ver=$(go list -m -f '{{.Version}}' github.com/microbus-io/fabric)
cat "$(go env GOMODCACHE)/github.com/microbus-io/fabric@$ver/<package>/CLAUDE.md"
```

The module cache is read-only; do not edit those files. If `go list` reports the framework is not in `go.mod`, the project does not depend on Microbus and these notes do not apply.

## Key Concepts

### Microservices and Features

A Microbus application comprises a multitude of microservices that communicate over a messaging bus using the HTTP protocol.

A microservice consists of the following features:
- **Web handler endpoints** - raw `http.ResponseWriter`/`http.Request` handlers for serving HTML, files, or custom HTTP responses
- **Functional endpoints (RPCs)** - typed request/response functions with input/output structs, marshaling, and client stubs
- **Outbound events** - messages this microservice fires for others to consume
- **Inbound event sinks** - handlers that react to events emitted by other microservices
- **Configuration properties** - runtime settings (strings, durations, booleans, etc.)
- **Task endpoints** - typed handlers for agentic workflow steps, receiving and returning state via a `workflow.Flow` carrier
- **Workflow graphs** - definitions of agentic workflow structure, describing task transitions and conditions
- **Tickers** - recurring operations on a schedule
- **Metrics** - counters, gauges, and histograms for observability

### Subscription Route

The subscription route of an endpoint is resolved relative to the hostname of the microservice to form an internal Microbus URL. External clients reach the endpoint through the HTTP ingress proxy, which prepends the microservice hostname to the path. For example, route `/my/path` on `myservice.hostname` is internally `https://myservice.hostname/my/path` and externally `https://webserver.hostname/myservice.hostname/my/path`.

It's recommended to set the route of a subscription to match the name of its handler, in kebab-case, e.g. `/my-handler`.

An empty route `""` or `/` maps to the root path of the microservice.

Set a port by prepending it to the route, e.g. `:123/my-path`. A port alone with no path is also valid, e.g. `:123`. The default port is `:443`.

Encase path arguments in curly braces, e.g. `/foo/{foo}/bar/{bar}`. Append a `...` to the name of the last path argument to capture the remainder of the path, e.g. `/my-path/{greedy...}`. Path arguments must span an entire segment of the route. A route such as `/x{bad}x` is not allowed.

Start a route with `//` or `https://` to set an absolute path mapped to the root of the ingress proxy (e.g. `//my/path` becomes `https://webserver.hostname/my/path`). The special route `//root` maps to `/`.

### Ports

Ports provide access isolation. Use port `:0` to accept requests on any port. Conventional port assignments:

- `:443` - default port for standard endpoints (functions and web handlers)
- `:888` - internal management and control endpoints
- `:666` - trust-root endpoints (token mint, shell exec, privileged writes); ACL-restricted to a tiny named set of callers
- `:444` - internal-only endpoints
- `:417` - dedicated port for outbound events
- `:428` - dedicated port for task endpoints
- `:0` - wildcard, accepts requests on any port

The HTTP ingress proxy always blocks inbound requests on internal ports `:666` and `:888`, regardless of deployment mode. Internal `:443` is implicitly reachable; any other internal port must be listed in the ingress's `AllowedInternalPorts` config (operator-tunable, `1024-65535`) to be reachable from outside the mesh. In `LOCAL` deployment that allowlist is ignored and every port except the hard floor is reachable for dev ergonomics.

Endpoints on `:666` are designated trust roots - their compromise undermines the framework's security guarantees (e.g. minting tokens with arbitrary claims, executing shell commands on the host). Operators grant `:666` publish rights only to explicitly trusted caller bundles via NATS ACLs. Do not place an endpoint on `:666` without confirming it meets the trust-root threshold. A trust-root endpoint cannot meaningfully gate itself with `requiredClaims`: it is typically a credential issuer, and requiring a verified claim to obtain one is circular. Its authorization layer is the NATS `:666` `PUB` ACL alone, which is why that allowlist must stay small - so never try to "secure" a `:666` endpoint by adding `requiredClaims` to it.

**Port choice is a security decision.** In a standard ingress configuration port `:443` (plus any port explicitly added to the ingress's `AllowedInternalPorts`) is reachable from outside the mesh through the HTTP ingress proxy, so any endpoint there is world-reachable unless its `requiredClaims` gates it. An operator can lock the proxy down further, but never rely on that: treat an open `:443` endpoint with no `requiredClaims` as public to the internet, and choose it only when the endpoint is genuinely public. Otherwise gate it with `requiredClaims`, and/or place it on an internal-only port (`:444`) so the ingress proxy can never route to it. This matters most for a microservice that holds a stored secret (an API key, a downstream credential) and uses it on the caller's behalf: an ungated `:443` endpoint there is a confused deputy, letting any external caller spend the operator's credential. Decide port and `requiredClaims` together, per endpoint, defaulting to closed.

### Magic HTTP Arguments

Functional endpoints use typed Go function signatures for their inputs and outputs. Three special argument names - `httpRequestBody`, `httpResponseBody` and `httpStatusCode` - provide direct control over HTTP request/response semantics from within a functional endpoint.

- **`httpRequestBody`** - Input argument deserialized directly from the HTTP request body. Other inputs come from query/path arguments.
- **`httpResponseBody`** - Output argument serialized as the sole HTTP response body. Other outputs are excluded.
- **`httpStatusCode`** - `int` output argument that sets the HTTP status code (defaults to `200`).

Used together they expose a functional endpoint as a REST API - e.g. a `POST` create taking
`httpRequestBody *Object` and returning `(key ObjectKey, httpStatusCode int, err error)` with
`http.StatusCreated`, or a `GET` load returning `(httpResponseBody *Object, httpStatusCode int, err error)`
with `http.StatusNotFound` when absent.

### Authentication and Authorization

Microbus uses JWT-based authentication. Endpoints can require specific claims using `requiredClaims` boolean expressions (e.g. `roles.admin`); the expression syntax is in Required JWT Claims below.

Default to closed. An empty `requiredClaims` means anyone who can reach the endpoint may invoke it, which on `:443` (or any port the operator added to `AllowedInternalPorts`) means the entire internet under a standard ingress configuration (see Ports). Leave `requiredClaims` empty only for an endpoint that is intentionally public; otherwise express the actors permitted to call it. An endpoint that wields a stored secret or performs a privileged side effect must never be both ungated and on an externally reachable port.

**IMPORTANT**: If the microservice reads actor claims (`frame.Of(ctx).IfActor` or `ParseActor`), imports auth-related packages (`bearertokenapi`, `accesstokenapi`), or the task involves setting up authentication infrastructure, read `.claude/rules/auth.txt` before proceeding.

### Required JWT Claims

`requiredClaims` is a boolean expression over access token claims. Supported syntax:

- Boolean operators `&&`, `||` and `!`
- Comparison operators `==` and `!=`
- Order operators `>`, `>=`, `<` and `<=`
- Regexp operators `=~` and `!~`. The regular expression must be quoted and on the right, e.g. `prop=~"regexp"`
- Grouping operators `(` and `)`
- String quotation marks `"` or `'`
- Dot notation `.` for traversing nested objects. For this purpose, arrays are treated as sets, enabling expressions such as `roles.admin` to check whether the value `"admin"` is present in the claim `"roles": ["admin", "elevated"]`

Issuer verification is enforced at the framework layer: the connector pins JWKS lookup to the framework's token services, and the `iss` claim of every verified token is guaranteed to match a pinned hostname. Adding an `iss=~"access.token.core"` predicate to `requiredClaims` is therefore redundant and should be omitted. The original identity provider is available in the `idp` claim.

### SQL CRUD Microservices

SQL CRUD microservices are Microbus microservices that expose a CRUD API to persist and retrieve objects in and out of a SQL database. They follow a standardized pattern for multi-tenant isolation, schema migration, column mapping, query filtering, and optimistic concurrency control via revisions.

**IMPORTANT**: If the microservice uses SQL (imports `database/sql` or `github.com/microbus-io/sequel`), read `.claude/rules/sequel.txt` before proceeding.

### Agentic Workflows

Agentic workflows allow microservices to collaborate on multi-step processes. A workflow is a directed graph of tasks orchestrated by the Foreman core service. Each task is an endpoint that reads from and writes to a shared state carried by a `*workflow.Flow`. Tasks are registered on port `:428` by default. To invoke another microservice's workflow as an isolated child flow from inside a task body, use its generated `xapi.NewSubgraph(flow)` client (the `Executor` is test-only); see `.claude/rules/workflows.txt`.

**A task is not an independent call unit.** A task is only ever a node in a graph; the independently-invocable, engine-backed unit is the workflow. There is no way to call a task as a synchronous function, expose it as an LLM tool, or run it as a standalone child flow - to make one unit of work invocable, author a (one-node) workflow and run or subgraph it. Composition from inside a task body is therefore either graph composition (the callee is a node in the same graph, sharing state) or a subgraph of a declared workflow (state-isolated). If the same logic must exist as both a function and a task, extract a plain Go helper that both handlers call, rather than making the task callable as a function.

In this codebase, **"workflow", "agent", and "agentic workflow" are interchangeable terms** for the same `workflow.Graph` construct. An LLM call is not a prerequisite - rule-based, deterministic, small-model, and LLM-driven workflows are all agents. When a user says "create an agent" or "create an agentic workflow", they mean "create a workflow"; use the `add-workflow` skill. When they say "agent step" or "agentic task", they mean a task endpoint; use the `add-task` skill. A microservice whose sole purpose is to host one workflow is scaffolded with `add-microservice` like any other, and its hostname follows the same domain-oriented naming as every other host - there is no special suffix that marks a host as an agent.

**IMPORTANT**: If the microservice defines tasks or workflows (has `tasks` or `workflows` in its `manifest.yaml`), read `.claude/rules/workflows.txt` before proceeding.

### Calling Python

Microservices that need Python libraries (PyTorch, pandas, sentence-transformers, etc.) for their core compute own their own in-process Python virtual environment via the [`github.com/microbus-io/pyvenv`](https://github.com/microbus-io/pyvenv) module. The Go side handles all framework concerns (logging, tracing, config, downstream calls); Python is a pure compute kernel reached via local pipes to a long-lived worker subprocess.

**IMPORTANT**: If the microservice's directory contains a `python.go` and/or a `service.py` at its root (i.e. it was scaffolded with `add-python-microservice`), read `.claude/rules/python.txt` before proceeding.

### Manual subscriptions and per-name activation

A subscription can be registered with the `sub.Manual()` option, in which case the connector skips it during the user-business activation wave of `Startup` and during the deactivation wave of `Shutdown`. The subscription stays off the bus and unicast callers see a clean 404 ack-timeout until user code (typically inside `OnStartup` or a resource-ready callback) brings it online via `svc.ActivateSubscription(name)`. The symmetric `svc.DeactivateSubscription(name)` takes it back off the bus while leaving it registered.

Useful when a handler depends on a heavy resource (Python venv, database connection, ML model load) that isn't ready by the end of `OnStartup`, or that may need to be rebuilt mid-lifecycle (e.g. on a `410 Gone` from an upstream allocator). The microservice goes live on the bus immediately for control-plane reachability while the manual handler stays off-bus until the resource is provisioned. Unicast callers' load balancing routes around the cold replica until the sub activates.

Activation and deactivation are valid while the connector is `startingUp`, `startedUp`, or `shuttingDown`, so user code can drive the lifecycle of a manual sub from inside both `OnStartup` and `OnShutdown`.

To act on a group of related subscriptions, tag them at registration with `sub.Tag("name")` (multiple tags allowed) and iterate `svc.Subscriptions()` to filter by tag, type, or any other field on `connector.SubscriptionInfo`. Each call to `svc.ActivateSubscription(name)` is idempotent.

```go
for _, s := range svc.Subscriptions() {
    if slices.Contains(s.Tags, "python") {
        svc.ActivateSubscription(s.Name)
    }
}
```

The pattern lets a microservice with multiple manual groups (e.g. a Python venv and a database pool) recover one group without disturbing the other.

### Manifest

Each microservice has a `manifest.yaml` documenting the features it *exposes* - its interface contract. Read all
manifests when starting work to build a map of the system. Outbound dependencies (which microservices it calls),
database usage, and HTTP egress targets are *not* here - those are derived from source by `cmd/gencreds` (ACL signing)
and `cmd/gentopology` (topology diagram).

**IMPORTANT**: `myserviceapi/definition.go` is the single source of truth. `manifest.yaml`, `myserviceapi/client.go`,
`intermediate.go`, `mock.go`, and `mock_test.go` are generated from it by `cmd/genservice` (the `housekeeping`
skill runs it). Hand edits to those files are overwritten on the next regeneration. Change `definition.go` and
regenerate, never the reverse - so you never author manifest YAML, only read it.

Top-level sections: `general`, `webs`, `functions`, `outboundEvents`, `inboundEvents`, `configs`, `tickers`,
`metrics`, `tasks`, `workflows`. Each feature is keyed by its PascalCase name with a `description` and, where
applicable, a `signature`. The YAML is self-describing; the non-obvious decode rules are:

- **`general`**: holds `name`, `hostname`, `description`, `package`, and `modifiedAt`. `hostname` is unique
  across the app; `modifiedAt` is RFC 3339 UTC at the actual edit time (not midnight).
- **`method`** (webs/functions/events): one of `GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH`, or `ANY` for
  all methods. Case-insensitive; unknown tokens are rejected at registration with `405`.
- **`loadBalancing`**: `default` (load-balanced on the hostname queue), `none` (multicast to all peers), or a custom
  queue name matching `^[a-zA-Z0-9\.]+$` (load-balanced within the queue, multicast across queues).
- **`requiredClaims`**: a boolean expression over JWT claims; the request is denied if unmet (syntax in Required JWT
  Claims above).
- **`timeBudget`**: per-endpoint max duration declared via the `sub.TimeBudget` option. The framework shortens the
  inbound deadline to `min(caller budget, declared)` and cancels an over-running handler via its context. Recorded
  only when declared, and only the declared value (never the deployment-effective one); otherwise the connector
  default and the foreman `TimeBudget` ceiling apply. Same meaning for tasks.
- **`signature`**: a valid Go signature. Configs - the getter, return type `string`/`int`/`float64`/
  `time.Duration`/`bool`. Tasks - excludes `ctx`, `flow *workflow.Flow`, `err`; an `Out`-suffixed output is the
  read-modify-write pair of its same-named input (i.e. `foo`/`fooOut` map to the same state field). Workflows - declared input
  fields before the parens, output fields inside, an `Out` suffix marking a read-modify-write field; types are
  informational since state is untyped JSON.
- **`configs`**: `secret` (never logged), `callback` (`OnChanged` fires on change), `default`, and `validation`:
  `str <regexp>`, `bool`, `int [0,60]`, `float [0.0,1.0)`, `dur (0s,24h]`, `set Red|Green|Blue`, `url`, `email`,
  `json`. Range brackets: `[` inclusive, `(` exclusive; omit a bound for open-ended (`int [1,]`, `dur [,24h]`).
- **`metrics`**: `kind` is `counter`/`gauge`/`histogram`; `buckets` are histogram boundaries; `otelName` is the
  OpenTelemetry name; `observable` metrics are measured just-in-time. The `otelName` of a counter carries **no**
  `_total` suffix - that suffix is a Prometheus convention the scrape exporter appends at the `/metrics` boundary
  (de-duplicating if already present), whereas the OTLP push path uses the name verbatim; so a counter named
  `app_requests` is queried in Prometheus as `app_requests_total`.
- **`tickers`**: `interval` is the duration between iterations.
- **`inboundEvents`**: `package` is the import path of the event-source microservice.

### Markers

Each feature of a microservice has corresponding code across several files: the declaration in `*api/definition.go`, the handler in `service.go`, the subscription in `intermediate.go`, the client stub in `*api/client.go`, the mock in `mock.go`, and the test in `service_test.go`. To locate all code related to a specific feature, search for `// MARKER: FeatureName` comments within the microservice's directory. For example, to find all code related to the `Hello` feature of the `hello.example` microservice, search for `MARKER: Hello` under its directory.

Markers are scoped to a single microservice - different microservices can define features with the same name, so always search within the specific microservice's directory.

## Project Structure

- Each microservice lives in its own directory with a `*api/` subdirectory for its public interface (client stubs, types, and endpoint declarations)
- The main app is at `main/main.go`

### API Package Layout

Each `*api/` directory contains:

- **`definition.go`** (hand-written, the single source of truth) - the public contract: the `Hostname`, `Name`, `Version`, and `Description` consts, and one `define.*` var per feature together with its In/Out struct types, e.g. `var MyEndpoint = define.Function{Host: Hostname, Method: "POST", Route: "/my-endpoint", In: MyEndpointIn{}, Out: MyEndpointOut{}}`. The var's godoc is the feature's description; write it as a `/* */` block comment for better readability when it span multiple lines. Use backticks to quote strings, avoid backticks inside strings, and always use literal strings that are AST-readable.
- **sibling type files** (hand-written) - one per complex domain type (e.g. `point.go` -> `type Point struct{...}`), plus type aliases to other packages' types (`type Point = anotherapi.Point`).
- **`client.go`** (generated by `cmd/genservice`) - the proxy stubs: `Client`, `MulticastClient`, `MulticastTrigger`, `Hook`, `Executor`, `Subgraph`, and per-feature methods with `Response` wrappers for multicast.

Feature declarations and In/Out structs belong in `definition.go`. The OpenAPI document is *not* maintained here - it's built at runtime by the connector (`connector/control.go`) from the live subscription map, scoped to `Function`/`Web`/`Workflow` features and filtered by the caller's actor claims.
- `config.yaml` sets configuration properties, scoped by microservice hostname or `all:`. Secrets go in git-ignored `config.local.yaml`
  ```yaml
  my.service:
    MyConfig: value
  all:
    SharedConfig: true
  ```
- `env.yaml` overrides OS environment variables. Secrets go in git-ignored `env.local.yaml`

## Common Patterns

### Working With Static Resource Files

Place static resource files in the `resources` directory of the microservice, or a subdirectory thereof. These files are embedded inside the binary using `go:embed`.

Resource files are accessed in the code using `svc.ReadResFile` or `svc.ResFS`.

If the file is a text or HTML template, use `svc.WriteResTemplate(w, "template.html", data)` to read and execute it in a single operation. HTML templates must be named with a `.html` extension for proper escaping. `svc.WriteResTemplate` does not support func maps or custom delimiters - use the standard library pattern if either is required.

### Calling Downstream Microservices

Use the generated client stub for type-safe communication with downstream microservices. Four client types are available in each `*api` package:

- **`NewClient`** - unicast (one-to-one) request/response. Returns a single result, load-balanced among peers.
- **`NewMulticastClient`** - multicast (one-to-many) request/response. Returns an iterator of zero or more responses from all peers.
- **`NewMulticastTrigger`** - fires outbound events defined in the microservice's API. Used by the event source to notify subscribers.
- **`NewHook`** - subscribes to inbound events from another microservice. Used by event sinks to react to events.

Unicast returns a single result; trace and propagate its error:

```go
validated, err := downstreamapi.NewClient(svc).ValidateOrder(ctx, orderID)
if err != nil {
	return errors.Trace(err)
}
```

Multicast (a one-to-many call, or firing an outbound event and awaiting responses) returns an iterator of zero or
more responses; call `.Get()` on each element for its result and error:

```go
for e := range downstreamapi.NewMulticastClient(svc).ValidateOrder(ctx, orderID) {
	validated, err := e.Get()
	if err != nil {
		return errors.Trace(err)
	}
	// ...
}
```

Fire-and-forget an outbound event by calling the trigger without ranging over it:
`myserviceapi.NewMulticastTrigger(svc).OnOrderCreated(ctx, order)`. Subscribe to an inbound event - typically in
`OnStartup` - with `eventsourceapi.NewHook(svc).OnOrderCreated(svc.onOrderCreated)`, where the handler has the
event's signature and returns an `error`.

### Logging

Use `svc.LogDebug`, `svc.LogInfo`, `svc.LogWarn` and `svc.LogError` to print to the log. Logs include attributes in the `slog` name=value pair pattern, e.g. `svc.LogError(ctx, "Job failed", "job", jobID, "error", err)`. Use the label `"error"` when logging an `error`. In most cases there is no need to log errors - any error returned from an endpoint is automatically logged by Microbus.

`svc.LogDebug` calls are filtered out by default. Set the env var `MICROBUS_LOG_DEBUG=1` (e.g. when running `go test` or starting a binary) to enable verbose debug output when diagnosing an issue.

### Distributed Tracing

Every call to an endpoint is automatically wrapped with a trace span. The span can be accessed via `svc.Span(ctx)` and extended with `SetAttributes` or `LogInfo` using the slog name=value pair pattern. All downstream service calls automatically participate in the trace.

### Goroutines

Use `svc.Go(ctx, func)` to launch a goroutine in the context of a microservice. Use `svc.Parallel(ctx, func1, func2, ...)` to launch multiple goroutines and wait for all to complete. Each job receives a cancelable subcontext of `ctx` that is canceled when any job errors, so a job that observes its context can abandon its work early; a job that should run to completion regardless can ignore its context argument and capture the outer `ctx` instead.

For long-lived background work that should run for the whole microservice lifetime (worker pools,
refillers, model warmers, periodic reconcilers), launch raw goroutines from `OnStartup` passing
`svc.Lifetime()` as the root context. `svc.Lifetime()` is valid by the time `OnStartup` runs and
stays valid through `OnShutdown`; the framework cancels it only after `OnShutdown` returns. Track
your goroutines with a `sync.WaitGroup` and drain them inside `OnShutdown` before returning, so
in-flight work finishes cleanly before the lifetime ctx is cancelled. Don't capture `OnStartup`'s
`ctx` argument for this — it's bounded by the startup time budget and will be cancelled out from
under long-running goroutines.

### Making HTTP Web Requests

HTTP requests to the web should use the HTTP egress proxy rather than the standard Go `http.Client`.

```go
func (svc *Service) FetchListFromWeb(ctx context.Context) (result []string, err error) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := httpegressapi.NewClient(svc).Do(ctx, req)
	// Process response here...
	return result, nil
}
```

Add the HTTP egress proxy to the main app in `main/main.go`, if not already added.

### Miscellaneous

- **Sleeping**: `svc.Sleep(ctx, dur)` returns early if `ctx` or the microservice's lifetime is canceled,
  returning the canceling context's error (`context.Canceled` or `context.DeadlineExceeded`), traced, or `nil` if
  the full duration elapsed. Propagate a non-nil return so a canceled or shutting-down operation stops promptly
  instead of blocking and continuing as if it had slept:

  ```go
  err := svc.Sleep(ctx, dur)
  if err != nil {
      return errors.Trace(err)
  }
  ```

### Recording Metrics

Use `svc.IncrementMyMetric(ctx, delta)` for counters and `svc.RecordMyMetric(ctx, value)` for gauges and histograms. The method names are generated from the metric name defined in the manifest.

### Mocking

#### Mocking Microservices

A microservice's `Mock` provides type-safe methods for mocking all its endpoints. Build a test app from the
microservice under test, a `connector.New("tester.client")` (wrapped in the relevant `*api.NewClient`), and mocks
for its downstream microservices, then `app.RunInTest(t)`:

```go
webpayMock := webpay.NewMock()
webpayMock.MockCharge(func(ctx context.Context, userID string, amount int) (success bool, balance int, err error) {
	return true, 100, nil
})
app := application.New()
app.Add(
	// HINT: Add microservices or mocks required for this test
	svc, tester, webpayMock,
)
app.RunInTest(t)
```

#### Mocking the HTTP Egress Proxy

Mock the HTTP egress proxy with `httpegress.NewMock()` (added to the test app like any mock) to avoid real network
requests. In `MockMakeRequest`, extract the proxied request with `http.ReadRequest(bufio.NewReader(r.Body))`, branch
on its method/URL, and write the response; `defer httpEgressMock.MockMakeRequest(nil)` to reset:

```go
httpEgressMock.MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
	req, _ := http.ReadRequest(bufio.NewReader(r.Body))
	if req.Method == "GET" && req.URL.String() == "https://api.example.com/data" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return nil
})
defer httpEgressMock.MockMakeRequest(nil)
```

Only the wrapper's own tests should mock the egress proxy directly. Upstream microservices should mock the wrapper itself using its `NewMock()`.

### Error Handling

#### Tracing Errors

Use `errors.Trace(err)` to associate the current stack location with the error. Stack traces are preserved across microservice boundaries. Attach properties using the `slog` name=value pair pattern, and HTTP status codes as unpaired arguments: `errors.Trace(err, "userID", userID, http.StatusBadRequest)`.

#### New Errors

Use `errors.New` to create new errors. Do not use the deprecated constructors `errors.Newc`, `errors.Newf` or `errors.Newcf`.
Error strings should not be capitalized (unless beginning with proper nouns or acronyms), nor end with punctuation.

```go
if count == 0 {
	return errors.New("no objects")
}
```

`errors.New` supports `fmt`-style formatting, `slog` name=value properties, HTTP status codes as unpaired arguments, and error wrapping (unpaired error argument, equivalent to `%w`). All can be combined:

```go
file, err := os.Open(fileName)
if err != nil {
	return errors.New("failed to open file '%s'", fileName,
		"fileName", fileName,
		http.StatusBadRequest,
		err,
	)
}
```

### Caching

The distributed cache allows multiple replicas of the same microservice to share a single cache.
Access the distributed cache at `svc.DistribCache`.
Use `Get`, `Set`, `Delete`, `Peek`, `Has` or `Clear` to interact with the distributed cache.
Initialize the cache in the microservice's `OnStartup` callback, with the distributed cache's `SetMaxAge` and `SetMaxMemory`.

### Reading Path Argument Values

To read the value of a path argument in a web handler, use `r.PathValue("argumentName")`. Do not use `httpx.PathValues`.

```go
// RenderObjectProperty's route is /object/{id}/property/{prop}

func (svc *Service) RenderObjectProperty(w http.ResponseWriter, r *http.Request) (err error) {
	id := r.PathValue("id")
	prop := r.PathValue("prop")
	// ...
}
```

To read the value of a path argument in a functional endpoint, name an argument the same as the path argument. If the function argument type is not a string, an attempt will be made to convert it to the right type.

```go
// LoadObject's route is /load/{category}/{name...}

func (svc *Service) LoadObject(ctx context.Context, category int, name string) (object SomeObject, err error) {
	// ...
}
```

### Exposing Endpoints as LLM Tools

To let an LLM invoke endpoints from other microservices, pass a `[]string` of canonical endpoint URLs (one per `Def`, e.g. `calculatorapi.Arithmetic.URL()`) as the third argument of `llmapi.NewClient(svc).Chat(ctx, messages, tools)`. Endpoints from multiple services can be combined. The LLM service fetches each host's OpenAPI document, finds the matching operation, and converts it into a callable tool. See the godoc on `llmapi.Client.Chat` for a full example.

Only `FeatureFunction`, `FeatureWeb`, and `FeatureWorkflow` endpoints are exposed; tasks and outbound events are filtered out at the OpenAPI document level (in `connector/control.go`) so they never reach the tool builder. When two endpoints share the same name, the first keeps the bare name and subsequent ones get `_2`, `_3`, ... suffixes.

The `llm.core` service orchestrates the tool-calling loop and dispatches workflow tools as dynamic subgraphs. Do not `ForHost` the client to a provider (`claudellm`, `chatgptllm`, `geminillm`) directly - that bypasses the loop.

### Connecting to Remote APIs

Wrap each remote API (e.g. a third-party web service) in its own microservice. This:
- Encapsulates connection logic, URLs, and request/response structures in one place
- Stores credentials in a single secret configuration property
- Generates type-safe client stubs for use by upstream microservices
- Enables mocking of the remote system in tests

Create a functional endpoint or web handler for each operation of the remote API. Use the HTTP egress proxy for making the outbound HTTP requests.

## Conventions

### Naming Conventions

- PascalCase for types: `User`, `OrderStatus`
- PascalCase for public functions: `GetUserById`, `ProcessOrder`
- camelCase for private functions: `getUserById`, `processOrder`
- lowercase for hostnames: `userservice.example`
- kebab-case for URL routes and paths: `/annual-report`
- lowercase for file names: `userservice.go, mytype.go`
- camelCase for JSON tags: `json:"myProperty,omitzero"`

The hostname naming convention (segment structure, when to include each segment, folding a role word into the leaf)
is a creation-time concern documented in the `name-microservice` skill, which the scaffolding skills consult.

### Naming-Driven Behavior

Microbus uses a number of naming conventions to drive framework behavior without explicit configuration. These are listed here as a single index; each is documented in detail near the feature it governs.

| Pattern | What it does | Where |
|---|---|---|
| `httpRequestBody`, `httpResponseBody`, `httpStatusCode` | Magic HTTP arguments on functional endpoints — the named arg is bound directly to the HTTP request/response body or status code. | Magic HTTP Arguments |
| Path argument names matching function arg names | A function argument whose name matches a `{argName}` segment in the route is auto-populated from the path. | Reading Path Argument Values |
| `dv8` tags and `Validate(ctx) error` / `Validate() error` methods on api types | Enforced automatically on the In structs of functions, tasks, and events after decoding (an invalid payload is rejected `400` before the handler runs, failing a flow at the offending task) and on structured config values before they are committed. Inert on Out structs. | `add-type` skill |

### OpenAPI Parameter Descriptions

Per-field descriptions enrich the OpenAPI spec (better docs and LLM tool-calling accuracy) via a single
convention: a `jsonschema_description:"..."` tag on the In/Out struct field (read by `invopop/jsonschema`).
Absent, descriptions are simply empty and the endpoint godoc remains the endpoint description. The tag works
uniformly for body fields, scalar query/path arguments, and fields within nested custom struct types.

```go
type ForecastIn struct {
    City string `json:"city,omitzero" jsonschema_description:"The city name, e.g. San Francisco"`
    Days int    `json:"days,omitzero" jsonschema_description:"Number of days to forecast, 1-14"`
}

type DayForecast struct {
    Date string  `json:"date,omitzero" jsonschema_description:"Date is the forecast date in ISO 8601 format"`
    High float64 `json:"high,omitzero" jsonschema_description:"High is the high temperature in Fahrenheit"`
}
```

Use the dedicated `jsonschema_description` tag rather than the `jsonschema:"description=..."` subtag form: it is
read whole, so the description may contain commas (the `jsonschema` tag is comma-split into directives, which
truncates a description at its first comma). Reserve the `jsonschema:"..."` tag for other directives such as
`jsonschema:"example=6"`. The legacy `Input:`/`Output:` godoc-section convention has been retired; descriptions
live only on the fields.

A `jsonschema_description` tag on a magic HTTP body field (`HTTPRequestBody`/`HTTPResponseBody`, whose `json` tag is
`-`) describes the body payload as a whole; it surfaces on the OpenAPI request-body/response node. This is the only
way to describe a magic body over a non-struct type (e.g. `[]string`), and it complements the endpoint's own godoc.

```go
type CreateIn struct {
    HTTPRequestBody *Object `json:"-" jsonschema_description:"The object to create"`
}
type LoadOut struct {
    HTTPResponseBody *Object `json:"-" jsonschema_description:"The loaded object"`
    HTTPStatusCode   int     `json:"-"`
}
```

## Development Workflow

### Building a Microservice With Skills

Always use skills in `.claude/skills/` to build microservices. Scaffold with `add-microservice`, then use `add-feature` skills for each feature.

The available feature skills are:

| Skill | Feature |
|---|---|
| `add-config` | Configuration property |
| `add-metric` | Metric |
| `add-outbound-event` | Outbound event |
| `add-function` | Functional endpoint (RPC) |
| `add-web` | Web handler endpoint |
| `add-inbound-event` | Inbound event sink |
| `add-task` | Task endpoint (workflow / agent step) |
| `add-workflow` | Workflow graph (agent / agentic workflow definition) |
| `add-ticker` | Ticker |

The recommended order is configs, metrics, outbound events, functions, webs, inbound events, tasks, workflows, then tickers. This order is not mandatory but it follows the natural dependency chain - for example, a function may reference a configuration property or record a metric, so those should exist first. Workflows reference task endpoints, so tasks should be defined first.

To modify or remove an existing feature, use `modify-feature` or `remove-feature`. Both invoke `housekeeping` at the end. Wire-format-affecting changes (renamed routes, renamed hostnames) don't require regenerating other services' manifests - NATS ACLs are derived from source at deploy time by `cmd/gencreds`, so per-service manifests stay focused on what each service exposes.

### Building the Project

Use `go vet ./main/...` to verify compilation (not `go build`, which conflicts with the `main/` directory name). For a binary: `go build -o app ./main/...`.
