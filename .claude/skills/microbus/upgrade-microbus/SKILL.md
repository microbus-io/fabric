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
SOURCE = v1.48.1
DEST   = v1.49.0
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

### Step 3: Release-Specific Migration (v1.48.1 -> v1.49.0)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.49.0 moves the foreman to dwarf v0.14.0 (by way of v0.13.0) and sequel v1.18.0. It requires Go 1.27.3: Step 2's
`go get` raises the `go` line in `go.mod`, and a Go older than 1.27.3 downloads the newer toolchain on its own unless
`GOTOOLCHAIN` is set to `local`, in which case ask the user to install Go 1.27.3 or later.

The foreman's API changes with the engine's:

- `List` is replaced by `ListOpen` (running and interrupted flows) and `ListClosed` (archived closed flows, which
  requires `Query.FromTime`, `Query.ToTime` and `Query.Limit`, with a window of at most 31 days). Neither paginates:
  `NextCursor` and `Query.Cursor` are gone, and so are `Query.NewerThan` and `Query.OlderThan`.
- Threads are gone: `FlowOptions.ThreadKey`, `Query.ThreadKey` and `FlowSummary.ThreadKey` are removed, and
  `Continue(ctx, flowKey, additionalState)` builds on exactly the completed flow it is given.
- `Delete` removes only a closed flow (an open one is a 409), and `Purge` returns `(removed, done, err)`.
- New: the `Retention` config (days the archive keeps closed flows, default 30), the `FlowExpiration` config (off by
  default), and the `PurgeBefore` endpoint.

#### 3a. Tell the User That the Foreman's Databases Must Be Recreated

This is not a source edit, but the upgrade does not work without it, so tell the user before going on. dwarf v0.13.0
and v0.14.0 edited their migrations in place, so a database an earlier release created cannot be upgraded, and
nothing detects one: the foreman starts on it, and every list of flows then fails on a missing column. Every foreman
shard must be a new, empty database, in every environment, and flows in the old databases do not carry over.
In addition:

- On PostgreSQL, the foreman must connect as the role that owns its tables.
- MySQL must be 8.0 or later, and MariaDB 10.6 or later.
- SQL Server is now reached through `github.com/microsoft/go-mssqldb`. A project that imports
  `github.com/denisenkom/go-mssqldb` itself must switch to the new package: both register the `mssql` driver, and
  linking both panics at startup. If a development server's self-signed certificate now fails the TLS handshake at
  login, set `encrypt=disable` in its DSN.
- Closed flows are now deleted about 30 days after they close (the foreman's `Retention` config sets the days).

In a LOCAL deployment with no `Shards` config, the foreman keeps its flows in a SQLite file in the working directory
of the process that runs it, typically `main/`. Find the old files:

```bash
find . -path ./vendor -prune -o -name 'shard_*.local.sqlite*' -print
```

Ask the user before deleting them; the foreman creates fresh ones on its next start.

#### 3b. Migrate Callers of the Foreman (Grep-Guided)

Step 2's `go vet` reports the compile breaks (`List`, the removed `Query` and `FlowOptions` fields, `Purge`'s extra
return value). Find them, and the `Continue` and `Delete` calls that still compile but changed meaning:

```bash
grep -rlE --include='*.go' --exclude-dir=vendor 'foremanapi' . \
  | xargs -r grep -HnE '\.(List|Continue|Purge|Delete)\(|\b(ThreadKey|NewerThan|OlderThan|NextCursor|Cursor)\b'
```

Rewrite each site:

- **`List`**: list open flows with `ListOpen`, and closed flows with `ListClosed` and a window
  (`FromTime: now.Add(-window), ToTime: now`). Set `Query.Limit`, which is now required. Drop any loop over
  `NextCursor`. Which list a site wants, and how wide a closed window, is a design choice; ask the user when the code
  does not make it clear. Do not merge the two lists to imitate the old one without saying so: a flow that closed in
  the last minute is in neither until it is archived.
- **`NewerThan` / `OlderThan`**: `FromTime` and `ToTime`. "Older than 30 days" is
  `ToTime: time.Now().Add(-30 * 24 * time.Hour)`.
