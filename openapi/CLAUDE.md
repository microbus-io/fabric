## Design Rationale

### Two-level schema name scoping for aggregator-safe component keys

Schemas live under `Components.Schemas` keyed by:

- **Service prefix** - `ServiceName` with dots converted to underscores (e.g. `my.service` → `my_service`).
- **Endpoint key** - `<servicePrefix>__<EndpointName>` with a *double* underscore separating the two.
- **Final key** - `<endpointKey>_<role>` where role is `IN`, `OUT`, or a referenced type name.

So `my.service`'s endpoint `Foo` whose input references type `Bar` produces component keys `my_service__Foo_IN` and `my_service__Foo_Bar`.

A doc rendered by an individual microservice stands on its own; the per-service prefix exists for the OpenAPI portal, which aggregates docs from many services into one. Without prefixing, two services that both define a `Bar` type would collide on the component key. The double underscore between hostname and endpoint name is purely visual - it makes the host-vs-endpoint boundary obvious when scanning component keys.

### The error schema is *not* scoped, deliberately

Every Microbus service produces the same `errors.StreamedError` shape under the same component key (`error_ErrorResponse`). The renderer skips the `scopePrefix` for this one schema specifically.

Errors are framework-level: every endpoint of every service emits the same shape. Since collision is impossible - there's only one shape - the schema is keyed at the framework level so that when N services' docs are merged at the OpenAPI portal, the N identical error schemas collapse into a single entry. The portal is the primary beneficiary of this mixed approach (per-service prefixing for endpoints, framework-global keying for the error type).

If you change the error schema, you change it everywhere.

### Greedy path arguments lose the `...` in the rendered path

Microbus routes use `{name...}` for greedy capture (the parameter swallows the rest of the path), but OpenAPI 3.1's path templating syntax has no greedy form - only `{name}`. The renderer strips `...` from both the rendered path and the parameter name. Substitution is best-effort: many OpenAPI clients accept slashes in `{name}` values and route correctly, but the spec doesn't guarantee it. An OpenAPI consumer cannot tell from the document alone that a path argument is greedy. If that distinction matters to a downstream tool, it has to read the original Microbus route, not the rendered OpenAPI path.

### `$id` is stripped from reflected schemas

The `invopop/jsonschema` reflector sets `$id` from the Go package path on every reflected schema. OpenAPI validators reject a schema entry that has both `$id` and `$ref`, which is the wrapper pattern the renderer uses everywhere. `resolveRefs` zeroes out `$id` before the schema lands in `Components.Schemas`. If you upgrade `invopop/jsonschema` and start seeing validation failures, this is the first place to check.

### `$ref` rewriting from `#/$defs/` to `#/components/schemas/<endpoint>_`

`invopop/jsonschema` emits `$defs` for nested type definitions and references them as `#/$defs/TypeName`. OpenAPI organizes shared schemas under `#/components/schemas/` instead. `resolveRefs` walks every schema and rewrites these references in place, simultaneously promoting each `$defs` entry to a `Components.Schemas` entry under the endpoint-scoped key. After this pass, the `Definitions` field is nilled out so the rendered schema doesn't carry stale `$defs`.

### `HTTPRequestBody` / `HTTPResponseBody` shape the schema, not just the wire

The same magic field names that `httpx` recognizes for runtime body routing are also recognized here, but they shape *schema generation*:

- **`HTTPRequestBody`** on the input type causes the request body schema to be reflected from *that field's type alone*. Other fields of the input struct become query or path parameters. The body magic is gated on `methodHasBody(method)` - for `GET`/`DELETE`/etc. the field is silently ignored and no `requestBody` is emitted, but the "other fields become query params" effect still applies (because all input fields are query/path params for body-less methods anyway).
- **`HTTPResponseBody`** on the output type causes the response body schema to be reflected from that field's type alone, preempting all other return values.

Renaming these fields silently changes the schema shape. Same caveat as in `httpx/CLAUDE.md`.

### `deepObject` is query-only per OpenAPI 3.1

OpenAPI 3.1 forbids the `deepObject` style (and `explode: true`) on path parameters - only on query parameters. The renderer sets these only when `parameter.In == "query"`; path params use the default `simple` style. Setting them on path params would produce a doc that fails validation in tools like Spectral or Stoplight.

### Query parameters are not marked `Required`

Query parameters generated from typed function inputs are not flagged `required` even when the underlying Go field is non-pointer/non-omitempty. This is intentional: the framework wants endpoints to accept partial input and apply Go zero-value defaults, with runtime validation in the handler catching wrong values. Marking them required in the OpenAPI doc would cause spec-strict clients to reject valid omissions before the request ever leaves the wire. Path parameters, by contrast, are always `Required: true` because the path itself can't render without them.

