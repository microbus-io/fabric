---
name: upgrade-microbus
description: TRIGGER when the user asks to upgrade the project to a newer or the latest version of Microbus, or to update the framework. Each Microbus release ships this one self-contained skill; it applies that release's single-version migration, then chains to the next release's copy of this skill until the target version is reached.
---

**CRITICAL**: Do NOT explore or analyze the project unless a migration step below tells you to. This skill is self-contained.

**CRITICAL**: This skill is self-propagating. It migrates the project by exactly one version increment (Phase 1), then replaces itself on disk with the next release's copy of this same skill and runs that (Phase 2). Only the "Release constants" and Step 3 differ between releases; the Phase 1 and Phase 2 machinery is identical in every release.

## Release Constants

*The only part of this skill a release author edits. Set these when cutting a release.*

- **`SOURCE`** = the version this release upgrades **from** (the previous release), e.g. `v1.44.0`.
- **`DEST`** = this release's own version, e.g. `v1.45.0`.
- **Migration** = the source edits taking a project from `SOURCE` to `DEST`, authored in Step 3. Empty when the release has no breaking changes.

```
SOURCE = v1.45.0
DEST   = v1.46.0
```

<!-- RELEASE AUTHOR: set SOURCE and DEST above, and fill Step 3, when cutting a release. -->

## Workflow

Copy this checklist and track your progress:

```
Upgrade Microbus:
- [ ] Step 1: Read the CURRENT and TARGET versions
- [ ] Step 2: Phase 1 - apply this release's increment (SOURCE -> DEST)
- [ ] Step 3: (release-specific migration, invoked by Step 2)
- [ ] Step 4: Phase 2 - chain to the next release, or finish
```

### Step 1: Read the CURRENT and TARGET Versions

Read the `github.com/microbus-io/fabric` version from `go.mod`; call it **`CURRENT`**. If the dependency is absent, this is not a Microbus project; exit.

Determine the **`TARGET`**. In order of precedence: the `TARGET` carried forward from the previous hop of the chain (see Step 4); the version the user named when starting the upgrade; or, if the user asked only to update or go to "latest", the latest published version:

```shell
go list -m -versions github.com/microbus-io/fabric
```

The versions are listed oldest to newest. A user-supplied `TARGET` is fixed for the whole run: carry it across every hop and do not silently re-derive it as "latest" on a later hop. If `CURRENT` is already `TARGET` (or newer), there is nothing to do; exit.

### Step 2: Phase 1 - Apply This Release's Increment (SOURCE -> DEST)

If `CURRENT` is not equal to `SOURCE`, skip straight to Step 4. (This happens on the very first hop: the project's installed copy of the skill is `CURRENT`'s own release, so its `DEST` equals `CURRENT` and its `SOURCE` is one behind - Phase 1 has nothing to do, and its only job is to bootstrap the chain in Step 4.)

Otherwise the project is exactly one version behind this release. Migrate it:

1. Pin the framework to `DEST`:

   ```shell
   go get github.com/microbus-io/fabric@DEST
   ```

2. Apply the migration in Step 3.

3. Regenerate each microservice's boilerplate with this release's generator, resolve dependencies, and compile-gate:

   ```shell
   find . -path ./vendor -prune -o -name definition.go -path '*api/definition.go' -print \
     | while read -r def; do
         svcdir=$(dirname "$(dirname "$def")")
         go run github.com/microbus-io/fabric/cmd/genservice "$svcdir"
       done
   go mod tidy
   go vet ./...
   ```

   `go vet` must pass before Step 4. A grep-guided or manual migration can leave a compile error at a site it could not fix mechanically; resolve those now - with the user when the fix is a design choice - so the project compiles at `DEST`. (A migration may deliberately leave a runtime `// TODO:` that still compiles; the final `go test` in Step 4 surfaces it.)

### Step 3: Release-Specific Migration (v1.45.0 -> v1.46.0)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.46.0 carries two independent migrations. Steps 3a through 3g apply to **every** project: a `Parallel`
signature change, the HTTP ingress and tracing changes, one deployment prerequisite, and two renamed metric
series. Steps 3h through 3n apply only to a project that defines tasks or workflows, or that configures
`foreman.core`, and 3h is the gate that lets everyone else skip them. Work through them in order; do not
reorder, because 3h exits early.

