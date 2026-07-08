## Design Rationale

### Lifecycle phases are an atomic state machine

The connector lives in one of four phases - `shutDown`, `startingUp`, `startedUp`, `shuttingDown` - held in an `atomic.Int32`. Almost every setter (`SetHostname`, `SetPlane`, `SetDeployment`, `SetOnStartup`, `DefineConfig`, ...) refuses to run unless the phase is `shutDown`. This is what makes the connector effectively immutable once started - there is no locking around the configuration fields because the phase guard makes mutation impossible after `Startup`.

`captureInitErr` exists to defer pre-start errors. Callers can chain `Init(...)` and similar setup without checking each error; the *first* init error is stashed and re-raised by `Startup`. After `Startup` runs, `initErr` is cleared so a subsequent restart isn't poisoned.

`Startup` uses `defer phase.CompareAndSwap(startingUp, shutDown)` as a fallback so a startup that errors out before reaching `startedUp` returns the connector to a startable state. The error path also calls `Shutdown` to clean up partially-initialized state (transport, dlru, OTel providers) - the deferred CAS only fires if `Shutdown` didn't already advance the phase.

### Subscription activation order during Startup

Subscriptions activate in deliberate waves, not all at once:

0. **Response sub.** Immediately after the transport connects, the connector subscribes to its own response subject (`<plane>.r.<reverseHost>.<id>`). This must come before everything else because the connector itself makes outbound requests during the rest of Startup - most obviously to `configurator.core` to fetch config values - and those replies have nowhere to land without the response sub already in place.
1. **dlru subs** (registered by `dlru.Cache.start` with `sub.Manual()`) activate *before* the user's `OnStartup` callback runs, so the distributed cache is reachable from inside `OnStartup`. The connector finds them by route - the cache lives at `:888/dcache/...`, so `lifecycle.go` scans for `s.Port == "888"` and a `/dcache/` route prefix. The connector chose the path when it called `dlru.NewCache`, so route-based identification keeps the contract one-sided rather than requiring a tagging convention.
2. **Control subs** (`:888/ping`, `/config-refresh`, `/metrics`, `/trace`, `/on-new-subs`, `/openapi.json`) activate *after* `OnStartup` returns - the control plane is not exposed until the service has finished its own init.
3. **User business subs** activate after the control subs, via `activateSubs` which skips any sub marked `sub.Manual`. User-owned manual subs (Python venv handlers, etc.) come on-bus when user code calls `Connector.ActivateSubscription` - typically from inside `OnStartup` once a backing resource is ready, or from a recovery path on `410 Gone`.

Tickers start only after `phase` reaches `startedUp`. The lifetime context (`lifetimeCtx`) is created immediately after the transport connects, *before* `OnStartup` runs, so user code can pass `c.Lifetime()` to long-lived goroutines launched from `OnStartup` (worker pools, refillers, ML workers) without rolling its own root context. The lifetime ctx stays valid through `OnShutdown` and is cancelled only after `OnShutdown` returns and the soft drain has elapsed - see "Shutdown's two-phase drain" below.

### Manual subscriptions and the dlru tag

`sub.Manual()` opts a subscription out of the connector's automatic activate/deactivate passes. The framework uses this for one concrete coordination: the [distributed cache](../dlru) must be reachable from `OnStartup` *and* from `OnShutdown` but must be off-bus during the offload-to-peers pass inside `dlru.Close`. The `dlru` package registers its two subs with `sub.Manual()`; the connector's lifecycle code identifies them by route (`:888/dcache/all`, `:888/dcache/rescue`) and brings them on-bus before `OnStartup` and off-bus after `OnShutdown`. The path is set by the connector itself when it calls `dlru.NewCache`, so route-based detection avoids a separate tagging contract.

User code uses the same `sub.Manual()` mechanism for any subscription whose backing resource is not ready by the end of `OnStartup` (Python venvs, ML model loads, database pools). Those subscriptions stay off-bus through Startup and are activated by user code once the resource is ready - typically by iterating `Connector.Subscriptions()` filtered by a user-chosen tag (e.g. `"python"`).

In `TESTING` deployment the `sub.Manual` flag is ignored: `activateSubs` brings every registered subscription on-bus, and `deactivateAutoSubs` takes them all off-bus on shutdown. This keeps mocks reachable in `application.RunInTest` without per-test setup - tests typically swap the backing resource for a mock, so the "wait for resource ready, then activate" idiom that real `OnStartup` code uses doesn't run. The carve-out is scoped to `TESTING`; `LOCAL`, `LAB`, and `PROD` retain the manual-stay-off-bus behavior.