### `x-feature-type` and `x-name` are consumed by tooling

Each rendered `Operation` carries two non-standard extensions:

- **`x-feature-type`** - `function`, `workflow`, `web`, or `task`. The producer (`connector/control.go`) emits all
  four; tasks are included so internal inspector tools (e.g. the agent studio on `:428`) can discover task endpoints.
  Outbound events are still filtered out *before* the renderer and never appear. Consumers that expose the doc to
  external callers or LLMs (`openapiportal`, `mcpportal`, `llm.core`) whitelist the feature types they surface and drop
  tasks at their own boundary.
- **`x-name`** - the endpoint's original Go-side name (PascalCase, pre-kebab-conversion).

Three internal consumers read these fields today:

- `llm.core` reads `x-feature-type` to decide whether an operation is a regular tool call (function/web) or a dynamic subgraph dispatch (workflow), and uses `x-name` for tool naming.
- `mcpportal` uses `x-name` to match MCP tool calls back to operations.
- `openapiportal` carries `x-name` and `x-feature-type` through to its aggregated output.

Nothing prevents external callers from depending on these extensions if `mcpportal` or `openapiportal` exposes the doc to them. Treat them as load-bearing.

### Server URL is derived from `RemoteURI` if present

If `Service.RemoteURI` is set, the renderer searches it for `/<serviceName>/` or `/<serviceName>:` and takes everything *before* that match as the `servers[0].url`. This is what makes a rendered doc point at the actual external ingress host (e.g. `https://my.example.com/`) rather than `localhost`. The default is `https://localhost/` when no `RemoteURI` is set.

If `RemoteURI` doesn't contain the service name, the localhost default sticks. The connector's OpenAPI handler pulls `RemoteURI` from the request's `X-Forwarded-*` headers, so a doc fetched without those headers always reads as `localhost`.

### Method defaults differ per feature type

Functions and workflows default to `POST` when method is empty or `ANY`. Web endpoints default to `GET`. The default is applied both to the rendered operation's method *and* to the path-key entry, so the doc has a single `post` entry rather than `any`. If a consumer needs to know the operation accepts any method, they need to rely on the framework's external-but-not-OpenAPI behavior (which the OpenAPI doc cannot express).

### `OutboundEvent` has a constant but never renders

`FeatureOutboundEvent = "outboundevent"` exists in `feature.go` so the value can be set on a `sub.Type` and round-tripped through the framework, but the renderer's switch only handles `function`, `workflow`, `web`, and `task`. Outbound events are filtered upstream by `connector/control.go`'s `handleOpenAPI` before reaching `Render`, so this branch is effectively dead in normal use - the constant is there for completeness and external introspection.

### `Doc` is a transitional alias for `Document`

Pre-v1.27 code used `openapi.Doc`. The type was renamed to `Document` to match OpenAPI's own terminology; `type Doc = Document` is a transitional alias for source compatibility and will be removed in a future release. New code should use `Document`.

### JSON output is byte-stable across renders

