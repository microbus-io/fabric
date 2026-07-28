# cmd/genservice

Generates a microservice's boilerplate from a single typed source of truth: the `*api/definition.go` file. From
that one file genservice emits five artifacts:

- `*api/client.go` - the client proxies (`Client`, `MulticastClient`, `Executor`, `Subgraph`, `MulticastTrigger`,
  `Hook`) and their marshaling helpers.
- `intermediate.go` - the `ToDo` interface, `NewService`/`Init`/`NewIntermediate`, `Subscribe` wiring, `doXxx`
  marshaling, config/metric/ticker registration, and inbound hook wiring.
- `mock.go` + `mock_test.go` - the mockable `Mock` and a structural smoke test.
- `manifest.yaml` - the derived navigational view of what the microservice exposes.

Beyond those five fully-generated artifacts, genservice edits two hand-written files in place. In
`service.go` it syncs the godoc of each existing `*Service` handler to its feature's description and
scaffolds a placeholder handler for any feature that has none (see Syncing handler godoc and Scaffolding
handlers below). In `service_test.go` it scaffolds a placeholder test for any feature that has none (see
Scaffolding feature tests below). All three edits are append-or-sync only: a hand-written body is never
rewritten.

genservice is the sole generator of a microservice's boilerplate. The housekeeping skill runs it; every
microservice authors a `definition.go` and genservice produces everything else.

## The pipeline: parse once, project many times

`extract.go` parses every non-test `.go` file in the api package into one `service` model: the api package name,
the `Hostname`/`Name`/`Version`/`Description` consts, the ordered list of `feature`s (one per `define.*` var), the
In/Out and domain `struct`s, and the file's import aliases. Every emitter is a pure projection of that one model,
so the five outputs cannot drift from each other - they are computed from the same tree in a single pass.

A `feature` keeps the var's raw keyed fields as `map[string]ast.Expr` (`attrs`) plus the godoc and the resolved
In/Out type names. Keeping the raw AST expressions lets each emitter render exactly what it needs (a duration as
`5s` for the manifest but as `5 * time.Second` for a `sub.TimeBudget` call) without the extractor having to
anticipate every consumer.

### Literals only

Field values in `definition.go` must be statically resolvable by AST walking: constant literals, struct composite
literals used as type carriers (`In: GreetIn{}`), `define.*` constants (`define.None`), and cross-package
references (`Source: srcapi.OnRegistered`). There is no `go/types` pass and no evaluation. This is the price of
keeping `definition.go` a declarative spec rather than code with logic, and it is why the `define` package models
config and metric value types as explicit type carriers (`Value: int(0)`) instead of inferring them.

The string-literal fields `Method`, `Route`, and `RequiredClaims` are the sharp edge of this rule: a value like
`RequiredClaims: myGateConst` walks to an `*ast.Ident`, which `attrString` returns as `""` - so before the guard
was added, factoring a claims expression into a shared const silently produced an *ungated* endpoint with no
error. `parseService` now rejects a non-string-literal in any of those three fields up front, naming the feature
and field (the same shape as the backtick checks alongside it). `Host` is deliberately excluded: it is idiomatically
`Host: Hostname` (an `*ast.Ident`), and genservice ignores the per-feature `Host` attr entirely, using the
service's `Hostname` const instead. `LoadBalancing` is also excluded because `define.None`/`define.Default` are
valid `*ast.SelectorExpr` values resolved separately.

## Verbatim-embedded strings must not contain a backtick

Two author-supplied strings are projected into the generated `intermediate.go` inside a Go raw string literal: a
feature's godoc (`sub.Description(`...`)`, `cfg.Description(`...`)`, the metric describe calls) and a config's
`Default` value (`cfg.DefaultValue(`...`)`). A backtick in either closes the raw string early and yields
uncompilable Go, which surfaces only as an opaque `format.Source` parse error far from the cause. `parseService`
therefore rejects both up front with a message naming the offending feature. A config default can smuggle in a
backtick because it may be written as a double-quoted literal (``Default: "a`b"``), which `attrString` unquotes to a
value the emitter then drops into the raw string. The godoc case cannot itself hold a backtick in a raw-string
comment, but a `//`-line godoc can, so both are checked.

## Why text/template for the Go files but a hand emitter for YAML

