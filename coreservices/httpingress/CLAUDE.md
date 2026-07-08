# http.ingress.core

## Agent Instructions

This microservice uses authentication APIs. See `.claude/rules/auth.txt` for the conventions.

This microservice does not maintain a `PROMPTS.md`. Skip the prompts step when running housekeeping.

## Design Rationale

### Token exchange copies verified bearer claims verbatim

`exchangeToken` verifies the external bearer token's signature, pins its issuer to `bearer.token.core` (an external
identity provider must wrap through that service; it cannot drop a token straight through the ingress), and then
calls `access.token.core`'s `Mint` with the verified claims passed through unchanged. The exchange transfers claims,
it does not vet them, and it deliberately cannot: an access token is enriched relative to the bearer (via the access
service's claims transformers), so its claims are not a subset of the bearer's. What each token may carry is
application policy - the authenticator's mint call and the registered transformers - not something the ingress or the
framework enforces. The exchange is therefore not a safer mint than a direct `:666` call.

### `resolveInternalURL` lowercases but does not validate

The hostname segment of an external URL is lowercased before publishing because Microbus identities are canonical lowercase, but external URLs may arrive with uppercase characters (browsers don't always preserve case in path segments). Without normalization, requests for `/MyService/path` would fail to route despite a perfectly valid `myservice` subscription on the bus.

The hostname is not otherwise validated here. A malformed or unknown hostname surfaces as a no-responder ack-timeout (404) from `Publish`, which is the same shape as any other unresolved internal target. Validating up front would shift the error to a different layer for marginal benefit - the ingress is gated by network position, not by hostname legality.

### TLS certificates come from `httpx.CertStore`

Cert loading, SNI matching, resolution order, and fsnotify-driven reload all live in `httpx.CertStore`; the rationale is documented in `httpx/CLAUDE.md`. This microservice constructs the store with `httpx.NewCertStore("", svc)` and uses the package's filename convention directly: `{token}-cert.pem` / `{token}-key.pem` for paired files, `{port}-cert.pem` (numeric token) as a per-port default, and bare `cert.pem` / `key.pem` as the server-wide default. The token is the substring after the last `-` in the prefix portion, so legacy `httpingress-{port}-cert.pem` files are still picked up - their token resolves to the port number and they continue to work without renaming.

The bare-port TLS auto-detection in `parsePorts` checks for both the new `{port}-cert.pem` / `-key.pem` form and the legacy `httpingress-{port}-cert.pem` / `-key.pem` form via an `os.Stat`, so existing single-cert-on-443 deployments need no file or config change to keep terminating TLS on 443.

### Two port states, with redirect derived rather than declared

A port is either TLS or plaintext. There is no third "serve plaintext vs redirect" state because redirect is a function of the configuration, not a property of a port: `:80` redirects to `:443` only when `:443` is open and TLS. Modeling redirect as derived avoids a latent trap in the alternative "a plaintext port redirects when any TLS port exists" rule, which would 301 a development `:8080` to HTTPS in a prod-like config. The upstream-TLS-termination topology (something in front terminates TLS and forwards plaintext) remains expressible: a bare `443` with no marker and no cert stays plaintext and does not redirect.

### `Ports` keeps its name and bare-port legacy semantics

The config name and the meaning of a bare port number are unchanged, and old values parse byte-identically. This is deliberate: the config lives in the production deployment (possibly via env or operator overrides outside the repo), while any migration tooling runs on a developer's checkout. A rename or a silent semantic change to an existing config would therefore be an unmigrate-able silent failure. The `tls` marker is purely additive opt-in; a bare port still enables TLS only when its legacy `httpingress-{port}-cert.pem` and `-key.pem` files are present.

### Internal-port firewall is default-deny with a tunable allowlist

The ingress is a generic forwarder whose source-derived NATS ACL collapses to a wildcard `safe.*` grant by construction, so the ACL layer cannot constrain which internal ports it reaches. That makes the ingress's own port filter the load-bearing structural control for the framework's claim-less ACL-only admin endpoints. The implementation is the `AllowedInternalPorts` config plus a hard floor for `:666` and `:888`:

- Hard floor: `:666` (trust-root) and `:888` (management) are blocked in every deployment mode, regardless of config. They cannot be allowlisted; operators cannot opt them in. The framework places its claim-less admin endpoints there precisely so this single structural control protects them.
- Implicit allow: `:443` is always reachable. It is the actor-facing surface and the framework's default endpoint port.
- Operator allowlist: `AllowedInternalPorts` is a comma-separated list of ports or ranges (`N` or `N-M`) added to the allow set. Every entry must satisfy `1024 <= port <= 65535`; anything outside that window causes the microservice to refuse to start, rather than fail closed silently or expose a port the operator didn't actually intend. The `<1024` floor on config values is the same axis as the framework's port-as-trust-tier convention - the low-port range is reserved for the framework's internal-only and trust-root tiers and is not operator-tunable.
- LOCAL deployment: `AllowedInternalPorts` is ignored entirely. Every port except the hard floor is reachable, for dev ergonomics. Production-grade safety only applies in non-LOCAL modes; demanding an explicit allowlist for every local-dev port would be a constant papercut and would not improve security on a developer's laptop.
- A request for a disallowed internal port returns 404, with no rewrite. The previous `PortMappings` model silently coerced disallowed ports to `:443`, which was a footgun (a request for `:888` could quietly become a different endpoint at `:443`). Reject-don't-rewrite is more honest, and the per-endpoint `requiredClaims` enforced at the connector is the second, authoritative layer of defense regardless.

### `PortMappings` is removed; setting it refuses startup

The previous `x:y->z` port-rewrite mechanism is gone. The `PortMappings` config name is still defined, but `OnChangedPortMappings` returns an error if it has any non-empty value, which makes the ingress refuse to start. Failing closed is deliberately stricter than "warn and ignore": a prod operator who set `PortMappings` to enforce a posture should see their configuration take effect or see a hard failure that points them at `AllowedInternalPorts`, never silent erosion of the posture they thought they were running.

### JWKS fetch is debounced per issuer

`exchangeToken` resolves an external bearer token's `kid` to a public key, fetching JWKS on a cache miss.
`fetchBearerTokenKeys` debounces those fetches per issuer at `jwksFetchCooldown` (1s), so internet-origin tokens
carrying unrecognized kids cannot amplify into bus calls to the bearer token service - the ingress being the
internet-facing verifier makes this the primary surface for that amplification. The 1s window is safe by the contract
on the bearer token service's `JWKS` endpoint, whose godoc declares it cacheable for that long.

The cooldown timestamp is written only after a successful fetch, so a failed fetch (e.g. the token service briefly
unreachable at a key-rotation boundary) does not suppress retries for a second of synchronized 401s. Concurrent
callers for the same issuer are collapsed into a single in-flight fetch via `singleflight`, which both spares them
a spurious miss while the fetch is in flight and preserves the anti-amplification property that the deferred
timestamp write would otherwise open up (every unknown-kid request launching its own fetch until the first lands).