v1.46.0 changes the signature of `Parallel` (on the connector, the `*Service` base type, and the
`service.Executor` interface):

```go
// before
Parallel(jobs ...func() (err error)) error
// after
Parallel(ctx context.Context, jobs ...func(ctx context.Context) (err error)) error
```

`Parallel` still waits for all jobs to complete and returns the first error. What changed: each job now receives a
cancelable subcontext of the parent `ctx` that is canceled when any job errors, so a job that observes its context
can abandon its work early once a sibling has failed.

Generated code fixes itself: Step 2's `genservice` run regenerates `doOnObserveMetrics` in every `intermediate.go`.
Only hand-written call sites need edits.

#### 3a. Rewrite `Parallel` Call Sites (Grep-Guided)

Find every call site (`t.Parallel()` hits are Go's testing API, not this migration - skip them):

```bash
grep -rn --include='*.go' --exclude-dir=vendor '\.Parallel(' . | grep -v 't\.Parallel()'
```

For each hit:

1. Pass a context as the new first argument - the `ctx` already in scope, or `r.Context()` in a web handler.
2. Change each job from `func() error` to `func(ctx context.Context) error`. This includes job slices built for
   `Parallel`: `[]func() error` becomes `[]func(ctx context.Context) error`.
3. Decide, per job, which context its body should use:
   - **Use the job's `ctx` parameter** (shadowing the outer one) when the job makes downstream calls or queries and
     should abandon its work once a sibling job fails. This is the right default.
   - **Name the parameter `_` and keep using the outer `ctx`** when the job must run to completion regardless of
     sibling failures. This preserves the pre-v1.46.0 behavior, where jobs could not observe each other's errors:
     ```go
     svc.Parallel(ctx,
         func(_ context.Context) error { return svc.recordAudit(ctx, entry) },
     )
     ```

The choice in item 3 is a per-site judgment call: a read fan-out (parallel queries assembling one response) wants the
subcontext; independent side effects (audit writes, notifications) usually want to run to completion. Ask the user
when the intent is unclear.

#### 3b. Update `service.Executor` Implementations (Grep-Guided)

A project that hand-implements the `service.Executor` interface (rare - mocks or decorators around a service) must
update its `Parallel` method to the new signature:

```bash
grep -rn --include='*.go' --exclude-dir=vendor 'Parallel(jobs \.\.\.func()' .
```

Rewrite each declaration to `Parallel(ctx context.Context, jobs ...func(ctx context.Context) (err error)) error`
and thread the new arguments through the body.

#### 3c. Split the Ingress `AllowedOrigins` Config (Grep-Guided)

v1.46.0 splits the HTTP ingress's `AllowedOrigins` config into `AllowedCredentialedOrigins` (origins trusted with
credentialed requests; the wildcard is rejected) and `AllowedUncredentialedOrigins` (origins, or `*`, that may read
responses without credentials). The old name refuses startup when set to any non-empty value. Find every setting:

```bash
grep -rn 'AllowedOrigins' config.yaml config.local.yaml *_test.go 2>/dev/null
grep -rn --include='*.go' --include='*.yaml' --exclude-dir=vendor 'SetAllowedOrigins\|AllowedOrigins:' .
```

For each configured origin, ask the user which posture it needs - do not guess, this is a security decision:

- An origin whose users log in through the browser (cookie/`Authorization`-cookie flows) goes to
  `AllowedCredentialedOrigins`.
- A public-API consumer, and the `*` wildcard, go to `AllowedUncredentialedOrigins`. Note that under the old config
  `*` reflected the caller's origin *with* credentials; the new wildcard is uncredentialed by construction. A
  deployment that relied on credentialed access from arbitrary origins must now name those origins explicitly.

Both lists may be set together; a named credentialed origin takes precedence over the wildcard.

This step can only migrate the config files in this checkout. Production and staging deployments often set config
outside the repo (operator-managed `config.yaml`, env overrides), which this skill cannot see or edit. Tell the
user to apply the same rename wherever `AllowedOrigins` is set for `http.ingress.core` in their deployment
environments. This is safe to get wrong in only one direction: an environment still setting the old name refuses
to start with an error naming the two new configs, rather than silently coming up with a changed CORS posture.