`client.go`, `intermediate.go`, `mock.go`, and `mock_test.go` are rendered from `//go:embed`-ed `.txt` templates
(`templates/`), then run through `format.Source` (gofmt). Go is gofmt-normalized afterward, so the templates only
have to be *valid* Go, not *pretty* Go: alignment, import grouping, and spacing are all fixed by gofmt. Templates
keep the code skeleton readable as code rather than buried in string concatenation.

`manifest.yaml` is emitted by hand in `manifest.go` instead. YAML has no gofmt to lean on: quoting, key order, and
block-scalar formatting all have to be exactly right in the emitted bytes. A custom emitter with deterministic key
order per section is more predictable than `yaml.v3`, which quotes inconsistently and reorders keys.

Do not embed Go (or YAML) source as string constants inside the `.go` emitters. Code skeletons live in the `.txt`
templates; the emitters build the data model and compute the small fragments (signatures, import sets) that the
templates interpolate.

## Type qualification: the `apiPkg` opt-in

In/Out struct fields are declared in the api package. `client.go` lives in that same package, so it refers to a
domain type by its bare name (`Pet`). But `intermediate.go` and `mock.go` live in the *service* package one level
up, where the same type must be written `svcapi.Pet`.

`featureView` carries an `apiPkg` field that is empty for the client (bare types) and set for the intermediate and
mock (qualified types). `Params()`/`Returns()` run field types through `qualifyTypes`, which prefixes a bare,
non-builtin identifier with `apiPkg` while leaving already-qualified selectors (`time.Time`,
`*errors.TracedError`) and builtins alone. Inbound-event views qualify with the *source* package alias instead,
since an inbound handler's parameters are the source event's types.

Because qualified field types pull in packages the api struct imported (`time`, another api package),
`featureSelectorImports(svc, kinds)` walks the In/Out fields of the features whose kind is in `kinds` and resolves
each `pkg.Type` selector against the api package's own import aliases, feeding the computed import set. The `kinds`
filter exists because the emitters reference different feature subsets, and an import that no emitted code uses is a
compile error in a file the agent may not hand-edit:

- `client.go` renders the In/Out of *every* feature, so it passes `nil` (all kinds). An outbound event's field
  types reach `client.go` through its `Trigger`, and an inbound event's through its `Hook`.
- `intermediate.go` generates handlers only for functions, web handlers, tasks, and workflows, so it passes exactly
  that set. An outbound event has no handler in `intermediate.go` (it is fired from the `Trigger` in `client.go`),
  so pulling its field types' packages into `intermediate.go`'s imports would leave an unused import - e.g. an
  outbound event with a `time.Time` field would import `time` into an `intermediate.go` that never references it.

