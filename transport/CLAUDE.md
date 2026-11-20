## Design Rationale

### NATS connections are pooled by credential hash

Multiple `Conn.Open` calls in the same process share one underlying `*nats.Conn` if their full configuration - URL, user/pw, token, JWT+seed, `ca.pem`, `cert.pem`+`key.pem`, `nats.creds` - hashes to the same key (sha256 of all fields concatenated with `|` separators). Reference counts on `refCountedNATSConn` track how many connectors are using it; the underlying connection closes only when the last reference goes away.

The motivation is twofold and the same mechanism handles both:

- **Bundled executables with shared creds.** When several services in one process all use the same `nats.creds` (or no creds at all), they hash identically and share one NATS connection. This is the common path for development bundles and `application.RunInTest`.
- **Bundled executables with per-service creds.** When each service ships its own `<hostname>_nats.creds`, the resolved file paths and contents differ per service, so the hash differs and each connector gets its own connection. This is the path that gives broker-pinned per-service identity in production.

The hash is an exact-match key - different credentials, even if functionally equivalent, will not share.

### Per-service auth artifact lookup

`Open` resolves four optional auth artifacts via `resolveArtifact(hostname, artifact)`. For each, the per-service form `{hostname}_{artifact}` in CWD wins, falling back to the bare `{artifact}` form:

| Artifact      | Per-service form              | Shared default form |
| ------------- | ----------------------------- | ------------------- |
| NATS creds    | `{hostname}_nats.creds`       | `nats.creds`        |
| TLS cert      | `{hostname}_cert.pem`         | `cert.pem`          |
| TLS key       | `{hostname}_key.pem`          | `key.pem`           |
| TLS root CAs  | `{hostname}_ca.pem`           | `ca.pem`            |