- **`ThreadKey`**: remove it from `FlowOptions` and `Query`. Code that kept a conversation by its thread key keeps the
  key of the latest flow instead: call `Continue` with that key and store the key it returns. Continuing one flow
  twice now makes two flows, and continuing a flow that is not completed is a 409. A `Create` that joined a thread to
  set its own options becomes a `Create` with the previous flow's final state, from `Snapshot`.
- **`Purge`**: take `(removed, done, err)`, set `Query.Limit`, `FromTime` and `ToTime`, and call it until `done`. It
  removes only archived flows; `PurgeBefore` drops whole archive days and is the cheaper way to remove in bulk.
- **`Delete`**: a running or interrupted flow is now a 409; `Terminate` it first if the code deleted open flows.

#### 3c. Rename the Metrics in Dashboards and Alerts

The metric `dwarf_flows_ended` is now `dwarf_flows_closed`, and `dwarf_operation_seconds`'s `op="list"` label is now
`op="list_open"` or `op="list_closed"`:

```bash
grep -rlI --exclude-dir=.git --exclude-dir=vendor --exclude-dir=.claude 'dwarf_flows_ended' . \
  | xargs -r perl -pi -e 's/dwarf_flows_ended/dwarf_flows_closed/g'
grep -rnIE --exclude-dir=.git --exclude-dir=vendor --exclude-dir=.claude 'op=\\?"list\\?"' .
```

Rewrite each `op="list"` selector the second command finds to `op=~"list_open|list_closed"`. Fabric's own dashboard
(`setup/grafana/dashboards/workflow-overview.json` in the framework repo) also gained an "Archive & Retention" row;
offer to copy it if the project keeps its own copy.

#### 3d. Review What Changed Behavior (No Edit Unless a Site Needs One)

Review these, and rewrite a site only when its intent no longer holds:

- **`flow.Retry`'s `giveUpAfter` bounds the total of the backoff delays**, not the time since the step was created.
  Time spent queued or running no longer counts, so a retry gives up later on a loaded system; a task that wants its
  run time bounded keeps its own budget. Comments that say "give up after 1h" mean an hour of backoff now.
- **A subgraph that cannot start comes back as `flow.Subgraph`'s `err`**: a 503 when no microservice serves the
  workflow, 404 for no such graph, 400 for an invalid one. A task that already returns `Subgraph`'s error needs no
  change; one that should wait out a 503 arms `flow.Retry`. For the same reason, a `Create` or `Run` with a workflow
  URL that no microservice serves now returns 503 rather than 404, and a step whose task host never answers fails
  with a 503. Code that branches on those status codes needs updating.
- **State field names are checked.** A name is an ASCII letter or underscore, then ASCII letters, digits, underscores
  or hyphens. Step 2's `genservice` rejects a task or workflow argument whose `json` name breaks the rule (rename the
  tag), and a task that writes such a name with `flow.Set` and its relatives panics, failing the step. This finds
  a literal first argument that breaks the rule; check names built at run time, and `flow.Del`'s later arguments,
  by reading the code:

  ```bash
  grep -rnE --include='*.go' --exclude-dir=vendor \
    'flow\.(Set[A-Za-z]*|Overwrite|Del)\("([^A-Za-z_"][^"]*|[^"]*[^A-Za-z0-9_"-][^"]*)?"' .
  ```

- **A `FlowOptions.FairnessKey`** is at most 256 ASCII letters, digits, `-`, `_` and `.`; any other key is a 400.
- **Tests that build a flow carrier or a key by hand.** A carrier's JSON needs `"v": 1`, and `sleepDuration`,
  `backoffInitialDelay` and `backoffMaxDelay` are now `sleepMillis`, `backoffInitialDelayMillis` and
  `backoffMaxDelayMillis`, in milliseconds. A hand-built flow or step key must use the new shape
  (`{shard}-{hour}-{flowID}-{token}`); better, take keys from the foreman.

Finally, tell the user that the foreman and every microservice that hosts tasks must be deployed together: the flow
carrier is now versioned, and a foreman and a task on different releases fail the step.

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
