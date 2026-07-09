## Configurator Core Service

Create a core microservice at hostname `configurator.core` that centralizes configuration value distribution to all microservices in the application.

The service maintains an in-memory `repository` (a map of `hostname -> property name -> value`) protected by a `sync.RWMutex`. A separate `sync.Mutex` (`refreshLock`) coalesces concurrent `Refresh` calls: callers arriving while a refresh is in flight wait for a subsequent round (guaranteed to start after they arrived) rather than piggybacking on the in-flight one, so a change is never dropped, and every waiter on a round observes that round's result.

On startup, walk from the current working directory up to the filesystem root looking for `config.yaml` and `config.local.yaml` at each level. Load all found files in order (later files override earlier values). After loading, broadcast the repository to replica peers via `SyncRepo`, then call `Refresh` to push configs to all microservices.

The repository YAML format is:

```yaml
hello.example:
  Greeting: Ciao
all:
  SharedProp: value
```

The `all` key is a wildcard that applies to every hostname. Value lookup is specificity-ordered: an exact hostname match beats a suffix match (`example` matches `hello.example`), which beats `all`. Segments are resolved right-to-left - `core` is less specific than `ingress.core` which is less specific than `http.ingress.core`.

Expose these endpoints:

- `Values` on `:888/values` - takes `names []string`, reads `frame.Of(ctx).FromHost()` to identify the caller, and returns the map of matching values from the repository. Read-locked.
- `Refresh` on `:444/refresh` - coalesces concurrent callers, then delegates to `PeriodicRefresh`.
- `SyncRepo` on `:888/sync-repo` (multicast, no queue) - accepts a `timestamp time.Time` and `values map[string]map[string]string` from a peer. Ignores requests from self (same `FromID()`). If repos differ, the newer timestamp wins: if incoming is newer, replace the local repo; if local is newer, re-broadcast the local repo to peers.
- `PeriodicRefresh` ticker at 20-minute intervals - multicasts `ConfigRefresh` to all microservices via `controlapi.NewMulticastClient(svc).ForHost("all").ConfigRefresh(ctx)`. Ignores `http.StatusNotFound` errors (microservices that don't have a config endpoint). Returns the last non-404 error encountered.

Deprecated endpoints `Values443` (`:443/values`), `Refresh443` (`:443/refresh`), and `Sync443` (`:443/sync`) forward to the current implementations but reject requests arriving from outside the bus (check `frame.Of(ctx).XForwardedBaseURL() != ""`).

The `repoTimestamp` tracks when the repo was last modified and is used to resolve conflicts during peer sync.