The forward construction is `{hostname}_{artifact}`; the reverse is "trim trailing `_{artifact}`." The last `_` in the filename is always the separator. Hostname identity rejects `_` (per `httpx/CLAUDE.md`'s "Hostname identity vs. route hostnames"), so the rule is unambiguous.

Cert/key are paired - both must resolve (per-service or default) for a client cert to be configured. Mixing per-service cert with default key, or vice versa, works because each is resolved independently; the pair is just that both must exist.

Only artifact **contents** (not their paths) feed the pool hash. Two connectors that resolve to byte-identical creds - same file or distinct files with identical contents - share a NATS connection, since they present the same identity to the broker. Per-service identity isolation is enforced by the JWT contents at the broker, not by the framework's pool key.

### Env-var auth options are deprecated

`MICROBUS_NATS_USER`, `MICROBUS_NATS_PASSWORD`, `MICROBUS_NATS_TOKEN`, `MICROBUS_NATS_USER_JWT`, `MICROBUS_NATS_NKEY_SEED` remain functional but are no longer documented. The file-based lookup chain covers every shape they expressed, and the env-var form can't carry per-service identity (env is process-scoped). New deployments should use `<hostname>_nats.creds` (per-service) or `nats.creds` (shared default).

When any of these env vars is set at `Open` time, the transport emits a single `LogInfo` deprecation line listing the offending names. Active enough to surface in production startup logs and in deploy-canary log scrapes; non-fatal because the env-var path still works for backward compatibility. Removing the env-var auth path entirely is left to a future major release.

### Two transports in one: short-circuit and NATS, registered together

When `shortCircuitEnabled` is true (default; toggle via `MICROBUS_SHORT_CIRCUIT=0`), `Subscribe` / `QueueSubscribe` register the handler on *both* the process-global short-circuit trie and (if connected) NATS. A single `*Subscription` wraps both registrations. This is what makes co-located service-to-service calls bypass NATS entirely while still being reachable from remote peers via NATS.

The short-circuit trie is a **process-global** singleton - `var shortCircuit trie` at package level. All connectors in one process share one trie. This is intentional: short-circuit only works between connectors in the same process, so a process-wide trie is the right granularity.

### Routing rules differ between unicast and multicast

The three send paths - `Request` (unicast), `Response` (unicast), `Publish` (multicast) - make different choices:

- **`Request` and `Response` (unicast)**: Try short-circuit first; if at least one matching handler is found locally, deliver and return. Otherwise fall back to NATS. This is the fast path that gives co-located services a 10× latency improvement over NATS.
- **`Publish` (multicast)**: Use NATS *only* when NATS is connected. Short-circuit is used only when NATS is not connected. The reason is correctness: short-circuit cannot reach remote peers, and a multicast must reach *all* subscribers (local and remote). Sending only locally would silently drop remote subscribers. The rule is universal - there is no opt-in to short-circuit-only multicasts when NATS is up.

### Single-handler short-circuit is zero-copy

When `deliverWithShortCircuit` finds exactly one matching handler, it passes the *live* `*http.Request` (or `*http.Response`) inside the `Msg` directly to the handler - no serialization. The handler can read the live object as-is.

With multiple handlers (e.g., a no-queue subscription with several subscribers), the message is serialized once into `Msg.Data` bytes, and each handler gets a separate `Msg{Data: ...}`. The handler receives serialized bytes in this case because mutating the live request mid-fan-out would corrupt other handlers' views.

The single-handler fast path is what gives the short-circuit its dramatic latency improvement. If you add subscriptions that always fan out (e.g., a non-queue subscriber on a hot path), you lose the zero-copy benefit.

### Trie wildcards mirror NATS, intentionally

The short-circuit's subscription matcher is a custom prefix trie that supports `*` (single segment) and `>` (greedy suffix) wildcards - the same syntax NATS uses. This is what allows short-circuit and NATS to handle the same subject patterns interchangeably. The trie matcher is custom-built rather than reused from a library because it also has to support queue-group round-robin and unsubscribe-with-trim, which most prefix-tree libraries don't.

`Handlers(subject)` walks segments left-to-right BFS-style, expanding into literal/`*`/`>` children at each step. Greedy `>` matches are collected separately and processed last. The traversal queue is reused via `sync.Pool` so high-volume publishers don't churn allocations.

### Queue group semantics: named = one member, unnamed = broadcast

Each trie node has a `map[string]*ringList` keyed by queue name. The empty queue name `""` is the broadcast bucket. `Handlers` selects:

- **One handler from each named queue**, across every node that matches the subject.
- **All handlers from the unnamed queue** of every matching node (every subscriber receives a copy).

A queue group is the set of subscriptions with one queue name whose subjects match the message, wherever they sit in the trie. `a.b.>` and `a.b.c.>` under queue `q` form one group for `a.b.c.d`, as do `a.b.*` and `a.b.c` for `a.b.c`. This is how NATS groups queue subscribers (`addNodeToResults` in nats-server's `sublist.go`), and the trie has to match it, or a request to overlapping routes in one queue runs every one of them when delivered over the short-circuit and only one over NATS. Neither transport picks the most specific subscription: among overlapping members of one queue the pick is uniform.

The common case is one matching node, and `appendNodeHandlers` rotates each named queue's ring, a deterministic round-robin. When several nodes match, `appendMergedHandlers` walks each node's queues and, for a queue it has not met at an earlier node, looks the same queue name up in the later nodes' maps. It picks one of the rings it finds with probability `len(ring) / members`, then rotates that ring, so every member is equally likely and the members at one node keep their round-robin. The cost is linear in the number of queues, times the number of matching nodes, which is small. Grouping by comparing queue names pairwise instead would be quadratic, and it runs under the process-wide trie lock, while a subject can carry hundreds of queues in a large bundle (every microservice subscribes to the `//all:888` control routes in its own queue).

NATS starts its pick from a random index rather than rotating, so the two transports agree on who is eligible and on the distribution, not on the sequence.

### Trie self-trims on unsubscribe

When a subscription is removed, `unsub` deletes its handler from its `ringList`. If the ring becomes empty, the queue entry is removed from the leaf node. If the leaf becomes empty (no queues, no children), it's removed from its parent. The trim recurses up the trie until a non-empty node or the root is reached. This keeps the trie minimal under heavy churn - long-lived subscribers don't accumulate stale internal nodes.

### `Latency()` drives the connector's ack timeout

The connector calls `Latency()` after `Open` and uses the result as its `ackTimeout`. The transport returns:

- **10 ms** for short-circuit-only deployments (no NATS).
- **NATS RTT × 2 + 50 ms, rounded up to the nearest 100 ms** for single-server NATS.
- **300 ms default** for multi-server NATS or RTT failure.

The 100 ms minimum is what was observed necessary on localhost under high load - anything below that over-fires on transient hiccups. The `× 2 + 50` shape lets the formula scale up for world-wide deployments where RTT itself is large. A single RTT measurement at startup is a weak basis for predicting steady-state ack latency, so this formula is best-effort and may need adjustment as deployments evolve. Multi-server NATS just gets the conservative 300 ms default because RTT to the connected server doesn't predict cluster-wide latency.

### `WaitForSub` is a 20 ms sleep, observed-required

NATS subscription registration is asynchronous - `Subscribe` returns before the server has acknowledged the subscription. Until the server acks, messages published to the subject won't reach this subscriber. The 20 ms sleep is empirically the minimum that was observed to make NATS reliably recognize a new subscription before traffic flows; without it, race-window-dependent test flakes appeared. There is no synchronous wait primitive in the NATS Go client for this.

It's a no-op when NATS isn't connected (short-circuit subscriptions are synchronous). The connector calls it after activating subs in batches.

### `SetPendingLimits(-1, -1)` is "no arbitrary limit"

NATS subscriptions default to bounded pending-message and pending-byte limits - when exceeded, NATS *drops* messages and the subscription stays alive. The framework calls `SetPendingLimits(-1, -1)` to disable both limits.

The reasoning is not "memory pressure is fine," it's "any cap we pick would be arbitrary." Microbus consumes from the bus quickly enough that hitting a finite limit in normal operation would imply something is already wrong - and at that point silently dropping messages is worse than the symptoms of unbounded queues. If you observe memory growth on a slow handler, the right response is to fix the handler, not to re-enable the cap.

### `mem.Alloc` / `mem.Free` for publish buffers

`Publish` / `Request` / `Response` allocate a 2KB-minimum buffer via `mem.Alloc` to serialize the HTTP message before handing it to NATS, then `mem.Free` it. This is a per-process buffer pool to reduce GC pressure on the publish hot path. The buffer is sized as `1KB + ContentLength`, so most small requests fit in the minimum 2KB block and small-payload publishes don't trigger any allocation at all.

### `Msg` carries one of three forms

`Msg` has three nullable payload fields - `Data`, `Request`, `Response`. Exactly one is populated:

- **`Request` / `Response`**: live HTTP object, used by the single-handler short-circuit zero-copy path.
- **`Data`**: serialized bytes, used by NATS receive, multi-handler short-circuit fan-out, and the connector's defragmenter.

Handlers that only handle one form will need to call `http.ReadRequest` / `http.ReadResponse` themselves on `Data` if Request/Response is nil. The connector's `onRequest` does this - it lazily parses `Data` when needed and caches the result on the `Msg`.

### `Msg.Subject` is populated by every delivery path

Every `*Msg` handed to a `MsgHandler` carries the NATS subject the message was delivered on. The transport populates `Subject` at six call sites - once per `Subscribe`/`QueueSubscribe` callback (NATS path) and once per branch in `deliverWithShortCircuit` (single-handler zero-copy and multi-handler fan-out). The contract is unconditional: receivers can rely on `Msg.Subject` being non-empty for any normally-delivered message.

This exists because NATS PUB ACLs in production pin each NATS user to publish only under their own source-segment prefix. The framework's connector reads the source segment off `Msg.Subject` to recover an ACL-verified sender identity, then overwrites the `Microbus-From-Host` header with that value before dispatch - see "Verified source on receive" in `connector/CLAUDE.md`. Without `Msg.Subject`, the connector would have no transport-side handle for the verified context, since the subject doesn't survive into the parsed `*http.Request` on its own.

The short-circuit transport carries `Subject` even though there is no broker to enforce ACLs in-process. This is by design: the connector's verified-source path runs uniformly across both transports, and short-circuit's `Msg.Subject` is the same string the publisher addressed (constructed by the publisher's connector from its own hostname). In-bundle traffic gets the same observability and per-caller throttle behavior as cross-bundle traffic without a separate code path.
