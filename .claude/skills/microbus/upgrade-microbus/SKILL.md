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