Downstream cache mechanisms (notably Anthropic's prompt cache via `claudellm`) depend on the rendered JSON being byte-identical for the same input across multiple calls. Two facts make this hold today:

1. **`Document` is reflected from Go structs.** `encoding/json` marshals struct fields in declaration order, deterministically.
2. **Maps inside `Document`** (`Paths`, `Components.Schemas`, etc.) are marshaled with alphabetically-sorted keys by Go's `encoding/json` (since Go 1.12). So even though Go map iteration order is randomized, JSON output isn't.
3. **`invopop/jsonschema`** emits schema definitions whose property maps marshal alphabetically too, for the same reason.

If you change anything in this package that introduces a non-deterministic ordering - for example, replacing a map with a custom slice traversed in an order derived from `range` over another map without sorting - you can silently destroy cache hit rates in `claudellm` (and any future consumer that depends on byte-stable schemas). Tests would still pass; the failure would only surface as a worsening cache hit ratio in production.

### Per-field descriptions come from struct tags, including for query/path params

Field descriptions flow from the In/Out struct field tags that `invopop/jsonschema` reads when reflecting a whole struct: the dedicated `jsonschema_description:"..."` tag (preferred, read whole) or a `description=` directive in the comma-split `jsonschema:"..."` tag. For a request body or a nested custom type the whole struct is reflected, so these tags land on the schema properties for free.

A scalar **query or path** parameter is the one case that needs help: the renderer reflects each such field from its *type alone* (`jsonschemaReflectFromType(field.Type)`), which drops the field-level tag. `fieldTagDescription` reads the tag directly off the `reflect.StructField` and sets `Parameter.Description`, mirroring invopop's precedence (`jsonschema_description` first, then `description=` in the `jsonschema` tag). This keeps query/path params describable by the same tags as body fields.

Per-argument descriptions live only on the fields, one mechanism. Prefer `jsonschema_description` over the `jsonschema:"description=..."` subtag because the latter is comma-split (a description with a comma is silently truncated at the first comma); reserve the `jsonschema` tag for directives like `example=`. The legacy `Input:`/`Output:` godoc-section convention that once filled property descriptions from bulleted godoc lines is **retired** - do not reintroduce it.

A **magic HTTP body** argument (`httpRequestBody`/`httpResponseBody`) can carry a body-level description on the field itself: a `jsonschema_description` (or `description=` directive) tag on the `HTTPRequestBody`/`HTTPResponseBody` field is read by `fieldTagDescription` and set on the `requestBody.description` / response node's `description` (replacing the `"OK"` default). This is needed because the body schema is reflected from the field's *type alone* (`jsonschemaReflectFromType(field.Type)`), which drops the enclosing-struct field tag - the same reason scalar query/path params read the tag directly. The description lands on the `RequestBody`/`Response` node rather than beside the schema `$ref` (a sibling `description` next to a `$ref` is illegal in OpenAPI 3.0), so it is version-safe. The referenced struct's own *fields* are still described by their own `jsonschema_description` tags (as above), reflected as a whole. The struct's godoc does **not** reach the document: the reflector runs without `AddGoComments`, so no Go comments are read at runtime. A magic body over a non-struct type (e.g. `[]string`) has no fields to tag, so the field-level description is the only way to document it - which is the case this most helps.

Per-argument descriptions are optional in general: the endpoint's own description (the feature var's godoc, surfaced as the operation description) can state what it takes and returns, so most endpoints need no per-argument tags at all.

### dv8 directives are projected onto the schema

The `dv8` validation tags that the framework enforces at the trust boundary are also projected onto the rendered schemas (`dv8schema.go`), so consumers - human docs and LLM tool-calling above all - see the real contract instead of discovering it via 400s: `len<=64` becomes `maxLength`, `val>=0` becomes `minimum`, `oneof` becomes `enum`, `regexp` becomes `pattern`, `notzero` adds the field to the parent's `required` (plus a type-appropriate minimum: `minLength`/`minItems`/`minProperties` of 1, `const: true` for booleans), `default=` becomes `default`, and the `each`/`key` prefixes descend into `items`/`additionalProperties`/`propertyNames`.

`invopop/jsonschema` has no hook for custom struct tags, so the projection is a post-reflection pass: `applyDV8Type`, called from the two reflect helpers, walks the Go type in parallel with the reflected schema (resolving `$defs` references) and patches JSON Schema keywords in. The tag grammar itself is parsed by `dv8.ParseTag`, which dv8 exports precisely so the grammar (comma splitting with `\,` escapes, `each`/`key` prefix chains, operator forms) stays single-sourced in the repo that owns it rather than duplicated here where it would silently drift.

The projection is best-effort and deliberately *looser or equal* to what the runtime enforces - it must never cause a spec-strict client to reject a request the server would accept:

- Mutating directives (`trim`, `tolower`, `toupper`) and structural ones (`delegate`, `on`, `-`) have no constraint analog and are skipped.
- A `val` bound whose value is not a JSON number (durations like `24h`, timestamps) is skipped; numeric bounds on a `time.Duration` field would otherwise be correct only by accident (the wire format is nanoseconds).
- Unrecognized or malformed directives are skipped, never an error - validity is `dv8.Compile`'s job at microservice startup; rendering never fails on a bad tag.
- Query parameters get the constraints on their schema but are still not marked `required` (see above), even for `notzero`.

Two sharing rules keep constraints from leaking across uses of a type. A named type's `$defs` entry is shared by every field of that type, so it is mutated only by the type's *own* field tags (identical for all uses) and only once per reflection. Conversely, a *field-level* prefixed directive (`each len<=10` on a field of a named container type) is skipped when descent hits a `$ref` - projecting it onto the shared definition would impose one field's constraint on every other use.

Field-level tags on scalar query/path parameters and on magic `HTTPRequestBody` fields are applied directly off the `reflect.StructField` (`applyDV8FieldTag`) because those schemas are reflected from the field's type alone - the same reason `fieldTagDescription` exists. The response side (`HTTPResponseBody` field tags) is deliberately not projected: dv8 tags are inert on outputs. Domain types shared between In and Out structs still carry their constraints in OUT schemas, since type-level reflection is uniform; that is descriptive and harmless.

Determinism is preserved: every projected keyword derives from struct tags visited in field declaration order (`required` entries append in that order), so the byte-stability guarantee above still holds.
