# metrics.core

## Design Rationale

### Scrape aggregation is the opt-in path

The default telemetry pipeline is OpenTelemetry push over gRPC, configured on the connector, and it is the
recommended one: a scrape aggregation transiently holds every replica's metrics output in memory, a cost that grows
with the mesh, while push spreads the same data over time and connections. This microservice exists for deployments
that nonetheless prefer pull-based Prometheus scraping: it concatenates every replica's `:888/metrics` output into a
single scrape target. It does nothing unless explicitly added to the app, so its cost and attack surface are opt-in.

### Collection is a single multicast, not a discovery-then-fetch fan-out

`Collect` publishes one multicast `GET https://all:888/metrics` (or to the specific host named by the `service`
query argument) and streams the responses into the output as they arrive. The `:888/metrics` control subscription
is registered `sub.NoQueue()` and mirrored on the `all` hostname, so the single publish reaches every replica and
each responds with its own metrics; the replica-unique `id` attribute keeps the concatenated series distinct.

An earlier implementation discovered services with `PingServices` and then fetched each service's metrics in its
own goroutine-per-service multicast, serialized onto the response by a mutex. That was two mesh-wide response waves
instead of one - the ping already reaches every replica, it just returns smaller bodies - plus a goroutine, a
stagger delay, and lock traffic that the single channel loop makes unnecessary. Peak memory is unchanged by the
consolidation: in both shapes every replica responds at once and each response is a fully buffered message by the
time it is yielded. Capping concurrency was considered and rejected - a worker pool would serialize the scrape and
stretch its wall time past the Prometheus scrape timeout on large meshes, a worse failure mode than the transient
memory of holding the responses.

### The secret key prefers the Authorization header over the URL

`Collect` accepts the secret key in an "Authorization: Bearer" header, falling back to the legacy `secretKey`
query argument (and its `secretkey` / `secret_key` spellings). The header is preferred because URLs land verbatim
in access logs, proxy logs, and monitoring UIs, while headers conventionally do not. Prometheus expresses this
natively (`authorization: {type: Bearer, credentials_file: ...}` in the scrape config), as do Grafana Alloy and
compatible scrapers. The query arguments are retained so existing scrape configs keep working; dropping them would
be a silent breakage an operator could not detect until metrics went dark.

### Key comparison is constant-time over digests

A plain string comparison short-circuits at the first differing byte, which turns response timing into a per-byte
oracle on the key. `Collect` instead compares SHA-256 digests of the provided and configured keys with
`subtle.ConstantTimeCompare`. Hashing first also equalizes the two inputs' lengths, so neither the key's content
nor its length influences timing. An incorrect key returns 404 rather than 401, masking the endpoint's existence
from probing.

The key is required except in LOCAL and TESTING deployments, where the friction of provisioning a secret outweighs
the risk on a developer's machine.