### Shutdown's ordering, drain budget, and the dlru offload window

`Shutdown` runs in three blocks: prepare, drain, mandatory teardown.

1. **Prepare.** Stop tickers; deactivate auto subscriptions (inbound business requests stop). Manual subscriptions (dlru's `:888/dcache/...` plus anything user-tagged) stay on-bus.
2. **`OnShutdown(ctx)`.** Runs *before* any drain. `c.Lifetime()` is still valid, dlru is still up, the transport is still connected, outbound calls work, `svc.Go` can still launch goroutines. User code that owns long-lived workers drains them here (the foreman, for example, drains its worker pool, timer, and refiller in strict order before returning).
3. **Soft drain.** Cooperatively polls `pendingOps` until it reaches zero, bounded by the partition of the remaining ctx budget. `pendingOps` counts in-flight requests, ticker goroutines and their invocations, and goroutines launched via `Go` / `Parallel`. Anything launched with bare `go` is invisible to this counter. Ticker goroutines are stopped in the prepare phase (each ticker's `Done` channel is closed, and the lifetime context is a backstop), so they exit during the soft drain rather than blocking it.
4. **Cancel lifetime ctx.** The escalation step. Goroutines still running observe `Lifetime().Done()` and should exit promptly.
5. **Hard drain.** A short tail (capped at 2s) for cancellation-aware goroutines to exit after seeing it.
6. **Mandatory teardown.** Closes dlru, unsubscribes the response sub, disconnects the transport, flushes OTel. These steps run on `context.WithoutCancel(ctx)` wrapped in a 2-second `teardownBudget`, so they are never preempted by the caller's deadline expiring upstream - a missed flush or skipped offload would silently lose data.

dlru-specific asymmetries:

- **dlru subs stay active *through* OnShutdown.** User shutdown code can still hit `DistribCache`.
- **dlru subs go down inside `dlru.Close` itself, *before* the offload pass.** `Cache.Close` deactivates its own `/all` and `/rescue` subscriptions first and only then offloads to peers, so rescue PUTs route only to peers - the sender's own queue group is already gone and entries can't loop back into this about-to-be-discarded cache. The connector just calls `Close`; the cache owns the ordering.

Drain budget partitioning (`shutdownDrainBudgets`): if the caller ctx has a deadline, `remaining = time.Until(deadline) - teardownBudget`; hard drain is `min(remaining/4, 2s)` floored at 100ms; soft drain is what's left. Without a deadline the fixed defaults are 8s soft and 2s hard, matching the pre-refactor behavior. `application.Run` reads `MICROBUS_SHUTDOWN_TIME_BUDGET` (default 24s, deliberately below k8s's 30s `terminationGracePeriodSeconds`) and `MICROBUS_STARTUP_TIME_BUDGET` (default 120s, no comparable k8s ceiling so the budget targets headroom for cache warmup / model load / slow dependency wait) and applies them as deadlines on the ctx it passes to `Shutdown` / `Startup`.

### Plane is a NATS-level isolation prefix

The plane prefixes every NATS subject this connector subscribes to or publishes on, allowing multiple Microbus apps to coexist on the same NATS infrastructure without crosstalk. Three uses:

- **Test parallelism.** When no plane is set and the binary is running under `go test`, the plane is derived from `sha256(testFuncName)[:8]`. Concurrent `go test` runs against the same NATS cluster therefore cannot leak messages into each other.
- **Multi-tenant prod.** Multiple production apps can share NATS infrastructure by each setting their own plane.
- **Default development.** Outside of tests the default plane is `microbus`, which is what local dev typically runs on.

`MICROBUS_PLANE` env var or `SetPlane` overrides the default. The plane is otherwise opaque - it has no parsing rules beyond `[0-9a-zA-Z]*`.

### Time budget is a duration, depth is a counter

Time budget propagates as a duration header (decremented per hop by `networkRoundtrip`), not a deadline timestamp - clocks across replicas are not assumed to be synchronized. A request whose remaining budget falls below one network round-trip errors out as `408 Request Timeout` rather than being dispatched.

That 408 is delivered to the caller as an `OpCodeError` response, not signalled by a bare early `return` from `handleRequest`. `onRequest` acks before it spawns the handler goroutine, so once acked the caller is past the ack-timeout fast-fail and is waiting for a real response; a subscriber that rejected the request but sent nothing back would strand the caller until its own `pub.Timeout`. This bites hardest when the budget is shortened below a round-trip by the subscription's own `sub.TimeBudget` rather than by caller drawdown: the caller's `pub.Timeout` is generous, the caller-side `time.Until(deadline) <= networkRoundtrip` check in `Publish` therefore does not fire, and only the subscriber knows the declared budget is too small - so the subscriber must be the one to report it. The budget rejection feeds `handlerErr` and falls through the shared error-response path (skipping the handler) instead of returning early.

Call depth is propagated similarly, incremented by `Publish` on each outbound hop. The default cap of 64 is a cycle detector - at depth 64 `Publish` returns `508 Loop Detected` synchronously without touching the bus.

### Ack-or-fail-fast and the LOCAL escape hatch

`makeRequest` waits up to `ackTimeout` (defaulting to the transport's measured latency) for any responder to ack. If no ack arrives:

- For unicast: synthesize a `404 Not Found` "ack timeout" error.
- For multicast: assume zero responders matched, drop the known-responders cache entry, return cleanly with no responses.

In LOCAL deployment only, if the ack timer fires after `8 * ackTimeout` of wall time the timer is reset once and a debug log is emitted. This is a safety net for long pauses that aren't reflective of a missing responder - most obviously a developer paused in a debugger. The threshold is not exposed; LOCAL is the gating signal.

### Multicast known-responders cache

Multicasts can finish before the full timeout when the connector recognizes "I've seen everyone I expected." `knownResponders` is keyed by NATS subject and stores the set of responder queues seen on the previous successful call. Each subsequent call compares `seenQueues == expectedResponders` and short-circuits when the set matches.

The cache is invalidated when any peer announces new subscriptions on `:888/on-new-subs`. `notifyOnNewSubs` is broadcast on every `Subscribe` call (after Startup), telling other microservices that a new subscriber is in town so they can drop stale cache entries. On a request timeout the local cache entry for that subject is also dropped.

The broadcast is a deliberate O(replicas)-per-activation cost paid for multicast determinism. Without it, a client
holding a stale known-responders set short-circuits before the new responder answers and misses it - while another
client, whose timing differs, sees it. Two clients issuing the same multicast would then get different responder
sets indefinitely, with nothing driving convergence: a successful short-circuit never times out, so the stale cache
never self-corrects. The broadcast also has to land before the new subscription comes on-bus, which is why it cannot
be debounced or coalesced without delaying activation. Post-startup subscription churn is rare and coarse (a manual
sub activating once its backing resource is ready, a `410 Gone` recovery), so the per-activation cost is acceptable;
a service that flips manual subscriptions at high frequency is an application design smell, not a load the framework
optimizes for.

### Locality-aware routing is a NATS subscription trick, not a router

A connector with `locality = "us-west-b-1"` subscribes its handlers to *several* NATS subjects derived from the same route: the bare-slot subject, the per-instance subject (`id-<id>` slot), and one slot per locality prefix (`loc-us`, `loc-us-west`, `loc-us-west-b`, `loc-us-west-b-1`). The locality is stored hyphen-joined and broadest-first (the AWS region/AZ shape), and `escapeLocality` simply prepends `loc-`. `SetLocality` also accepts DNS-style dot notation with the most specific identifier first (`1.b.west.us`); when the input contains a dot, the segments are reversed and joined with hyphens before storage, so legacy dot-form values continue to work without behavior change.

NATS itself does not pick the "most specific" slot - it only queue-group-dispatches within whichever slot the publisher addresses. The narrowing happens publisher-side, driven by a response header. Every responder stamps its own locality on the response (`frame.Of(res).Locality()` carries the hyphen-form). `Publish` reads that header, walks segment-by-segment to find the longest common prefix between the caller's locality and the responder's, and caches that prefix in a `localResponder` LRU keyed by canonical URL (sans query). Subsequent unicasts wrap the cached prefix in slot form and inject it as a single hostname segment before publishing (e.g. `https://example.com/foo` becomes `https://loc-us-west.example.com/foo`). If a locality-prefixed request comes back `404` (the slot has no responders), `Publish` falls back to the original URL once and clears the cache entry. Over a few requests the publisher converges on the most specific slot whose subscribers actually answer.

Hostnames addressed by instance ID (`https://id-<id>.host/...`) skip locality optimization - the caller has already pinned a specific replica.

### Subject encoding

NATS subjects are derived from `(plane, trust, port, src, dest, idOrLocality, method, path)`. The trust segment is `safe` for non-`:666` ports, `danger` for `:666`, or `reply` for response subjects. A dedicated `id_or_locality` slot sits between the dest and the method, carrying an instance prefix (`id-XXXX`), a locality slot (`loc-flat-suffix`), or `_` when neither is present. Subject layout:

```
<plane>.<trust>.<port>.<src_flat>.<dest_flat>.<id_or_locality>.<method>.<path...>
```

A request whose derived subject exceeds 1024 characters (`maxSubjectLength`) is rejected by `Publish` with a
`414 Request URI Too Long` before touching the bus, and `activateSub` refuses to subscribe to a subject over the
same cap - NATS recommends short subjects, and the cap bounds attacker-lengthened URLs. The HTTP ingress enforces
the same bound at the edge, rejecting external request paths longer than 1024 so oversized URLs never enter the
mesh, its logs, or its spans.

The slot keeps per-instance and locality-aware addressing on a separate axis from the dest hostname so publishers can target without ambiguous segment-level reasoning - the publisher inspects the URL hostname's first segment, and a `id-` or `loc-` prefix becomes the slot value while the rest of the hostname becomes the dest. The reservation is enforced centrally in `httpx.ValidateHostname`, which rejects any hostname matching `^id-` or `^loc-`. Both service identities (via `SetHostname` / `SetLocality`) and subscription route hostnames (via `sub.NewSubscription`'s route-validation helper) flow through the same check, so `id-`/`loc-` first segments cannot enter the system at either registration point.

Hostname encoding (`escapeHostname` / `unescapeHostname`):

- `.` becomes `_` (the legacy flattening - keeps typical service identities readable: `payments.core` → `payments_core`).
- URL-special characters that the route validator allows (e.g. `$`, `!`, `~`) are percent-encoded as `%xx` (2-digit lowercase hex per byte). So a route hostname `my$.xml` flattens to `my%24_xml`.
- `[A-Za-z0-9_-]` pass through unchanged.

The asymmetry between `.` (legacy `_`) and other specials (`%xx`) is intentional: the readable flat form is preserved for the common case, and route hostnames with URL specials remain representable. Note that `_` in input is preserved, so a route hostname containing a literal underscore can collide with one whose dot maps to underscore. The framework forbids `_` in service identities to avoid this collision; route hostnames inherit the legacy ambiguity for compatibility.

`unescapeHostname` first reverses `_` → `.` and then percent-decodes `%xx` sequences, recovering the canonical hostname form.

Path encoding rules:

- An empty path becomes `_`.
- A greedy path argument `{name...}` becomes `>`.
- Other path arguments and segments equal to `*` become `*`.
- Each path segment is independently escaped: every byte outside `[A-Za-z0-9-]` becomes `%xx` (2-digit lowercase hex). Includes `.` (`%2e`), `_` (`%5f`), `{` (`%7b`), `}` (`%7d`), and any byte of a multi-byte UTF-8 rune.

Path segments are case-sensitive: uppercase letters pass through unchanged. Hostnames, methods, and the id/locality segment are lowercased before encoding; path segments are not.

Subscription wildcards: port `0` becomes `*` (any port - by convention nothing publishes to port `0`); method `ANY` becomes `*` (any method); the source segment is always `*` (see below).

Reply subjects use `reply._` for the trust+port slots: `<plane>.reply._.<src_flat>.<dest_flat>.<id>`. The trust segment alone identifies the channel; the port slot uses the `_` placeholder for symmetry with request subjects of constant depth. The receiver's response subscription wildcards the source: `<plane>.reply._.*.<own_dest_flat>.<id>`.

The trust segment exists to make "any port except `:666`" expressible as a single ACL pattern (`<plane>.safe.*.<from>.>`) without colliding with trust-root subscriptions on `<plane>.danger.666.*.<dest>.>`. NATS deny-precedence semantics would otherwise make the carve-out impossible.

Subscriptions always wildcard the source segment (`*`). A service does not discriminate at the subscription layer over which peers may call it - that's the publisher's NATS PUB ACL's responsibility. Per-source SUB would also explode the subscription count by allowed-caller and compound with replicas and locality-aware routing.

`splitSubject` parses an inbound subject back into the six slots and unflattens the source and dest hostnames to canonical dot form. Every subject the framework emits has at least six segments by construction (requests have six plus method and path; replies have exactly six), so `splitSubject` does not signal malformed input. A subject with fewer segments produces zero values for the missing fields, which downstream code already treats as unverifiable (an empty source segment fails the `ackRequest` contract).

### Subject encoding is a contract, not an implementation detail

The exact subject layout is **load-bearing for security**, not a private implementation detail of the connector. Three independent things have to agree on it byte-for-byte:

1. **`subjectOf` and helpers in `subjects.go`** - what publishers emit and subscribers match against at runtime.
2. **This document** - the operator-facing description of the wire format. Operators reading or writing NATS ACLs depend on it being accurate.
3. **A future ACL generator skill** - Markdown + prompt instructions, hardcodes the layout from this doc since it can't call Go at run time. Drift here means generated ACLs don't match runtime traffic, and the failure looks like "ACLs are broken" rather than "the encoding changed."

The pinning mechanism is a regression-test golden table in `subjects_test.go` - `TestConnector_SubjectOfSubscription`, `TestConnector_SubjectOfRequest`, `TestConnector_subjectOfResponseSub`, `TestConnector_subjectOfResponse` cover representative cases (multi-segment hostnames, hyphenated hostnames, root paths, path arguments, greedy paths, wildcard ports, lowercasing). Any change to the layout has to update those tests, and the diff in the test file is what code review catches.

If you change anything in `subjectOf`, `splitSubject`, `flattenHostname`, `extractPosition`, `localitySlot`, or `escapePathPart`:

- Update the regression-test goldens in `subjects_test.go` (the test failure is the contract being enforced).
- Update the "Subject encoding" section above to reflect the new layout. The doc and the test goldens are the contract, in two forms.
- Treat it as a release-notes-worthy breaking change. There is no transitional state where mixed-version services interoperate, since the wire formats won't match.

### Verified source on receive

NATS PUB ACLs in production pin each NATS user to publish only under their own `<fromHost>` segment. The segment is therefore an ACL-enforced sender identity by the time a message arrives at a subscriber, in a way the `Microbus-From-Host` header (publisher-set, no broker-side check) is not. `onRequest` (request path) and `handleResponse` (response path) **unconditionally overwrite** `Microbus-From-Host` with the source segment parsed from `msg.Subject` via `splitSubject` before any downstream code reads it.

The overwrite is unconditional even when `splitSubject` returns `ok=false`. A malformed subject or transport bug that fails to populate `Msg.Subject` produces an empty verified source, which the framework propagates as an empty `From-Host`. The downstream `ackRequest` path treats an empty `From-Host` as a hard error - the correct response to an unverifiable inbound message. Falling back to the publisher-set value would defeat the verification contract; if we can't verify, we don't accept.

The short-circuit transport carries `Msg.Subject` the same way the NATS path does (set from the `subject` argument in `deliverWithShortCircuit`), so in-process traffic gets the same overwrite semantics. Cryptographic enforcement of the source segment doesn't apply within a bundle (no broker), but the framework still routes through the verified-source path so observability and per-caller throttle behavior is uniform across transports.

### Direct addressing for fragmented requests

The first fragment of a multi-fragment request publishes on the normal subject so any replica's queue group can pick it up. Once a replica acks, fragments 2..N are published with the responder's `id-XXXX` value as the `id_or_locality` slot, so they all land on the exact replica that took the first fragment. Without direct addressing the queue group would round-robin subsequent fragments across replicas, and the receiving replica would never see a complete request. Any locality slot present in the original URL is stripped at fragment-publish time - once we have an instance ID, locality is no longer relevant.

The ack op-code reflects this: a fragmented request acks with `100 Continue`, an unfragmented one with `202 Accepted`. The defragger times out a partial fragment set after `8 * networkRoundtrip` of inactivity (polled every `networkRoundtrip / 2`).

### Frame propagation in Publish is an explicit allowlist

Outgoing requests do *not* inherit the inbound request's full header set. `Publish` copies only:

- `X-Forwarded-*` (set by the ingress proxy)
- `Accept-Language`
- The actor (access token) header
- Any header with the baggage prefix

This is a security boundary, not an oversight. Internal control headers (`Microbus-Op-Code`, `Microbus-From-*`, etc.) are set per-call from connector state, so a malicious upstream that smuggled them onto an inbound request cannot have them propagate.

### `Go` and `Parallel` decouple goroutines from the request that started them

`c.Go(ctx, fn)` clones the *frame* of `ctx` (so baggage and actor flow), copies the trace span, but parents the new goroutine on `c.Lifetime()` rather than `ctx`. The goroutine therefore outlives the originating request but is cancelled by Shutdown. `Parallel` follows the same accounting via `pendingOps` so the shutdown drain waits for it.

### `:888` control surface

Control endpoints subscribe twice: once on the connector's own host, once mirrored on `//all<route>` (e.g. `//all:888/ping`) so they are reachable via the broadcast hostname `all`. The mirror preserves the port - only the hostname segment is replaced. The OpenAPI handler skips the `//all` mirror entries when rendering the document because they are not separate operations.

The OpenAPI handler is *actor-aware*: operations whose `RequiredClaims` the caller cannot satisfy are omitted from the document. The response is `Cache-Control: private, no-store` because the rendered document varies per caller. Only `function`, `web`, and `workflow` subscription types are included; tasks, events, and infra/control subs are filtered out at this boundary, which is what allows `llm.core` to safely use the document for LLM tool resolution.

### Trace sampling - `selectiveProcessor` is a tail sampler

In `PROD`, spans are not exported eagerly. `selectiveProcessor` buffers ended spans in a ring; when a trace ID is `Select`ed (either by `ForceTrace` locally, or by an inbound `:888/trace?id=...` from a peer that hit an error), the buffer is scanned and matching spans are flushed downstream. Future spans on that trace ID are also flushed.

The selected-trace-IDs map uses a two-generation rotation (`selected1`, `selected2`) primarily as a memory cap - old entries roll out without an explicit TTL sweep. Buffer capacity is fixed at ~8192 spans / ~10MB per microservice.

`ForceTrace` broadcasts to `https://all:888/trace?id=...` so every microservice with spans on that trace exports its share - without this fan-out only the connector that hit the error would flush.

### OTLP exporter resilience

Telemetry export is best-effort sideband - its failure must never affect service health. The three signals - traces (`tracer.go`), metrics (`metrics.go`) and logs (`logger.go`, when a logs endpoint is configured) - share two configuration choices that enforce this:

1. **`WithRetry(RetryConfig{Enabled: false})`.** With retries on, the OTLP gRPC client retries failed exports with internal timeouts that can exceed 75 seconds per call. A flaky or down collector therefore stalls the connector's batch span/metric processor flushes, which in turn stalls `Shutdown` (the SDK contract is to drain before returning). Disabling retries means each export attempt is single-shot - succeed or drop on the floor with a log line.

2. **No `WithBlock`, no `WithTimeout` option set explicitly.** The SDK's default constructor connects lazily (first export, not at `New(...)`), so `Startup` is never blocked on the dial. The per-export timeout is governed by `OTEL_EXPORTER_OTLP_TIMEOUT` (or the signal-specific `OTEL_EXPORTER_OTLP_TRACES_TIMEOUT` / `OTEL_EXPORTER_OTLP_METRICS_TIMEOUT`) per the OTel spec, in milliseconds. The SDK auto-reads it from `os.Getenv`; values set in `env.yaml` or via `env.Push` (in tests) reach the SDK because the env package writes through to the OS env. The SDK's spec-default 10s applies when unset.

The combined effect: a service with a configured-but-unreachable collector starts up immediately, makes one bounded export attempt per flush (giving up after the configured timeout), and shuts down within `timeout × N` worst case (one final flush per exporter type). Without these settings, the same misconfiguration would hang both startup and shutdown indefinitely - observed pre-fix as orchestrator-killed pods on rolling deploys.

The regression tests live in `metrics_test.go` (`TestConnector_OTLPMetricsUnreachable` for the fast connection-refused path, `TestConnector_OTLPSlowEndpoint` for the slow timeout path that actually exercises the spec-defined timeout) and `logger_test.go` (`TestConnector_OTLPLogsUnreachable` for the logs signal).

### Shared OTLP connection

All OTLP exporters in one executable that target the same endpoint over the same protocol share a single connection, created lazily and reference-counted in `otel.go` (`acquireOTLPConn` / `releaseOTLPConn`). A bundle of N microservices exporting three signals would otherwise open up to 3N TCP/TLS connections to the collector; sharing collapses that to one per distinct target.

The registry key is `protocol|endpoint|caFile|clientCert|clientKey|insecure` - the TLS material and the insecure override are part of the key so two signals pointed at the same endpoint but secured differently never alias onto one connection. Each signal of each connector holds one reference; the connection (a gRPC `*grpc.ClientConn` or a pooled `*http.Client`) is dialed on first acquire and closed on last release. Per-signal teardown (`termLogger` / `termTracer` / `termMeter`) flushes its provider *then* releases the connection, so the final batch still exports before a possible close.

Two invariants make sharing safe:

- **The connector owns the connection, not the SDK.** Exporters are constructed with `WithGRPCConn` / `WithHTTPClient`, and the SDK never closes a caller-provided connection on its own `Shutdown` (it only closes connections it dialed itself). So one connector's shutdown cannot pull the connection out from under another that still holds a reference.
- **Dialing is lazy.** `grpc.NewClient` and the cloned default `http.Transport` connect on first export, not at construction, preserving the non-blocking-`Startup` guarantee above.

Transport security follows the endpoint URL scheme (https is secured) unless `OTEL_EXPORTER_OTLP[_signal]_INSECURE` overrides it. Because the connector builds the connection itself, the SDK's env-driven dial setup is bypassed once `WithGRPCConn`/`WithHTTPClient` is supplied - and the SDK exposes no public factory to build a connection or dial options from env (`otlpconfig` is `internal/`). So `otel.go` reads the OTLP certificate env vars directly to honor a custom CA (`..._CERTIFICATE`) and a client certificate for mTLS (`..._CLIENT_CERTIFICATE`/`_KEY`). Per-signal exporter options that aren't connection-level - headers, timeout, retry - still flow through the SDK, which applies them at the client layer regardless of the injected connection.

### Log enrichment and fan-out live in the handler

`LogDebug/LogInfo/LogWarn/LogError` are thin delegations to the `*slog.Logger` returned by `Logger()`; all enrichment lives in `logHandler`, the connector's root slog handler. This is deliberate: a record logged through `Logger()` (e.g. `svc.Logger().InfoContext(ctx, ...)`) is enriched identically to one logged through `LogInfo`, because both flow through the same handler. The `*Context` slog methods carry the span and actor in the context; the non-context methods carry a background context, so they pass through un-enriched - the intended behavior.

`logHandler.Handle` does the context-derived work once - mirror the record onto the active span as an event (non-PROD), increment `microbus_log_messages`, dump errors to stderr in developer deployments - then fans out to the deployment terminal handler (`colorfulLogHandler` in LOCAL, text in TESTING, JSON in LAB/PROD) and, when a logs endpoint is configured, an `otelslog` bridge over the shared connection. The `MICROBUS_LOG_DEBUG` gate lives in `Enabled`; a gated debug record never reaches `Handle`, so it is neither counted nor exported. In PROD the terminal handler's `Info` level also drops debug regardless of the flag.

The two legs diverge on one point: the trace id is added as a `trace` string attribute only on the console leg, where it is the sole correlation handle. The OTLP leg omits it because the otelslog bridge stamps native `TraceId`/`SpanId` from the context, which backends use to correlate logs with traces. Span-event mirroring is retained alongside OTLP logs because it is a different view - inline in the span timeline, and it survives the PROD tail sampler even when no logs pipeline is configured.

`microbus_log_messages` counts every record emitted through the logger, including bare `Logger().Info(...)` that bypasses the `LogXXX` methods - the metric reflects all logging, not just the convenience entry points.

On shutdown the final "Shutdown" log is recorded while the meter and tracer are still live (it feeds both - a log counter and a span event), then the providers are torn down in reverse of initialization order: logs, then traces, then metrics. Tearing the meter or tracer down first would silently drop that last log's counter increment and span event.

### Telemetry provider accessors are concrete, not interface

`Logger() *slog.Logger`, `MeterProvider()` and `TracerProvider()` expose the underlying handles so application code can instrument third-party libraries against the same pipelines and resource as the framework. `MeterProvider`/`TracerProvider` return a no-op provider (never nil) when the signal is disabled. They are concrete methods on the connector and deliberately *not* part of any `service` interface: `MeterProvider`/`TracerProvider` return raw OTEL types, and the `service` package is kept insulated from OTEL behind the `trc` wrapper (e.g. `service.Tracer` returns `trc.Span`, not `trace.Span`). A provider cannot be wrapped without defeating its purpose - handing the real provider to OTEL-instrumented libraries - so the accessors stay off the interfaces and remain reachable via the embedded connector on the concrete `*Service`.

### `MeterProvider()` injects identity attributes; the resource alone is not filterable

The connector's own `microbus_*` metrics carry `service`, `deployment`, `plane`, `ver`, `id` as explicit per-measurement attributes, so a Grafana panel can scope them with a plain `service="…"` label selector. The microservice identity also lives in the OTEL **resource** (`service.name`, `deployment.environment.name`, …), but a resource attribute is *not* a label on each series - the OTLP/Prometheus pipeline splits it into a separate `target_info` series, reachable only by a join. So a metric emitted with only the resource (no per-measurement attributes) cannot be filtered by service without that join.

Third-party libraries we hand the meter to - sequel (`sequel_*`) and the dwarf engine (`dwarf_*`) - emit through our `MeterProvider()` but do not add those identity attributes, so on their own their series would carry no `service`/`deployment` label. The `metric.MeterProvider` interface exposes only `Meter(...)`; there is no public accessor for the resource on the SDK provider, so a library cannot read `service.name` to re-stamp it. The connector knows the identity, so `MeterProvider()` returns `attributedMeterProvider` (`metricsdecor.go`), a decorator that injects the identity attributes into every measurement of every instrument it vends. Result: `sequel_*`/`dwarf_*` carry the same `service`/`deployment` labels as `microbus_*` and the Grafana dashboards filter all three uniformly with a plain selector, no `target_info` join.

The decorator wraps synchronous instruments (appending the attribute option on `Add`/`Record`) and, for observable instruments, wraps the `metric.Observer` passed to a `RegisterCallback` callback - the path both sequel's pool gauges and dwarf's gauges actually use. A callback registered through a `WithInt64Callback`/`WithFloat64Callback` *construction option* is not intercepted (neither library uses that form); its observations would emit without the injected attributes.

The decorator is the **single** injection mechanism. The connector's own `c.meter` is built from it too (`decoratedMeterProvider().Meter("microbus")` in `startMetricsCollector`), so `microbus_*` metrics get the identity attributes the same way sequel/dwarf do. The per-call recording path (`makeAttributeSetOption`) therefore assembles only the caller's per-measurement attributes; the decorated meter merges the common identity attributes in. (Earlier the recording path prepended the identity by hand - that manual echo was removed when the meter moved onto the decorator, so identity is stamped in exactly one place and never doubled. The two attribute groups are disjoint, so the merge is a plain union.) The identity is immutable once the connector starts, so `decoratedMeterProvider` derives it from the accessors (`Hostname`, `Deployment`, …) on each call rather than caching it in a field.

### Actor tokens are verified whenever present, not only under requiredClaims

A present actor token is signature-verified on the receive path even when the endpoint declares an empty
`requiredClaims`. The gate on the request path (`subscribe.go`) branches on whether a token is present, not on
whether the endpoint gates itself: a missing token fails only when `requiredClaims` is non-empty, but a token that
is present is always run through `verifyToken`, and a token that fails verification is rejected `401` before the
handler runs. When `requiredClaims` is empty the connector passes the literal `"true"` as the expression so the
claim evaluation inside `verifyToken` passes unconditionally while the signature check still executes.

This exists because signature verification and claim reads were otherwise decoupled: handlers read actor claims via
`frame.IfActor` / `frame.ParseActor`, which parse with `ParseUnverified` (the `frame` package is a thin header
wrapper with no key access, so it cannot verify). If verification only ran under a non-empty `requiredClaims`, an
endpoint that authorized in-handler via `IfActor` instead of declaratively would trust claims nothing had checked,
and any mesh peer authorized to reach it could forge an actor with arbitrary claims. Verifying on presence closes
that gap: by the time a handler reads claims, the framework has already validated the signature. The external path
was never exposed here - the ingress `InternalHeaders` middleware strips inbound `Microbus-*` headers and re-mints
the actor from a verified bearer - so this specifically hardens peer-to-peer and in-bundle traffic.

### `alg=none` JWTs in TESTING

`verifyToken` accepts unsigned tokens (`alg=none`) only when `deployment == TESTING`. Required-claim evaluation still runs against the unsigned payload - TESTING relaxes the *signature* check, not the *authorization* check. This is what lets test code use `pub.Actor(claims)` without standing up a signing key.

### Configurator is disabled in TESTING

`refreshConfig` skips the call to `configurator.core` when `deployment == TESTING` and uses YAML defaults plus values set via `SetConfig`. Tests that want to override config call `SetConfig` / `ResetConfig` directly; outside of TESTING those calls error out.

### JWKS fetch is debounced per issuer

`verifyToken` resolves an actor token's `kid` to a public key, fetching JWKS on a cache miss. `fetchActorKeys`
debounces those fetches per issuer at `jwksFetchCooldown` (1s), so a flood of tokens carrying unrecognized kids
cannot amplify one-to-one into bus calls to the token services on the auth hot path - reachable by any mesh peer,
since a present actor token is verified even under empty `requiredClaims` (see "Actor tokens are verified whenever
present"). The 1s window is safe by the contract on the token services' `JWKS` endpoint, whose godoc declares it
cacheable for that long.

The cooldown timestamp is written only after a successful fetch, so a failed fetch (e.g. the token service briefly
unreachable at a key-rotation boundary) does not suppress retries for a second of synchronized 401s across the mesh.
Concurrent callers for the same issuer are collapsed into a single in-flight fetch via `singleflight`, which both
spares them a spurious miss while the fetch is in flight and preserves the anti-amplification property that the
deferred timestamp write would otherwise open up (every unknown-kid request launching its own fetch until the first
lands). The same structure is mirrored in the HTTP ingress's bearer-token JWKS fetch.
