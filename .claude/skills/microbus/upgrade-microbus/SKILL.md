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
SOURCE = v1.50.0
DEST   = v1.51.0
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

### Step 3: Release-Specific Migration (v1.50.0 -> v1.51.0)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.51.0 moves the foreman to dwarf v0.16.0 (by way of v0.15.0) and sequel v1.20.2, which Step 2's `go get` and
`go mod tidy` already pull in. The foreman gains `ListRecentlyClosed`, the closed flows not archived yet, and
`Summary`, one flow's summary by its key; every `FlowSummary` now names its `RootFlowKey` and `ParentStepKey`. The
release also changes how the short-circuit transport delivers to overlapping subscriptions, and the status `Chat`
returns for a task tool (3g).

#### 3a. Tell the User That the Foreman's Databases Need a Schema Change

This is not a source edit, but the foreman fails without it, so tell the user before going on. dwarf v0.15.0 and
v0.16.0 edited their migrations in place, so a database v1.50.0's foreman created does not pick the changes up, and
nothing detects it: the foreman starts, then fails on its first write of a flow or step. Each environment's foreman
shards need one of:

- **New, empty databases.** Flows in the old databases do not carry over.
- **The statements by hand**, with every foreman replica stopped: first those in the Breaking section of dwarf's
  [v0.15.0 release notes](https://github.com/microbus-io/dwarf/releases/tag/v0.15.0), then those of
  [v0.16.0](https://github.com/microbus-io/dwarf/releases/tag/v0.16.0), on the live tables and on every archive day.
  The v0.16.0 statements rewrite their tables, so plan the window by the size of `dwarf_steps`.

Which one each environment takes is the user's choice: ask. Tests need nothing, since the foreman runs on
throwaway databases under `TESTING`. In a LOCAL deployment with no `Shards` config, the foreman keeps its flows in a
SQLite file in the working directory of the process that runs it, typically `main/`, where a new file is the simple
path. Find the old files:

```bash
find . -path ./vendor -prune -o -name 'shard_*.local.sqlite*' -print
```

Ask the user before deleting them; the foreman creates fresh ones on its next start.

#### 3b. Tell the User What Else Changed in the Foreman

- **Roll out together.** A task's carrier now holds a subgraph's error as a JSON object rather than a string. Code
  that uses the generated clients needs no change, but a foreman and a task-bearing microservice built on different
  releases misread a subgraph's error: deploy them together, as for every dwarf bump.
- **Text PostgreSQL cannot store is refused on every database.** A NUL in a string, or invalid UTF-8 in JSON passed
  through as given, fails the step that writes it, and is a 400 at `Create`, `Run`, `Resume`, `Continue` and `Fork`.
  It used to pass on SQLite. An LLM's or a tool's output is the likely way in; carry binary data base64-encoded.
- **An expired flow is deleted, not archived.** A flow the foreman's `FlowExpiration` ends is terminated as before
  and deleted within minutes, so `ListClosed` never lists it. The check runs hourly, or every half of the expiration
  when that is shorter.

#### 3c. Rename the Duration Fields in Hand-Written JSON (Grep-Guided)

A duration crossing the foreman's API as JSON is now an integer number of milliseconds:
`FlowOptions.TimeBudget` is `timeBudgetMillis` (was `timeBudget`, in nanoseconds), and `ShardSummary.Latency` is
`latencyMillis`. Go code picks the new names up with the bump. A body written by hand that still sends `timeBudget` is
ignored, and the flow runs on the default budget. Find candidates outside Go code, and in Go raw strings:

```bash
grep -rnIE --exclude-dir=.git --exclude-dir=vendor --exclude-dir=.claude '"(timeBudget|latency)"' .
```

A match in a `manifest.yaml`, or a `timeBudget` key that is not a foreman `FlowOptions` body, is unrelated: leave it.
Rewrite each foreman body to `"timeBudgetMillis": <milliseconds>`, and each reader of a shard's `latency` to
`latencyMillis`.

#### 3d. Update Dashboards and Alerts on the dwarf Metrics (Grep-Guided)

`dwarf_operation_seconds` is now `dwarf_shard_operation_seconds`:

```bash
grep -rlI --exclude-dir=.git --exclude-dir=vendor --exclude-dir=.claude 'dwarf_operation_seconds' . \
  | xargs -r perl -pi -e 's/\bdwarf_operation_seconds/dwarf_shard_operation_seconds/g'
```

Then find the metrics that are gone or changed shape:

```bash
grep -rnIE --exclude-dir=.git --exclude-dir=vendor --exclude-dir=.claude \
  'dwarf_(archive_days|reap_pass_seconds|archive_lag_seconds|archive_failing_seconds)' .
```

- `dwarf_archive_days` is removed. `dwarf_archive_oldest_day_age_seconds` shows the retention being kept: about the
  retention plus a day.
- `dwarf_reap_pass_seconds` is removed. The background work is timed per job by `dwarf_janitor_batch_seconds`
  (labels `job` and `part`), and whether each job keeps moving is `dwarf_janitor_progress_age_seconds`.
- `dwarf_archive_lag_seconds` and `dwarf_archive_failing_seconds` gain a `job` label, so a shard reports several
  series of each: a query that reads one series per shard must aggregate with `max`.

Rewrite the queries that read a removed metric; which replacement a panel or an alert wants is the user's choice
when it is not clear. The dashboard Fabric ships, `setup/grafana/dashboards/workflow-overview.json` in the framework,
already reads the new metrics, and a project that copied it can copy it again.

#### 3e. Optionally List the Recently Closed Flows (Ask First)

A flow that closes leaves `ListOpen` at once but reaches `ListClosed` only when it is archived, about a minute
later, so for that minute it was in neither list. `ListRecentlyClosed` lists it. Find the callers of the closed list:

```bash
grep -rlE --include='*.go' --exclude-dir=vendor 'foremanapi' . | xargs -r grep -HnE '\.ListClosed\('
```

A caller that should see every closed flow calls `ListRecentlyClosed` and then `ListClosed`, in that order, and
drops duplicates by `FlowKey`. Whether a caller wants that is a design choice, so ask. Separately, `Resume`,
`Cancel`, `Terminate`, `Delete` and `Continue` take a root flow's key and refuse a subgraph child's: code that holds
a child's key can read its `RootFlowKey` with `Summary`.

#### 3f. Rename the Email Parsing Package

`smtpingressapi.Email` is an alias of the `Email` type of `github.com/go-letters/letters`, the successor to the
deprecated `github.com/mnako/letters`, with the same API. A project that imports the old package itself, for example
to name the type of an attachment, now holds a type that does not match. Rename the import:

```bash
grep -rlF --include='*.go' --exclude-dir=vendor '"github.com/mnako/letters"' . \
  | xargs -r perl -pi -e 's#"github.com/mnako/letters"#"github.com/go-letters/letters"#'
```

Step 2's `go mod tidy` then drops the old module, and `gofmt -w` on the edited files restores the import order.

#### 3g. Tell the User About the Other Changes

`llm.core`'s `Chat` now fails with a 400 rather than a 500 when a tool URL names a task, the same status as a tool
URL that is unknown or that the caller is not authorized for. Such a call already failed, so only its status
changes.

A request that several subscriptions in one queue match, such as `/{p...}` and `/orders/{p...}` in a
microservice's default queue, now runs one of their handlers when the caller is in the same process, as it already
did over NATS. It used to run every one of them. Code that relied on each overlapping handler running needs no edit
to compile, so tell the user about the change.

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