#### 3d. Set `TrustedProxyHops` for Deployments Behind a Reverse Proxy (Config, Ask the User)

v1.46.0 changes how the HTTP ingress handles inbound `X-Forwarded-*` headers. Previously they were trusted
whenever present; now they are ignored and rewritten from the actual request unless the new `TrustedProxyHops`
config (default 0) says how many reverse proxies (CDN, load balancer) sit in front of the ingress. Ask the user
whether their deployments run the ingress behind trusted proxies; if so, set for `http.ingress.core`:

```yaml
http.ingress.core:
  TrustedProxyHops: 1   # e.g. Cloudflare only; 2 for CDN + load balancer, etc.
```

Without it, a proxied deployment sees the proxy's address as every client's `X-Forwarded-For` and the ingress's
own host in absolute URLs - a visible but non-silent regression. As with 3c, production config often lives outside
this checkout; tell the user to apply it in their deployment environments.

#### 3e. Remove Calls to the Removed `Span.SetRequest` (Grep-Guided)

v1.46.0 removes `trc.Span.SetRequest`. The structural request attributes it used to add (method, URL path, host)
are now attached to every span at creation, and headers and query arguments are never recorded in any deployment
because they routinely carry credentials. Find any callers:

```bash
grep -rn --include='*.go' --exclude-dir=vendor '\.SetRequest(' .
```

Delete each call - the attributes it added are already on the span. A caller that relied on its client-IP side
effect can call `span.SetClientIP(r.RemoteAddr)` directly, which remains available.

#### 3f. Confirm PROD and LAB Run NATS Over TLS (Deployment Prerequisite, Ask the User)

**This one can stop a deployment from starting, so raise it before finishing the upgrade.** From v1.46.0 a
connector aborts `Startup` when all three of these hold:

- the deployment is `PROD` or `LAB` (`LOCAL` and `TESTING` are never affected), and
- the microservice defines at least one config declared `Secret: true`, and
- the transport is a live NATS connection without TLS.

It fails with `refusing to start with secret configs over an insecure transport`. A bundle with no wire at all
(short-circuit only) counts as secure, so this is specifically about a plaintext NATS connection.

**Assume the project is affected.** Several shipped core microservices define secret configs of their own -
`metrics.core`, `bearertoken.core` and `foreman.core` among them - so an app that runs any of those in PROD or
LAB hits this even if it declares no secrets itself. List the project's own ones too, so the user can see the
full surface:

```bash
grep -rln --include='definition.go' --exclude-dir=vendor 'Secret:[[:space:]]*true' .
```

Tell the user, whether or not that grep matches: **before or together with this upgrade, PROD and LAB must
connect to NATS over TLS.** That is the `MICROBUS_NATS` URL and the broker's own configuration, which live in
the deployment environment rather than this checkout, so this skill cannot verify or change it. Nothing needs
to change in the project's code. An environment that already uses TLS is unaffected and needs no action.

#### 3g. Update Renamed Metric Series in Dashboards and Alerts (Grep-Guided)

Two metric changes rename or reshape series. Neither affects application code; both silently blank a panel or
alert that still queries the old name.

**The sequel connection-pool wait metrics became counters** (sequel v1.11.2), which renames one instrument and
adds the Prometheus `_total` suffix to both:

| Before (PromQL) | After (PromQL) |
|---|---|
| `sequel_pool_wait_count` | `sequel_pool_waits_total` |
| `sequel_pool_wait_duration_seconds` | `sequel_pool_wait_duration_seconds_total` |

Projects often vendor their Grafana dashboards, so rewrite the ones in the repo. Run this as-is; it is
idempotent and matches nothing in a project that keeps its dashboards elsewhere, so there is no separate
check to make first (do not grep for the old names to decide whether to run it - `sequel_pool_wait_duration_seconds`
is a prefix of its own migrated form, so a plain grep reports a hit on an already-migrated file):

```bash
grep -rl 'sequel_pool_wait' --include='*.json' --include='*.yaml' --include='*.yml' --exclude-dir=vendor . \
  | xargs -r perl -pi -e '
      s/\bsequel_pool_wait_duration_seconds\b(?!_total)/sequel_pool_wait_duration_seconds_total/g;
      s/\bsequel_pool_wait_count\b/sequel_pool_waits_total/g;
    '
```