Inbound events are a third case, handled outside `featureSelectorImports`: an inbound handler's parameters are the
*source* event's types, declared in the source api package, not this one. `inboundView` resolves those field
selectors against the *source* package's aliases and returns both the source api package's import path (for the
handler's bare domain types, which qualify to `srcPkg.Type`) and the packages of any `pkg.Type` selectors in the
source fields (`time.Time`, another api package's type). Without the latter, an inbound handler taking a
`time.Time` from its source event would reference `time` in `intermediate.go` with no import. `mock.go` reaches the
same result by a different route: `mockAliases` merges the source package's aliases into its resolution set, so its
generic `addResolved` over the rendered handler signatures already covers the source field types.

In/Out fields are not the only types `intermediate.go` renders. A config getter (`RefreshInterval() time.Duration`),
a metric recorder (`RecordLatency(ctx, value time.Duration)`), a ticker interval (`30 * time.Second`), and a
`sub.TimeBudget(5 * time.Second)` option all emit Go fragments that may reference a non-stdlib-default package. So
`emitIntermediate` runs each of those rendered fragments through `addResolved` against the api package's aliases,
exactly as In/Out fields are resolved, rather than hard-coding which kinds happen to need `time`. The hazard this
closes: a `Value: time.Duration(0)` metric renders `time.Duration` into the recorder, and without resolving the
metric value type's package the generated `intermediate.go` referenced `time` with no import - leaving a file the
agent is forbidden to hand-edit uncompilable. This is intermediate-only: `client.go` has no config/metric/ticker
surface, and a metric's `OnObserve`/`OnChanged` callbacks (the only metric/config methods on `mock.go`) are
`(ctx) error` and carry no value type.

## Struct-valued configs

A `define.Config` value carrier is usually a scalar (`string("")`, `int(0)`, `time.Duration(0)`), but it may be an
api-package struct (`Value: RetryPolicy{}`) for a configuration knob whose shape is structured. The config value is
always stored as a string in the connector, so a struct config is just `cfg.Validation("json")` over a JSON-text
value; the typed accessor does the marshaling. `configView` carries both the raw `Type` (a scalar name or the bare
struct name - used by the scalar getter switch and the manifest signature, which documents the api-level contract
unqualified) and `GoType`, the same type qualified for the service package via `qualifyTypes` (`RetryPolicy` ->
`svcapi.RetryPolicy`, scalars unchanged). The generated getter (`func (svc *Intermediate) Retry() (value
svcapi.RetryPolicy)`) reads `svc.Config`, `json.Unmarshal`s it (swallowing the error to a zero value the same way
the scalar getters swallow parse errors), and runs `dv8.Validate` over the result to apply normalizing directives
(`default=`, `trim`); the setter `json.Marshal`s and `errors.Trace`s a marshal failure. The registration also wires
a `cfg.Validator` closure that unmarshals the raw string into the carrier type and runs `dv8.Validate`, so the
connector rejects a value that is not a `RetryPolicy` (or that fails its dv8 tags or custom `Validate` method)
before it is committed - the getter can therefore keep swallowing errors, because an invalid value never reaches
it. A struct config's declared `Validation` may only be `json` (or empty); emitIntermediate rejects anything else.
The struct type must live in the api package (it is named from `definition.go`, which is in the api package, so it
cannot live in the service package without an import cycle) - which is exactly what lets a test in the service
package name `svcapi.RetryPolicy{...}` and call the typed setter. `encoding/json` + `errors` + `dv8` are added to
`intermediate.go`'s imports only when a non-scalar config is present. `mock.go` is unaffected (a config surfaces
there only as its `OnChanged` callback, which carries no value type).

## Input validation via dv8

The `marshalFunction` helpers (one in `intermediate.go` for functions, one in `client.go` for inbound-event
hooks) and the task `doXxx` handlers run `dv8.Validate` over the decoded In struct before invoking the handler,
so dv8 tags and `Validate` methods on api types are enforced at the trust boundary. dv8 stamps the HTTP status
on its own errors (400 for invalid input, 500 for a malformed directive - a bug in the tags, not the input), so
the generated code propagates with a plain `errors.Trace` and never branches on the error kind. Tasks are included because a workflow's state is a cross-service contract:
its fields are populated by the workflow's caller, by LLMs, and by tasks hosted in other microservices, so a
task's declared inputs are a boundary, and a violated contract fails the flow at the offending step rather than
propagating garbage. Mocks inherit the validation because `Mock` routes through `NewIntermediate`. Out types are
deliberately not validated: dv8 mutates (`default=`, `trim`), and outputs are produced by trusted code.

`NewIntermediate` also wires a `Connector.Init` that calls `dv8.Compile` over every function In type, task In
type, and inbound event's source In type (one `// MARKER: <Feature>` per line), so a malformed directive fails
the microservice at startup rather than mid-request. Web handlers are raw and have no In struct.

## The request body is released before the handler runs

The same three generated marshalers set `r.Body = http.NoBody` immediately after decoding, before dispatching to
the handler. By that point the payload has been fully decoded into the In struct (or the `workflow.Flow`), so the
inbound buffer is dead weight for the whole duration of the handler - which for an LLM call, a database query, or
a slow downstream is far longer than the decode. Holding it makes a microservice's resident memory a function of
how long its own handlers run, multiplied by concurrent requests.

The connector cannot do this itself. It hands the handler an `*http.Request` and has no idea when, or whether, the
body has been consumed: a web handler may read it at any point, or stream it, so the framework must leave it
intact. Only the generated marshaler knows the body has been fully decoded and will never be read again, which is
why the release lives here and not in `subscribe.go`. Web handlers are raw and are deliberately not touched.

Measured over both transports with 64 concurrent 512KB requests parked in the handler, receiver-side retention
falls from ~100% of the bodies in flight to ~1%. The bytes really are freed rather than merely unreferenced by one
alias: `onRequest`'s handler goroutine captures the `transport.Msg`, but Go's precise liveness drops it once
`handleRequest` stops using it, and `httpReq.WithContext` has already shallow-copied the request by the time the
handler sees it, so the copy's `Body` field is the last root.

`http.NoBody` rather than `nil` because a nil `Body` panics anything that later reads or closes it, which is
exactly the failure the connector's own release path hit on the short-circuit transport.

## Feature-selective emission, conditional imports, no var guards

The client emits only the proxy types a microservice actually needs: no `MulticastTrigger` without outbound
events, no `Executor` without tasks or workflows, no `Subgraph` without workflows (it carries only workflow
methods - a bare task is not independently invocable), and only the `marshalXxx` helpers the emitted methods
call. Imports are computed in Go from the feature mix (see `buildClientModel`) and emitted conditionally, so every
import is referenced by real code. There is therefore no `var ( _ = pkg.Symbol )` guard block: such guards are
only needed when the templates import a fixed superset, which this generator does not.

genservice deliberately does not depend on `goimports` to fix up imports. `goimports` is not part of the standard
Go toolchain, so it cannot be assumed present on every machine that builds the project. Imports are computed
explicitly instead.

## Header preservation, never synthesis

No emitter writes a copyright header. Each one preserves the leading comment block of the file it is regenerating
(`existingHeader` for the Go files, `manifestHeaderOf` for the manifest), stripping any prior
`Code generated ... DO NOT EDIT` marker so it is re-emitted exactly once. A fresh file gets no header; an operator
who wants one adds it once and the generator keeps it on every subsequent run. This keeps genservice out of the
business of choosing or dating a license.

## Service-level consts live in definition.go

`definition.go` declares four consts that define the microservice's identity: `Hostname`, `Name` (the decorative
PascalCase name), `Version` (the API major version), and `Description`. The generated `intermediate.go` references
all four symmetrically:

```go
const (
    Hostname    = svcapi.Hostname
    Version     = svcapi.Version
    Description = svcapi.Description
)
```

`Description` is a `const` value rather than a doc comment for two reasons. First, a downstream project may run a
linter that forces every godoc to begin with the symbol it documents (a package doc must start `Package xxx`, a
`Hostname` doc must start `Hostname`), which makes prose-in-a-comment unusable as a clean, free-form description. A
plain string value sidesteps the convention entirely and supports multi-line text via a backtick raw string.
Second, the symmetric reference block above only compiles as const-init-from-const: `const Description =
svcapi.Description` requires `Description` to itself be a const, not a var.

`Name` cannot be derived: the package directory is lowercase (`creditflow`) and recovering the operator's intended
capitalization (`CreditFlow`) without a dictionary is lossy, so it is authored once in `definition.go`. genservice
errors if a service-directory generation finds no `Name`, `Version`, or `Description` const, rather than emitting a
file that references a missing symbol.

## The ToDo interface is generated and load-bearing

`NewIntermediate(impl ToDo)` takes an interface, not a concrete type, so the same constructor accepts both
`*Service` (production) and `*Mock` (tests). `ToDo` also doubles as the compile-time proof that `*Service` and
`*Mock` implement every handler. It is generated from the feature set (one method per function, web, task,
workflow, inbound event, ticker, observable metric, and config callback) and must stay an interface for that
polymorphism to hold.

## Manifest specifics

`manifest.go` builds the `general` block as `{name, hostname, description, package, modifiedAt}`. The decisions
behind that set:

- `name`/`hostname`/`description` come from the `definition.go` consts; `package` is computed from `go.mod` +
  directory via `importPathOf`.
- `frameworkVersion` was dropped: no Go code reads it and it is not navigational.
- `db`/`cloud` were dropped: they never appear in a committed manifest, and `cmd/gentopology` derives them itself
  by scanning service source code.
- `modifiedAt` is the only stateful field. `emitManifest` renders once with the prior timestamp, and if the bytes
  match the existing file it keeps that timestamp (content unchanged); only when other content actually changed
  does it re-render with the new timestamp. This render-twice dance is what makes regeneration idempotent and
  `-check` stable, since otherwise the timestamp would advance on every run.

