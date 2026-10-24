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
SOURCE = v1.48.0
DEST   = v1.48.1
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

### Step 3: Release-Specific Migration (v1.48.0 -> v1.48.1)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.48.1 fixes workflow tasks that call Python. The `add-python-task` pattern awaited the Python call with
`venv.Await` and retried only on a 408, but an `Await` that runs out of time returns `context.DeadlineExceeded`, a
500, and only at the task's own deadline, so the retry never fired and a long call failed the task instead of being
rejoined. The fix polls the call with `venv.Poll`, new in pyvenv v1.1.0, which Fabric v1.48.1 requires, so Step 2's
`go mod tidy` already moves the project to it.

#### 3a. Skip the Rest If the Project Has No Python Tasks

```bash
grep -rnE --include='*.go' --exclude-dir=vendor 'StatusRequestTimeout && flow\.Retry' .
```

If this finds nothing, no task carries the old pattern, and Step 3 is complete - go to Step 4. Hits that are not
preceded by a `venv.Await` on a pyvenv `Venv` (a retry after a downstream bus call, where a 408 is right) are not
part of this migration; leave them.

#### 3b. Rewrite Each Task to Poll (Grep-Guided)

At each hit that follows a `venv.Await`, the task has this shape, with the state key usually `pyCallID`:

```go
var out myserviceapi.MyTaskOut
err = svc.venv.Await(ctx, callID, &out)
if err != nil {
    if errors.StatusCode(err) == http.StatusRequestTimeout && flow.Retry(0, 1.0, 0, 0) {
        return 0, nil
    }
    flow.SetString("pyCallID", "") // clear on terminal error so downstream steps don't see it
    return 0, errors.Trace(err)
}
flow.SetString("pyCallID", "") // clear on success so downstream steps don't see it
return out.Output1, nil
```

Rewrite it to poll, keeping the task's own output type, zero values, state key and `Retry` arguments:

```go
var out myserviceapi.MyTaskOut
done, err := svc.venv.Poll(ctx, callID, &out)
if err != nil {
    return 0, errors.Trace(err)
}
if !done {
    flow.Retry(0, 1.0, 0, 0) // unlimited; the retry polls the same call
    return 0, nil
}
flow.Del("pyCallID") // the call is consumed, so later steps must not see it
return out.Output1, nil
```

The clear before the error return is dropped because it never had an effect: an error discards the writes of the run
that returns it. If the task's `Retry` was bounded (a non-zero `giveUpAfter`) and its result is checked, keep that
check on the `!done` path and return an error when `Retry` reports the horizon is spent. Remove the `net/http`
import if the rewrite leaves it unused; `go vet` in Step 2 reports it.

Calls to `venv.Await` or `venv.CallAndAwait` outside a workflow task (a functional endpoint running a call within one
request) are unaffected; leave them.

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