Report which files it changed, if any, so the user knows which dashboards to re-import.

**The server histograms dropped two labels.** `microbus_server_request_duration_seconds` and
`microbus_server_response_body_bytes` no longer carry `route` or `canonical`; both were derivable from the
`service`, `port` and `name` labels that remain. This one cannot be rewritten mechanically, because a query
that grouped by `route` has to be re-aggregated on something else - which grouping is right is the author's
call:

```bash
grep -rn 'microbus_server_request_duration_seconds\|microbus_server_response_body_bytes' --include='*.json' --include='*.yaml' --include='*.yml' --exclude-dir=vendor .
```

Show the user any hit that references `route` or `canonical` and ask what it should group by instead.

Finally, tell the user to apply both changes to any dashboard or alerting rule kept outside this repo - a
Grafana instance, an ops repo, a Prometheus rules file. This skill can only reach what is checked in here.

#### 3h. Skip 3i Through 3n If the Project Has No Workflows

v1.46.0 moves the embedded workflow engine from dwarf v0.9.5 to v0.10.5, which changes the `workflow` package's
state model and the foreman's configuration. Check whether any of it applies:

```bash
grep -rln --include='*.go' --exclude-dir=vendor 'microbus-io/dwarf' . | grep -v '/manifest.yaml'
grep -rn 'foreman.core' config.yaml config.local.yaml 2>/dev/null
```

If neither finds anything, the project has no workflows and no foreman configuration: 3i through 3n do not
apply, and Step 3 is complete - go to Step 4. `go.mod` still moves to dwarf v0.10.5 either way (it is a
fabric dependency), along with transitive bumps to sequel and boolexp, and none of those need source changes.

#### 3i. Rename `flow.Delete` to `flow.Del` (Mechanical)

`Flow.Delete` is now `Flow.Del`. Nothing else about it changed:

```bash
grep -rl --include='*.go' --exclude-dir=vendor 'flow\.Delete(' . | xargs -r perl -pi -e 's/\bflow\.Delete\(/flow.Del(/g'
```

The receiver is conventionally named `flow` in a task handler. If a project names it something else, widen the
pattern to that name, and check the result: `Delete` is a common method name on unrelated types, so a blind
repo-wide rename is wrong.

#### 3j. Replace `flow.Transform` (Grep-Guided)

`Flow.Transform(newKey, oldKey, ...)` - clear all state, then re-introduce the listed fields under new names - is
removed with no direct replacement. Find every call:

```bash
grep -rn --include='*.go' --exclude-dir=vendor '\.Transform(' .
```

Rewrite each as a snapshot, a clear, and explicit re-sets. `flow.Snapshot()` returns a decoded copy that is
unaffected by the following `Clear`, so the rename is safe in one dispatch:

```go
// before
flow.Transform("conversation", "messages", "answer", "answer")

// after
snap := flow.Snapshot()
var messages []llmapi.Item
snap.Get("messages", &messages)
var answer string
snap.Get("answer", &answer)
flow.Clear()
flow.Set("conversation", messages)
flow.Set("answer", answer)
```

A `Transform` used purely as a "keep these" (all pairs of the form `("name", "name")`) is usually clearer as
`flow.Del` of the fields that should go. Ask the user when the intended shape is not obvious from the call.

#### 3k. Migrate `map[string]any` State to `workflow.State` (Grep-Guided)

Flow state is now a `workflow.State` value rather than a `map[string]any`. The affected fields and returns:

| Was `map[string]any` | Now `workflow.State` |
|---|---|
| `FlowOutcome.State`, `FlowOutcome.InterruptPayload` | same names, `State`-typed |
| `FlowStep.State`, `FlowStep.Changes`, `FlowStep.InterruptPayload` | same names, `State`-typed |
| `Flow.Snapshot()`, `Flow.InterruptRequested()`, `Flow.SubgraphRequested()` | return `State` |
| `workflow.BaggageFrom(ctx)` (was `any`) | returns `State` |

`State` is not a map, so indexing, `len`, `range`, and comparison against `nil` all stop compiling. Find the sites:

