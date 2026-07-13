## Design Rationale

### `ValidateHostname` is identity-strict, single-purpose, and not lenient

`httpx.ValidateHostname` is the single source of truth for what a canonical Microbus identity hostname looks like: lowercase letters, digits, dots, and hyphens only (`^[a-z0-9-]+(\.[a-z0-9-]+)*$`), length ≤ 252, no leading or trailing dot, no consecutive dots, no `id-` or `loc-` first segment, not equal to `all`, no `.all` suffix. Callers are responsible for normalization - the validator does not trim or lowercase, and errors on non-canonical input.

The strict rules exist to keep the wire format unambiguous:

- **No underscore.** `_` is reserved as the flat-form encoding of `.` inside NATS subject segments (see `connector/CLAUDE.md`'s "Subject encoding" section). Allowing `_` in identity hostnames would silently collide on round-trip - `my_service.core` and `my.service.core` both flatten to `my_service_core`, and the un-flatten side picks the wrong one.
- **No `id-` or `loc-` first segment.** Those prefixes are reserved for the per-instance and locality slots in the NATS subject layout. A hostname starting with either would collide with the publisher's slot extraction.
- **No `all` or `*.all`.** `all` is the broadcast hostname used by the control plane; `<id>.all` is how broadcast addressing routes to a specific instance.
- **Lowercase only.** Keeps the canonical identity form unambiguous. The framework already lowercases hostnames in subject construction, so accepting mixed case at registration time would just delay the normalization to a less-obvious place.

`ValidateHostname` is intentionally narrow. It does not have variants for "loose" or "route" hostnames, and `httpx` exposes no other hostname helpers - the package owns the *what* of a valid Microbus identity and nothing more. Higher-level packages (notably `sub` for subscription route hostnames) sometimes need to accept a slightly looser shape; they handle that with their own preprocessing on top of this validator. See `sub/CLAUDE.md` for how subscription routes wrap a translation step around the strict check.

The HTTP ingress proxy, by contrast, performs no hostname validation at all - it lowercases the host segment of an incoming URL and lets `Publish` fail naturally when an unrecognized hostname produces no responders.

### Three magic field names drive reflection-based payload routing

`HTTPRequestBody`, `HTTPResponseBody`, and `HTTPStatusCode` are recognized by `ReadInputPayload` / `WriteInputPayload` / `ReadOutputPayload` / `WriteOutputPayload` via `reflect.FieldByName` lookup. They are not types or interfaces - they are well-known *string* names that the reflection layer matches against:

- **`HTTPRequestBody`** on an input struct reroutes body parsing into that field instead of the parent struct, and on the outbound side reroutes only that field into the body (other fields go into the query string).
- **`HTTPResponseBody`** on an output struct reroutes response-body decoding into that field instead of the parent.
- **`HTTPStatusCode`** on an output struct receives the response status code (read side) or sets the response status code (write side) via `Int` semantics, so the field type must be `int`.

The string-name convention exists because Go does not allow struct tags on *function arguments*, and the framework's typed-endpoint signatures are function arguments - there is no syntactic surface to attach a `microbus:"body"` tag to. The magic names are the only available channel for expressing this contract in pure Go signatures. Renaming any of these fields silently disables the magic; there is no compile-time check.

### Input payload precedence: path < body < query

`ReadInputPayload` populates the input struct in this order:

1. Path arguments via `DecodeDeepObject` (so path args set the initial values).
2. Body via `ParseRequestBody` (JSON or form, overwriting path values where they collide).
3. Query string via `DecodeDeepObject` (overwriting both).

Last write wins. So a query-string argument can override a body field, which can override a path arg. This is deliberate but easy to forget when debugging "why did the query value take effect?"

### `BodyReader` is the framework's reusable-buffered body

`BodyReader` wraps a `[]byte` as both an `io.Reader` and `io.Closer`, plus exposes `Bytes()` and `Reset()`. It exists because the framework needs to:

1. **Read the body more than once.** The connector's request handler may parse headers, dispatch, and re-parse the body during defragmentation. `Reset()` rewinds without reallocating.
2. **Hand the underlying bytes to optimization paths.** `Copy` and `NewFragRequest` both type-assert `Body.(*BodyReader)` and operate on the raw byte slice when possible.

`Close()` is a no-op - there's nothing to close, but the type satisfies `io.ReadCloser` for `http.Request.Body` / `http.Response.Body`. The connector substitutes `BodyReader` into incoming requests when it has a buffered body to give them; user handlers can therefore depend on `Reset()` working on inbound requests but should not depend on it on arbitrary `io.ReadCloser` bodies.

### `ReadRequest` splits headers from body to hand back a `BodyReader`

`ReadRequest([]byte)` is the connector's inbound-request parser, used in place of the standard
`http.ReadRequest(bufio.NewReader(...))`. It finds the `\r\n\r\n` header terminator, parses only the header
section, and sets `req.Body` to a `BodyReader` over the remaining bytes of the same buffer.

The point is *not* to avoid a body copy. `http.ReadRequest` does not copy the body either - it leaves `req.Body`
as a reader over the buffered source, and `bufio` bypasses its own buffer for a large read, so raw read
throughput is already fine. The point is the *type*: a body parsed by `http.ReadRequest` is an opaque
`http.body`, so the `BodyReader` fast paths never engage. By handing back a `*BodyReader` instead, `ReadRequest`
makes the inbound body (1) re-readable via `Reset()` - the connector re-reads it during defragmentation and the
handler may read it more than once - and (2) eligible for the zero-copy paths in `Copy` (ownership transfer into
a `ResponseRecorder`) and `NewFragRequest` (skip the fragment-array allocation). Those matter on a forwarding,
re-serializing, or re-fragmenting path, not on a terminal handler that decodes the body once. It also gives the
connector an exact body length (`Bytes()`) without re-measuring.

This is the substitution the `BodyReader` note above refers to when it says "the connector substitutes
`BodyReader` into incoming requests." The tradeoff is that manual header/body splitting assumes Content-Length
framing and never chunked transfer-encoding - which holds because the Microbus serializer always emits
Content-Length, so the body genuinely is the literal tail after the header block. This couples the parser to the
serializer's exact output rather than delegating framing to the standard library; the framework owns both sides,
so the coupling is acceptable, and a buffer with no header terminator falls back to the standard parser.

### `Copy` transfers byte ownership when possible

When both the source `*http.Response.Body` is a `BodyReader` and the destination `http.ResponseWriter` is a `*ResponseRecorder` (and the recorder is empty), `Copy` constructs the recorder's buffer *directly over the BodyReader's bytes* - `bytes.NewBuffer(br.bytes)` - instead of copying. The comment in code calls this "somewhat risky: bytes are now owned by the buffer." After this transfer, mutating either side affects the other.

The motivation was reducing memory churn on the request hot path - which on a busy ingress proxy or service mesh is the biggest source of allocator pressure. If you call `Copy` and then mutate the source's body bytes downstream, expect surprises.

### `DecodeDeepObject`: bracket and dot notation are interchangeable

`DecodeDeepObject(values, target)` accepts both `a[b][c]=v` and `a.b.c=v` for the same key path. The first step normalizes brackets to dots (`strings.ReplaceAll("[", ".")` and drop `]`), so callers don't need to pick one syntax. This matches the OpenAPI deepObject style on the input side and the human-friendly dot style on the output side.

### Sequential integer keys decode as arrays, not objects

If a decoded sub-tree has keys exactly equal to `"0"`, `"1"`, ..., `"len-1"` (no gaps, contiguous from 0), it is converted to a slice rather than a map. So `x[0]=a&x[1]=b&x[2]=c` becomes `{"x": ["a","b","c"]}`, not `{"x": {"0":"a","1":"b","2":"c"}}`.

A gap or out-of-range key disables the conversion for that sub-tree - `x[1]=b` alone produces a map, since "0" is missing. Nested sub-trees are evaluated independently.

### Two-pass type fallback in `DecodeDeepObject`

`detectValue` infers a type per query value: `null` → nil, `true`/`false` → bool, JSON-number-looking strings → `json.Number`, else string. The decoded tree is JSON-marshalled, then unmarshalled into the user's struct.

If unmarshal fails with `*json.UnmarshalTypeError` (e.g., decoded a number but the target field is a string), the decoder retries with *all* leaf values forced to strings (`leafsToStrings`). This means a target field that's typed as `string` will receive `"42"` even if the URL had `?x=42`. The cost is one wasted marshal/unmarshal round-trip on the mismatch path; the benefit is that user types don't need to match query-string literal types.

`json.Number` is used (rather than `float64`) per the in-code comment: *"to preserve precision through marshal/unmarshal."* Avoids the IEEE-754 round-trip that would clobber large integers.

### `SetPathValues` / `PathValues` round-trip uses Go 1.22 `r.PathValue`

`SetPathValues(r, routePath)` parses the request's actual path against the parameterized route (`/obj/{id}/{}`) and stores values via `r.SetPathValue(name, value)`. `PathValues(r, routePath)` reads them back into `url.Values` for downstream merging into the input struct. The framework relies on Go 1.22's first-class path values for storage rather than maintaining a side map.

Unnamed path arguments - bare `{}` - are auto-named `path1`, `path2`, ... in left-to-right declaration order. Greedy arguments `{name...}` capture the entire path tail.

### `FillPathArguments` and `ResolveURL` unescape `{` and `}`

`url.Parse` percent-encodes braces (`{` → `%7B`, `}` → `%7D`), which is correct for arbitrary URLs but breaks Microbus's parameterized-route format. `ResolveURL` does a post-pass `ReplaceAll` to restore the literal braces. `FillPathArguments` lifts query-string values into matching `{name}` placeholders before publishing - so a client can pass path args as either part of the URL or as query args, and the framework normalizes.

### `ParseURL` rejects backticks

`ParseURL` rejects URLs containing the backtick character (`` ` ``) up front. The reason is that Go raw-string literals are delimited with backticks; allowing them inside URLs would mean a URL embedded in source code via a raw string could break the string boundary. Rejecting them is a small defense that lets Microbus URLs round-trip safely through Go raw strings without escaping concerns.

### `SetRequestBody` content-type sniffing for raw bytes/strings

When the body is `[]byte` or `string` and no `Content-Type` is set, `SetRequestBody` sniffs:

1. If the body starts with `{` and ends with `}` and `json.Unmarshal` to `map[string]any` succeeds → `application/json`.
2. If the body starts with `[` and ends with `]` and `json.Unmarshal` to `[]any` succeeds → `application/json`.
3. Otherwise `http.DetectContentType` (the standard library's MIME sniffer).

This is why a string `"{"foo":1}"` posted via the egress proxy automatically gets the right content type without the caller setting it. `url.Values` and `QArgs` always force `application/x-www-form-urlencoded`. Anything else is JSON-marshalled with `application/json`.

### `FragRequest` short-circuits when body is a `BodyReader` and fits

`NewFragRequest` has two paths: if the body is already a `BodyReader` and its bytes fit within `fragmentSize`, it sets `noFrags=true` and skips the fragment-array allocation entirely - `Fragment(1)` returns the original request unchanged. For other readers it consumes via `io.LimitReader` until EOF. This is the common-case fast path; non-`BodyReader` bodies always pay for the read-through.

### `DefragRequest` tolerates out-of-order fragment arrival

`DefragRequest.Add` stores fragments by index; arrival order doesn't matter. `Integrated()` walks 1..maxIndex and errors if any are missing. The integrated body is built with `io.MultiReader` over the per-fragment readers - no big buffer copy. The result is set onto fragment 1's request (so headers come from the first fragment), with `Content-Length` summed across fragments.

This is what allows the connector's NATS subscriber to call `Add` from its receive callback without sequencing.

`Add` also enforces the framing rather than trusting the sender: the receiver reassembles a transfer sized by its declared fragment count (`connector/CLAUDE.md`, "Fragment reassembly"). The count is pinned from the first fragment seen: a later fragment declaring a different `max`, an index outside `1..max`, or a duplicate index is rejected instead of silently rewriting the declared size or double-counting `arrived` (which would let completion fire with fragments genuinely missing). `DefragResponse.Add` enforces the same. A rejection is the connector's cue to tombstone the whole transfer.

### `CertStore` matches on SAN, not on file name

`CertStore` serves TLS certificates by SNI from `{token}-cert.pem` / `{token}-key.pem` pairs in a directory. The file name only pairs a cert with its key (by sharing the full prefix portion before `-cert.pem` / `-key.pem`); certificate matching is done on the parsed leaf's `DNSNames`, not on the file name. The reason is correctness: a Go TLS listener can already select among many certificates per handshake via SNI (`tls.Config.GetCertificate`), and a misnamed file should never be able to cause a wrong-certificate mis-serve. SAN-driven matching makes that impossible by construction, and wildcard plus multi-SAN certificates work without special handling.

The token used for port-default detection is the substring after the **last** `-` in the prefix portion. That makes both `443-cert.pem` and `httpingress-443-cert.pem` resolve to token `"443"`, which lets a deployment that previously used a service-name prefix coexist or migrate without renaming any files. Two files claiming the same numeric token resolve via newest `NotBefore`, same tiebreak as SAN collisions.

Resolution is exact SAN, then wildcard, then the listener's port-named default (a cert whose token parses as the listener's port), then the server-wide default (`cert.pem` + `key.pem`, no token at all). With no match the handshake fails closed - a wrong certificate is worse than a clean handshake failure. The numeric-token convention preserves the legacy "one cert per port" deployment style; the no-token convention exists for a single solution-wide cert (for example a SAN cert covering all public hostnames) that doesn't want to be named after a port.

### `CertStore` reload watches the directory, not the files

`Watch` runs an fsnotify watch on the directory rather than on individual files. Real cert-rotation deployments use atomic-rename or symlink-swap to install new files (Kubernetes secret mounts, certbot), and a file-level watch misses or double-fires on those operations; a directory watch reliably sees the `CREATE`/`RENAME` event for the new entry. Events are filtered to the store's name prefix and the cert/key suffix to ignore unrelated files in the working directory, and debounced for 500ms so the multi-file burst of a single rotation coalesces into one rebuild and a half-written pair is not read mid-rotation.

The pure synchronous `ReloadIfChanged` method exists alongside the goroutine-based `Watch` so that the reload *action* stays unit-testable without timing or goroutines. `Watch` is just the trigger; `ReloadIfChanged` is the action, fingerprint-guarded so spurious events are cheap no-ops. The store's internal index is swapped via `atomic.Pointer` so `Get` (called on every handshake) takes no locks.

### `CertStore` lives in `httpx`, not in the ingress

The cert store is a public package primitive rather than internal to `coreservices/httpingress` because any microservice that terminates TLS - not just core ingresses - has the same correctness needs (SAN-driven matching, dynamic rotation, server-wide default). Putting it in `httpx` lets downstream code wire `tls.Config{GetCertificate: store.GetCertificate(port)}` on its own listener without rewriting the loader. The `CertLogger` interface matches the connector's `LogInfo`/`LogWarn` signatures so a microservice's `*Service` satisfies it directly with no adapter.

The store deliberately takes no service-name parameter. Co-located TLS services in the same process share the cert pool by default - SAN and per-port matching disambiguate which cert wins per handshake, so namespace fencing via a file prefix is rarely necessary. Operators who need real isolation between services point each store at a distinct directory.

Caveat: this only helps consumers whose TLS termination accepts a `tls.Config.GetCertificate` callback. `coreservices/smtpingress` uses `go-guerrilla`, which loads certs from file paths internally and does not expose such a hook, so it cannot consume `CertStore` without forking the SMTP daemon. It continues with the file-path model.