Signatures use each field's Go name lower-cased on the first letter (`lowerFirst(goName)`) uniformly across
functions, tasks, workflows, inbound and outbound events. This preserves an `Out` suffix (`countOut`), which a
JSON tag would collapse (`count`) into an invalid or colliding Go signature, and it matches the param names of the
generated client stubs. Config signatures are `Name() (value Type)`, matching the generated getter (whose named
return is `value`); ticker signatures are `Name()`; metric signatures are `Name(value Type, label string, ...)`.
Types are left as declared in the api package (no qualification), because the manifest documents the api-level
contract.

## The workflow-graph mock variant

A workflow's `Mock` cannot simply call the user's handler the way a function mock does, because callers do not
invoke the graph directly: the Foreman runtime executes its tasks by posting state to subscribed task URLs.
`MockMyWorkflow(handler)` therefore subscribes a synthetic task on `:428/mock-<kebab>-<rand>` that decodes the
incoming flow, parses the workflow's In struct, calls the typed handler, writes the Out struct back, and replaces
the graph with a single transition to that task.

## Mode detection

`emitAll` decides what to generate from the directory shape. A directory with an `<x>api` subdirectory containing
`definition.go` is a service directory, and all five artifacts are produced. A directory that itself contains
`definition.go` is an api package; if it belongs to a microservice (`owningServiceDir`: the parent is `<x>` to the
api's `<x>api` and has a sibling `service.go`) it is **promoted** to full service-directory generation from the
parent, so pointing genservice at the api dir produces all five artifacts and the other four cannot silently drift
behind a `client.go`-only regeneration. Only a standalone api/contract package with no owning `service.go` (the
`pressuretest/*` fixtures) falls back to `client.go`-only. Import paths (api package, resources, service package)
are computed by walking up to the nearest `go.mod` (`findModule`) and doing string math against the module root
(`importPathOf`, `inModuleDir`).

## Cross-package inbound events

An `InboundEvent.Source` is a typed reference to another api package's `OutboundEvent` var. To generate the hook
wiring and the handler signature, genservice must read that source package's In/Out structs. `resolveSource` maps
the source import path to its on-disk directory (string math against the module root, no `go list`) and parses it
into another `service` model. This only works for sources inside the same module, which holds for every
intra-project event.

## -check and golden tests

`-check` regenerates in memory and compares against the files on disk: it writes nothing and exits 2 if any output
is stale (a CI guard), 1 on error, 0 when current. It shares the exact `emitAll` collection path with normal
generation, so what `-check` validates is what a write would produce.

`genservice_test.go` runs the generators against the `testdata` fixtures in place and asserts byte equality
against the committed output (`TestGoldens`, refreshable with `-update`); `TestManifestModifiedAtStable` pins the
timestamp-stability property. The goldens are compared in place rather than in a temp copy, because service-
directory generation needs `findModule`/`go.mod` resolution, which fails for a directory copied outside the module
tree. The committed fixture files therefore *are* the goldens.

## Fixtures

- `testdata/svc` is a full service exercising every feature kind: functions (with magic HTTP args and the `xxxOut`
  task suffix), a web handler, tasks, a workflow, a cross-package inbound event (sourced from
  `pressuretest/srcapi`), an outbound event (`OnPeerSeen`), counter/gauge(observable)/histogram metrics,
  secret/callback/duration configs, a ticker, plus a domain type (`Pet`) and an external type (`time.Time`) to
  exercise qualification across the service-package files. Two fields pin the import-direction rules for events,
  each typed from a package no other feature imports: the inbound source event carries a `net/url` field, and so
  `net/url` must appear in `intermediate.go` (the handler signature) but the golden regresses if an inbound event's
  source-package field types stop reaching `intermediate.go`; the outbound event carries a `net/netip` field, and
  so `net/netip` must appear in `client.go` (the `Trigger`) but *not* in `intermediate.go`, the golden regressing
  if an outbound event's field types start leaking into the handler file that has no handler for it.
- `testdata/pressuretest/{srcapi,svcapi}` are api-only packages (client.go generation) covering every client type
  and helper, magic HTTP args, the `xxxOut` suffix, and a cross-package type alias.
- `testdata/configonly` is an endpoint-less service (only configs and one metric). Beyond pinning the no-Subscribe
  import mix, its `time.Duration`-valued metric is the *sole* `time` user, so the golden regresses if a metric
  value type's package stops being resolved into `intermediate.go`'s imports. `svc` cannot pin this because its
  duration config and ticker import `time` regardless.

## Syncing handler godoc

`definition.go` is the single source of a feature's description, but that description is a human's first stop when
reading the handler in `service.go`, not the api package. So `emitServiceDocs` keeps the two in lockstep: after the
five artifacts are built, it rewrites the godoc of every `*Service` method named after a feature to match that
feature's description, as a `/* */` block comment. This is the one place genservice edits hand-written code, so it
is deliberately narrow.

- **Which methods.** The kinds whose handler is a `*Service` method named exactly after the feature -
  functions, web handlers, tasks, workflows, inbound events, and tickers (`serviceDocKinds`) - take the feature's
  description verbatim. An observable metric's `OnObserveXxx` and a callback config's `OnChangedXxx` are also
  hand-written on `*Service`, but their names carry a fixed prefix, so a verbatim description would not open with
  the method name (the godoc convention, which some downstream linters enforce). For those, `handlerDocs`
  synthesizes a fixed first line naming the method (`OnObserveXxx emits the observed value of the Xxx metric.`,
  `OnChangedXxx is called when the Xxx config property changes.`) and appends the feature's description as a second
  paragraph. The generated surface (recorders, getters) that backs those callbacks lives on the intermediate and is
  untouched. A method whose name matches no feature (a private helper, `OnStartup`, an `OnObserve`/`OnChanged` for a
  non-observable/non-callback feature) is left alone, as is a same-named method on a different receiver.
- **Byte splice, not reprint.** The edit is a positional splice against the original file text (locate the existing
  `Doc` comment group by AST offset and replace it, or insert before the `func` keyword when there is none), not a
  `go/printer` round-trip. Reprinting the whole file would reformat unrelated hand-written code and reorder or drop
  comments; splicing touches only the located doc regions. The result is re-parsed to guarantee valid Go before it
  is written.
- **`/* */` style.** Descriptions are multi-line prose lifted verbatim from `definition.go`; a block comment
  carries them without a per-line `//` prefix that gofmt-alignment and hand-editing would fight over. A pre-existing
  `//` doc is replaced by the block form, so re-running is idempotent (`TestGoldens` and `-check` stay stable).
- **Only changed files are emitted.** `emitServiceDocs` returns an `output` only for a file whose bytes actually
  change, so an already-synced directory produces nothing and `-check` does not flag it. Generated files (the
  `Code generated ... DO NOT EDIT` marker) and `_test.go` files are skipped.

## Scaffolding handlers

Every feature that has a `*Service` method (functions, web handlers, tasks, workflows, inbound events, tickers, and
the `OnObserve`/`OnChanged` callbacks of observable metrics and callback configs - an outbound event has none)
starts as a compiling stub, so the agent fills in a body rather than authoring a signature by hand. `emitServiceCode`
combines this with the godoc sync, because both edit `service.go` and two separate outputs for one file would
clobber each other: it runs `emitServiceDocs`, then appends stubs on top of the already-synced bytes, and returns a
single `service.go` output.

- **Signature and godoc are projected from definition.go, so they cannot drift.** `buildHandlerStub` reuses the same
  field qualification as the mock (`standardMock`, so a domain type is written `svcapi.T` in the service package),
  and the godoc is `handlerDocs`' entry for that handler - the identical value the sync path would enforce. Web and
  workflow signatures are fixed shapes (`w, r` / the graph builder). Imports the signature needs (`context`,
  `net/http`, the `workflow` package for a task's `flow` or a workflow's graph, `time`, the api or source package
  for a domain field) are resolved via `mockAliases` + `addResolved` and merged into `service.go`'s import block by
  the shared `appendGoDecls`.
- **The body is a TODO plus a naked return; a workflow gets a `NewGraph` starter** that returns an empty graph,
  which fails `graph.Validate()` at run time until the agent defines it - a loud "not implemented" for the one kind
  where a silent zero return would look valid.
- **Append-only, keyed by method existence.** `existingServiceMethods` scans every hand-written `.go` file for
  `*Service` methods; a feature whose handler is defined anywhere is skipped, so a fully implemented microservice
  regenerates to no change and `-check` stays stable. A feature's handler is stubbed exactly once, then owned by the
  agent.

## Scaffolding feature tests

Every feature kind has a recommended integration-test shape (the `add-*` skills describe it). `emitServiceTests`
generates that placeholder into `service_test.go` so a feature always starts with a compiling test stub, rather than
depending on a human to paste the pattern. This is the second in-place edit of hand-written code, and like the
godoc sync it is deliberately narrow: it only ever *appends* what is missing.

- **Append-only, keyed by function name.** A feature is considered covered once a function of its test's name
  (`Test<Name>_<Suffix>`) exists in any of the microservice's `_test.go` files (`existingTestFuncs`), the exact
  parallel of `existingServiceMethods` on the handler side - both delegate to one `scanFuncDecls` that walks every
  hand-written file, so a handler or test split across several files is still found and never duplicated.
  `emitServiceTests` renders a scaffold only for features whose test function is absent (appending it to
  `service_test.go`), so a hand-filled test is never rewritten and a re-run is a no-op. It
  returns an `output` only when it added something, so `-check` does not flag an already-covered directory. The
  scaffold still carries a `// MARKER: <Feature>` comment, but for feature-code navigation (`grep "MARKER: X"`), not
  for coverage detection. The trade-off versus keying on the marker: renaming a generated test away from the
  convention yields one redundant empty scaffold on the next run (a different name, so never a duplicate-symbol
  error), which the handler side cannot suffer because a method name is fixed by the `ToDo` interface.
- **Which kinds.** Functions, web handlers, tasks, workflows, inbound and outbound events, and tickers always get a
  scaffold; a config only when it is a `Callback` and a metric only when it is `Observable` - matching the kinds
  that have a `*Service` method to exercise. The test function name is `Test<Name>_<Suffix>` using the decorative
  `Name` const (so it reads as a human following the skill would author), with the `OnChanged`/`OnObserve` prefix
  for callback configs and observable metrics; the marker is always the bare feature name.
- **The scaffold compiles; the assertions are commented.** Each scaffold emits the setup a human would write (the
  tester/client/executor for that kind, `app.Add`, `RunInTest`) and a `/* */` HINT block holding the `t.Run`
  pattern as pseudo-code. Only the setup is live Go, so the stub compiles and passes immediately; the per-kind
  templates therefore add `_ = client` / `_ = exec` style blank references for identifiers used only inside the
  HINT (a few skill scaffolds omit these and would not compile as-is - the generated form always includes them).
  Imports are computed from the compiling part only (e.g. a workflow scaffold pulls in `foreman`/`foremanapi`,
  everything else stays in the comment).
- **Create vs. append.** When `service_test.go` is absent, `createGoFile` writes a fresh file (package clause, the
  computed imports, the scaffolds) and gofmts it. When it exists, `appendGoDecls` appends the new functions and
  adds only the imports the file lacks, inserting them into the existing import block; the final `format.Source`
  sorts them into place without disturbing the file's other imports or hand-written code (gofmt operates on source
  text, so comments survive). The `svc`/`configonly` goldens pin the create path; `servicetest_test.go` pins the
  marker detection, kind selection, and the append/import-merge path.

## What genservice does not touch

- `*api/clientext.go` - hand-written client extensions that cannot be derived from `definition.go`. Never read or
  written.
- `service.go` handler *bodies* and `OnStartup`/`OnShutdown` - the hand-written half of the microservice. genservice
  syncs a handler's godoc and appends a stub for a missing handler, but never touches an existing handler's body
  (see Syncing handler godoc and Scaffolding handlers).
- `service_test.go` test *bodies* - once a feature has a `// MARKER` test, it is never rewritten; genservice only
  appends stubs for features that lack one (see Scaffolding feature tests).
- Anything outside the api package and the service directory.

## Known limitations

- The import-alias heuristic uses the last path segment as the package name, which is wrong for a `/v2`-style
  module path. It has not bitten any project package.
- Embedded In/Out struct fields are not flattened; In/Out types are assumed to be declared in the api package.
- Metric labels are assumed to be string-typed, which holds for every current metric.