```bash
grep -rn --include='*.go' --exclude-dir=vendor '\.State\b\|\.Changes\b\|\.InterruptPayload\b\|Snapshot()\|BaggageFrom(' .
```

Rewrite each with the typed accessors, which is usually shorter than what it replaces:

- `m["k"].(float64)` -> `s.GetInt("k")` / `s.GetFloat("k")`; likewise `GetString`, `GetBool`, `GetDuration`,
  `GetStrings`.
- A struct or slice value -> `ok, err := s.Get("k", &target)`.
- `len(m)` -> `s.Len()`; `m != nil` -> `!s.IsZero()`.
- Unmarshaling the whole state into a struct (a `json.Marshal` then `json.Unmarshal` round trip) -> `s.Parse(&target)`.
- Code that genuinely needs a generic map (a UI rendering arbitrary keys) -> `m := map[string]any{}; s.Parse(&m)`.

`workflow.MergeState` is also removed; the equivalents are the `State` methods `Merge`, `MergeReduce`, and
`MergeReduceAll`.

#### 3l. Fix the Remaining Removed Members (Grep-Guided)

Four smaller removals, each a compile error at the call site:

```bash
grep -rn --include='*.go' --exclude-dir=vendor 'WithInputFlow(\|\.Duration()\|HasFanIn()\|Renderer(' .
```

- **`Executor.WithInputFlow` takes a `*workflow.RawFlow`**, not a `*workflow.Flow`, because seeding a flow's state
  is now a raw-orchestration operation. Build the carrier with `workflow.NewRawFlow()` and populate it with
  `SetRawState(state)`. This is a test-only surface; `Flow.SetState` is gone for the same reason (a task's output
  is its changes, never a raw state write).
- **`FlowSummary.Duration()` and `FlowStep.Duration()` are removed.** Compute it: `UpdatedAt.Sub(StartedAt)`,
  guarded on neither being zero and on the result being non-negative.
- **`Graph.HasFanIn()` is removed** with no replacement. A caller inspecting a graph's shape should read
  `graph.Transitions()` instead.
- **`FlowRenderer.Render()` and `GraphRenderer.Render()` return only a string**, no error. Drop the second return
  value and the error branch that followed it.

#### 3m. Rewrite the `foreman.core` Shard Configuration (Config, Ask the User)

The foreman's database configuration is replaced. Each shard now declares its own connection string plus the CPU
count of its database server, from which the engine derives that shard's connection budget and its share of new-flow
placement. The old single-DSN-plus-count shape is gone:

| Removed | Replacement |
|---|---|
| `SQLDataSourceName` (one DSN, `%d` templated per shard) | `Shards`, a JSON array with one entry per shard |
| `NumShards` | the length of `Shards` |
| `SQLConnectionPool` | `MaxOpenConns`, now an expert override defaulting to 0 (derive) |

```yaml
foreman.core:
  Shards: '[{"index":1,"dsn":"postgres://user:pass@db1:5432/flows","virtualCPUs":16},
            {"index":2,"dsn":"postgres://user:pass@db2:5432/flows","virtualCPUs":16}]'
```

Per entry: `index` is >= 1, unique, and stable across restarts (it is encoded into every flow key created on the
shard, and the index-to-DSN mapping must be identical on every replica); `dsn` is used verbatim, with no templating,
so a percent-encoded credential survives intact; `virtualCPUs` is the vCPU count off the database instance's spec
sheet, assumed to be 2 when omitted; `cordoned` excludes the shard from new-flow placement while everything already
resident keeps running.

Ask the user for each shard's DSN and vCPU count rather than guessing - a large database left at the assumed 2 vCPUs
runs at a fraction of its capacity, and the DSN is a secret that belongs in `config.local.yaml` or the operator's
own configuration, not in a committed `config.yaml`. Then check the two override knobs:

- `Workers` now defaults to `-1`, meaning "let the engine derive the ceiling". A project that left it at the old
  default of 64 should drop the setting so it picks up the derived value; one that pinned it deliberately keeps
  its number. `0` remains meaningful and distinct: a replica that creates, awaits, and serves reads but never
  executes a task.
- `MaxOpenConns` replaces `SQLConnectionPool` and defaults to `0`, meaning "derive from `virtualCPUs`". Set it only
  when the connection budget is constrained by something the engine cannot see, such as a shared database or an
  external pooler.

