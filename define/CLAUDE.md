# define

The typed vocabulary a microservice's api package `definition.go` uses to declare every feature it exposes.
`cmd/genservice` reads these `define.*` literals statically (via AST) and generates `intermediate.go`,
`client.go`, `mock.go`, and `manifest.yaml` from them. `definition.go` is the single source of truth; the
generated files never drift because they are projections of it. The broader migration plan is in the
repo-root `_DEF.md`.

## Design Rationale

### Literals only

The generator resolves field values statically, without running the code, so a `definition.go` may use only
constant literals, struct literals as In/Out/Value type carriers, and references to other `define.*` vars
(including cross-package). No concatenation, function calls, or conditionals. The generator errors on a
non-resolvable field. If expressions ever become genuinely necessary, that is a deliberate future change, not
something to slip in.

### In, Out, and Value are type carriers, not runtime values

`In`/`Out`/`Value` exist solely so the generator can read a Go type out of the source; they are never used at
runtime. `In: FooIn{}` carries the type `FooIn` via a composite literal; `Value: int(0)` carries `int` via an
explicit conversion. The explicit conversion matters: it keeps the type readable from pure AST (a `CallExpr`
whose `Fun` is the type identifier) without needing `go/types` to resolve untyped-constant defaults, which is
why `int(0)` is required over a bare `0`. The generator rejects a `Metric.Value` whose carrier is non-numeric.

This is also why the types are not generic. Generic type parameters would add type machinery for zero runtime
benefit (the params would be phantom), force an explicit `[IN, OUT]` on every literal (Go has no
composite-literal inference), and break the clean `InboundEvent{Source: ...}` cross-package reference. Value
carriers keep every feature a flat, labeled struct literal.

### Routing fields are flattened, not embedded

`Host, Method, Route, RequiredClaims, TimeBudget, LoadBalancing` are repeated across the five reachable feature
kinds rather than shared via an embedded base. Go composite literals cannot set *promoted* fields of an
embedded type, and an unexported base cannot be named cross-package, so embedding would force
`define.Function{Endpoint: define.Endpoint{...}}` at every call site. Flattening keeps the repetition here,
once, so each `definition.go` reads as a flat literal. `URL()` is therefore a one-line method on each of the
five routable kinds.

### Host and URL()

Every routable literal sets `Host: Hostname` (the api package's own `Hostname` const). `URL()` joins `Host`
and `Route` and is what callers use to name an endpoint for LLM tools and graph/task references (e.g.
`calculatorapi.Arithmetic.URL()`). The per-literal `Host: Hostname` repetition is the cost of keeping `URL()`
on the shared types without a runtime hostname registry; the migration generator writes it.

`joinHostAndPath` is a verbatim copy of `httpx.JoinHostAndPath` so that `define` depends only on the standard
library, never on a framework package. `define_test.go` pins the copy to `httpx.JoinHostAndPath` to catch
drift.

### InboundEvent references the source's OutboundEvent

`InboundEvent.Source` is typed `OutboundEvent` (not `any`), so a removed or renamed event in the source
microservice is a compile error in every consumer. The Service method that handles the event is named after
the `InboundEvent` var, consistent with how every other feature's handler name derives from its var name; there
is no separate `Handler` field. If a consumer sinks two same-named events from different sources, the
package-level vars must already have distinct names, which yields distinct handler methods for free.

`InboundEvent` also carries `RequiredClaims`, `TimeBudget`, and `LoadBalancing` - the consumer-side subscription
options the previous hand-written `NewHook(svc).WithOptions(...)` binding supported. It has no `Host`/`Method`/
`Route` of its own (those come from `Source`), so it is not a routable kind with a `URL()`; these three fields
are the only consumer-tunable knobs. genservice renders them into the generated hook wiring as
`NewHook(svc).WithOptions(sub.RequiredClaims(...), sub.TimeBudget(...), sub.NoQueue()/Queue(...)).OnX(svc.OnX)`,
omitting `WithOptions` entirely when none are set. `LoadBalancing: define.None` is the load-bearing one: it makes
every replica process the event (the cache-invalidation pattern), versus the default queue where one replica
handles each delivery.

### Descriptions live in godoc, not a field

There is no `Description` field on any type. The endpoint/feature description is the godoc on the var, which
keeps each literal purely structural and makes `go doc` of the api package a readable contract. Per-field
descriptions live in `jsonschema_description:"..."` tags on the In/Out struct fields.

### Config: explicit Value, verbatim Validation, string Default

`Config.Value` is an explicit type carrier rather than a type derived from `Validation`. The deciding factor is
that a config value may be an arbitrary JSON-unmarshalable struct, which the cfg validation grammar cannot name
(`json` only means "valid JSON", not "a `MyStruct`").

`Validation` stays the verbatim cfg expression on the raw string. It is kept whole rather than split into a
constraint-only form because cfg's string validators (`str`, `url`, `email`, `json`, `set`) are validator
selectors, not "str + constraint", so they do not decompose cleanly. The generator cross-checks the leading
validator keyword against `Value`'s type.

`Default` stays a raw string because the source of truth for a config value is always a string (from YAML or
env); the default flows through the same validate-then-convert path as a supplied value. A `Default any` would
have to be normalized back to a string anyway.

A struct-valued config (`Value: MyStruct{}`, `Validation: "json"`) is fully runtime-supported: the configurator
canonicalizes a nested YAML value to a JSON string, the generated getter json-unmarshals it, and the generated
`cfg.Validator` closure rejects a value that does not unmarshal into the carrier type or that fails
`dv8.Validate` (struct tags plus a custom `Validate` method).

### Metric carries Value and Labels

The recorder method's value type (`int` vs `float64`, which varies independently of kind) and its label
parameters live only in the hand-written recorder method in today's code. To generate that method they must be
declared, hence `Value` (the numeric type carrier) and `Labels` (the recorder's additional parameter names).
Labels are assumed string-typed, which holds for every current service; revisit only if a non-string label
appears.