As with 3c and 3d, production config often lives outside this checkout; tell the user to apply the same rewrite
wherever `foreman.core` is configured in their deployment environments.

#### 3n. Drop Calls to the Removed `foremanapi.Signal`

Dwarf replicas no longer message each other: a fleet sharing a database coordinates entirely by polling it. The
foreman's `Signal` endpoint and its `SignalIn`/`SignalOut` types are removed along with the host-side
`SignalPeers`. Nothing replaces them, and nothing needs to.

```bash
grep -rn --include='*.go' --exclude-dir=vendor 'foremanapi\.Signal\|\.Signal(ctx' .
```

Delete each call. A test that exercised cross-replica coordination through it should assert on the outcome instead:
several foreman replicas in one app resolve to the same databases, so the work simply completes across them.

**Tell the user this upgrade needs a maintenance window.** The engine's schema migrations are forward-only and run
at `Startup`, so the first replica of the new version migrates the database every old replica is still using, and
there is no downgrade path. The supported procedure is: back up every shard, drain the whole fleet, start one
replica and confirm it comes up clean, then start the rest. Flows survive it untouched - pending steps stay pending
and interrupted flows stay parked.

### Step 4: Phase 2 - Chain to the Next Release, or Finish

Find the next release to apply: from `go list -m -versions github.com/microbus-io/fabric`, the smallest published version `NEXT` in the range `DEST < NEXT <= TARGET` (semver). The upper bound `<= TARGET` is what stops the chain from overshooting a user-supplied `TARGET`.

**If there is no such `NEXT`** the chain is complete - either `TARGET` is at or below `DEST`, or no published release lies between `DEST` and `TARGET`. The correctness gate is `go vet`, which already passed at every increment including this one - it is deterministic and always runnable locally. As a final behavior check, run the tests where feasible:

```shell
go test ./...
```

- **Tests can't run here, or are known-flaky / need external services:** say so and rely on the `go vet` gate; the upgrade stands.
- **Tests pass:** the upgrade is confirmed.
- **Tests fail:** triage each failure rather than declaring success or failure blindly.
  - *Caused by a migration* (a test referencing a removed or renamed symbol, an incompletely migrated call site): fix it - that is finishing the migration - and re-run.
  - *Not clearly the migration's doing* (a deliberate `// TODO:` a migration left to fill, or a pre-existing / environmental failure): report it and ask the user how to proceed - fix it together, accept the upgrade with the failure recorded, or roll back. Do not silently declare success on failures you have not explained.

If `CURRENT` is still below `TARGET` here, `TARGET` names no published release the chain could reach (for example a version that was never published); say so and leave the project at `CURRENT`.

**If `NEXT` exists**, hand off to it. Install `NEXT`'s agent rules and skills - which include `NEXT`'s copy of this skill (`NEXT` already carries its `v` prefix, e.g. `v1.46.0`):

```shell
git clone --depth 1 --branch NEXT https://github.com/microbus-io/fabric temp-clone
rm -rf .claude/rules/{auth.txt,microbus.md,python.txt,sequel.txt,workflows.txt}
rm -rf .claude/skills/{microbus,python,sequel,upgrade}
cp -r temp-clone/.claude .
rm -rf temp-clone
```

(Removing the fabric-managed rules files and skill groups before the copy purges anything `NEXT` dropped; `cp` overwrites and adds but never deletes. Rules and skills the project added of its own are left untouched.)

**CRITICAL**: The copy just overwrote this skill on disk with `NEXT`'s copy. Now **Read** `.claude/skills/microbus/upgrade-microbus/SKILL.md` (the new content) and follow it from Step 1, **carrying the same `TARGET` forward** - if the user supplied a specific `TARGET`, it stays the `TARGET` for `NEXT` and every later hop; only an unspecified `TARGET` defaults to "latest". Do not continue from this in-context copy, because `NEXT`'s `SOURCE`/`DEST` and migration are what must run next. `CURRENT` is now `DEST`, which is `NEXT`'s `SOURCE`, so `NEXT`'s Phase 1 fires and advances one more increment. The chain ends at the hop that finds no `NEXT` in range.
