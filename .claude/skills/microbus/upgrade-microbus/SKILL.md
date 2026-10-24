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
SOURCE = v1.47.0
DEST   = v1.48.0
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

### Step 3: Release-Specific Migration (v1.47.0 -> v1.48.0)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.48.0 moves the embedded workflow engine to dwarf v0.12.0 (and sequel to v1.15.0), which changes three things a
project can notice:

- **A write to a reduced field is a contribution everywhere.** A field wired with `graph.SetReducer` is folded
  through its reducer at every hand-off (the next step, the fan-in, the final state and `Continue`), not only at a
  fan-in. In v1.47.0 a write outside a fan-out branch was the field's whole value; now it adds to the value, so a
  task that returns the value it read plus its addition adds the history twice.
- **Every value a task returns without an error is applied to the state,** zero value included. The regenerated
  `intermediate.go` calls `flow.Set` once per output. v1.47.0 skipped an output equal to its input; now it is
  applied too, which matters only in the two narrow cases at the end of 3d.
- **Renamed and removed engine APIs, a new priority range, and an engine schema changed in place.**

Work through the steps in order; 3a narrows the scope.

#### 3a. Determine the Scope

```bash
grep -rln --include='*.go' --exclude-dir=vendor 'microbus-io/dwarf\|foremanapi' .
grep -rln --include='*.go' --exclude-dir=vendor 'foreman.NewService' .
```

If the first grep finds nothing, the project neither defines nor drives workflows: skip 3b-3g. If the second grep
also finds nothing, the project does not run the foreman either: skip to 3k. Otherwise do 3h-3j, which apply to
every project that runs the foreman, even one with no workflows of its own. 3k applies to every project.

#### 3b. Apply the Renames (Mechanical)

```bash
grep -rlE --include='*.go' --exclude-dir=vendor 'microbus-io/dwarf/workflow' . \
  | xargs grep -lE 'workflow\.END\b|\.ParseState\(|\.SetEndpoint\(|SubWorkflow(URL|Name)|\bworkflow\.Node\b' \
  | while read -r f; do
      perl -pi -e 's/\bworkflow\.END\b/workflow.End/g; s/\.ParseState\(/.Parse(/g; s/\.SetEndpoint\(/.SetURL(/g; s/\bSubWorkflow(URL|Name)\b/Subgraph$1/g; s/\bworkflow\.Node\b/workflow.Task/g' "$f"
    done
```

This renames `workflow.END` to `workflow.End`, `Flow.ParseState` to `Parse`, `Graph.SetEndpoint` to `SetURL`,
`FlowStep.SubWorkflowURL`/`SubWorkflowName` to `SubgraphURL`/`SubgraphName`, and `workflow.Node` to `workflow.Task`,
only in files that import dwarf's `workflow` package.

#### 3c. Fix the Changed Signatures (Grep-Guided)

These no longer compile; Step 2's `go vet` lists every site. Find them up front:

```bash
grep -rnE --include='*.go' --exclude-dir=vendor 'err :?= (flow|f)\.Get\(|\.(Goto|Retry|Sleep|Interrupt|Subgraph)Requested\(|\.SetChanges\(|TerminalStatuses|\.TaskName\b|LatencyMs|\.Nodes\(\)|\.(WithGoto|OnError|Switch)\b|TestProxy' . \
  | grep -v '/intermediate.go:\|/mock.go:\|api/client.go:'
```

The generated files are excluded because Step 2 regenerates them. Skip hits that are not dwarf or `foremanapi`
types (`TaskName` on a `FlowStep` is unchanged). Rewrite each remaining site:

- **`Flow.Get` returns `(ok bool, err error)`.** `err := flow.Get("x", &v)` becomes `_, err := flow.Get("x", &v)`; use
  `ok` where the code checked `flow.Has` first. A call that ignores the result still compiles.
- **`GotoRequested`, `RetryRequested`, `SleepRequested`, `InterruptRequested` and `SubgraphRequested` moved to
  `workflow.RawFlow`.** A test that inspects the `workflow.Flow` from `Executor.WithOutputFlow` wraps it first:
  `raw := workflow.RawFlow{Flow: out}`, then `raw.SubgraphRequested()`.
- **`Flow.SetChanges(source, snap)` is gone.** A hand-written call becomes one `flow.Set(name, value)` per field, and
  the `snap := flow.Snapshot()` that fed it goes.
- **`FlowSummary.TaskName` and `Query.TaskName` are `StoppedTaskName`,** and the meaning changed: the task the flow
  stopped at, empty while it runs, so a filter on it matches no running flow. Ask the user when the code relied
  on a running flow's task name.
- **`workflow.TerminalStatuses` is gone;** use `workflow.IsTerminalStatus(status)`.
- **`foremanapi.ShardSummary`:** `Shard` is `Index`, `LatencyMs int` is `Latency time.Duration`, and `Steps` and
  `Flows` are gone (`ShardInfo` reports health only).
- **`Graph.Nodes()` is `Tasks()`,** and a `Transition`'s `WithGoto`/`OnError`/`Switch` booleans are one `Kind`
  (`""`, `workflow.TransitionSwitch`, `workflow.TransitionGoto`, `workflow.TransitionError`).
- **`engine.TestProxy`/`NewTestProxy` are `hosts.Mux`/`hosts.NewMux`** in `github.com/microbus-io/dwarf/hosts`:
  `HandleGraph(workflowURL, graph)`, and `HandleTask(taskURL, func)`, which takes a plain func.

#### 3d. Make Every Write to a Reduced Field a Contribution (Grep-Guided, Ask the User)

This is the change `go vet` cannot catch. List the reduced fields:

```bash
grep -rn --include='*.go' --exclude-dir=vendor 'SetReducer(' .
```

For each field, find every task that writes it: an output whose `json` tag names the field in a
`*api/definition.go` (often an `Out` pair, such as `ItemsOut` with `json:"items,omitzero"`), and any
`flow.Set`/`flow.Set*` of the field in a task body. At each writer:

- **Return only what the task adds.** `itemsOut = append(items, reply)` becomes `itemsOut = []Item{reply}`, and a
  running total becomes the task's own increment. A read of the field still returns the combined value.
- **A pass-through return adds the history again.** `if nothingToDo { return items }`, a `flow.Retry` rewind that
  returns the input, or a park's `yield` path that returns the input: return a value that changes nothing under
  the field's reducer, such as nil for a list.
- **Make contributions after `flow.Interrupt` or `flow.Subgraph`.** The task runs from the top again after the park
  (and after a `Retry`), and each run's contributions add to the earlier runs'.
- **To trim or reset the field, call `flow.Overwrite(name, value)`;** `flow.Del` is `Overwrite` with nil. Inside a
  fan-out branch `Overwrite` returns an error, and a `Del` or `Clear` of a reduced field fails the step: move the
  reset before the fan-out or after its fan-in.
- **A subgraph's result written into a reduced field of the parent is a contribution.** If the child returns the
  history it was given, write only the elements past it.

A rewrite that changes what a downstream task reads is a design decision; ask the user when the intended total is
not clear from the code.

Fields with no reducer need no change, except in two narrow cases, both because a returned value equal to the input
is now applied rather than skipped:

- **A task that both `flow.Set`s a field in its body and returns it as an output:** the returned value is applied
  after the body, so it now overwrites the `Set` even when it equals the input. Keep one of the two.
- **A fan-out branch that returns an output unchanged:** the pass-through is now a write, and at the fan-in it can
  overwrite another branch's change to the field. Return the field only from the branch that changes it.

#### 3e. Adjust Callers of `llm.core`'s `ChatLoop` (Grep-Guided)

`ChatLoop`'s `items` is now `Append`-reduced inside the workflow. A caller that runs it as a subgraph still receives
the whole conversation. A caller that `Continue`s a finished `ChatLoop` thread now has the `items` it passes
appended to the conversation, so it must pass only the new turn:

```bash
grep -rn --include='*.go' --exclude-dir=vendor 'llmapi.ChatLoop\|\.Continue(' .
```

At each `Continue` of a `ChatLoop` thread, pass only the new items, not the history.

#### 3f. Bring Priorities into Range (Grep-Guided, Ask the User)

Priorities now run from 1 to the foreman's new `MaxPriority` config (5 by default, at most 100), and an unset
priority is the midpoint, 3. The `DefaultPriority` config is gone. `Create` and `Run` reject a priority above
`MaxPriority` with a 400.

```bash
grep -rnE --include='*.go' --exclude-dir=vendor 'Priority:|\.Priority =' .
grep -rnE --include='*.yaml' --include='*.yml' --exclude-dir=vendor --exclude-dir=.git 'DefaultPriority|MaxPriority' .
```

For each explicit priority above 5, ask the user whether to remap the levels into 1-5 or to set `MaxPriority` under
`foreman.core` in `config.yaml` to the highest level used. Tell the user the default moved: in v1.47.0 an unset
priority ran at `DefaultPriority` (5 by default), behind every explicit 1-4; now it runs at 3, ahead of explicit
4s and 5s. `Continue`, `Fork` and subgraphs inherit the stored priority, so a thread created at the old default of 5
keeps running at 5, behind new unset flows. Remove any `DefaultPriority` setting.

#### 3g. Fix Assertions on Changed Status Codes (Grep-Guided)

```bash
grep -rnE --include='*_test.go' --exclude-dir=vendor 'Terminate\(|Await\(|\.Run\(' . | head -50
```

At each site that asserts an error code: `Terminate` of an ended flow now returns nil (was 409); an `Await` or `Run`
timeout is a 408 that also satisfies `errors.Is(err, context.DeadlineExceeded)`; a `Terminate` or `Cancel` reason
over 4 KB is a 400; any value over 256 MB is a 413.

#### 3h. Update the Foreman's Shard and Pool Settings (Grep-Guided, Ask the User)

```bash
grep -rnE --exclude-dir=vendor --exclude-dir=.git 'MaxOpenConns|virtualCPUs|VirtualCPUs' .
```

Look in `config.yaml`, `config.local.yaml`, and any deployment manifests or secrets that carry the foreman's
`Shards`:

- **`MaxOpenConns` is gone.** The engine derives each shard's connection budget (250-500 by round trip, clamped by
  the server's `max_connections`, shared by the shard's two dispatcher replicas). Where the project pinned it, ask
  the user whether to drop the setting or to set `maxConnections` on each `Shards` entry: the shard's fleet-wide
  budget, which matters for a database shared with other workloads or several shards on one server.
- **`virtualCPUs` is gone from `Shards` entries;** delete it. The new `weight` sets a shard's relative share of new
  flows; any weight other than 1 switches the fleet to weighted placement.

#### 3i. Update Dashboards and Alerts (Mechanical, then Grep-Guided)

The `dwarf_flows_terminated` metric (Prometheus `dwarf_flows_terminated_total`) is now `dwarf_flows_ended`, and the
engine's `FlowsTerminated` counter method is `FlowsEnded`. Rename them in text files only (`-I` skips binaries, such
as a stale build that embeds the old name):

```bash
grep -rlIE --exclude-dir=vendor --exclude-dir=.git --exclude-dir=.claude 'dwarf_flows_terminated|FlowsTerminated\(' . \
  | while read -r f; do perl -pi -e 's/dwarf_flows_terminated/dwarf_flows_ended/g; s/\bFlowsTerminated\(/FlowsEnded(/g' "$f"; done
grep -rnIE --exclude-dir=vendor --exclude-dir=.git --exclude-dir=.claude 'dwarf_steps_pending|dwarf_steps_oldest_pending_age_seconds' .
```

The pending-steps gauges are now reported every 30 seconds by one leading replica per shard. Aggregate
`dwarf_steps_pending` with `max` per shard and priority, then `sum` across shards, e.g.
`sum by (priority) (max by (shard, priority) (dwarf_steps_pending))`; the oldest-pending age stays a `max`. Either
can be absent for up to 30 seconds while a shard changes hands. `dwarf_flows_started` and `dwarf_flows_ended` gained
a `priority` label.

#### 3j. Recreate or Migrate Workflow Databases (Ask the User)

dwarf v0.12.0 changes the engine's schema by revising its existing migrations, so a database migrated by v1.47.0
does not receive the changes and the new engine fails on it.

In `LOCAL` development the foreman keeps its single shard in `shard_1.local.sqlite` (plus `-wal`/`-shm` siblings) in
the directory the app runs from:

```bash
find . -name 'shard_*.local.sqlite*' -not -path './vendor/*'
```

Tell the user these files hold local flow history that will be lost, and delete them once they agree; the engine
recreates the schema on the next start.

For `LAB` and `PROD`, tell the user the upgrade needs a maintenance window, which this skill cannot perform, and
point them to "Upgrading from v0.11.0" in the dwarf v0.12.0 release notes. In short:

- **Drain in-flight flows,** or `Terminate` them, before stopping the old replicas: a stored write to a reduced
  field is read differently now.
- **Use a fresh database per shard, or apply the schema changes by hand** while every replica is stopped: the new
  columns on `dwarf_flows`, `dwarf_steps` and `dwarf_peers` (including
  `ALTER TABLE dwarf_peers ADD COLUMN ordering_seq BIGINT NOT NULL DEFAULT 0`, or `ADD ordering_seq ...` with no
  `COLUMN` on SQL Server), the `dwarf_flows.step_id` rename to `stopped_step_id`, the `subgraph_error` JSON
  column, the rebuilt `idx_dwarf_steps_due`, and the PostgreSQL `TIMESTAMP` columns.
- **Bring old priorities into range** with `UPDATE dwarf_flows SET priority=3 WHERE priority>5` and the same on
  `dwarf_steps`, unless `MaxPriority` covers them.
- **Deploy every microservice and every foreman replica together.** Task microservices decode and re-encode the
  flow, whose JSON changed, so old and new must not run side by side.
- **Expect the foreman's memory to grow.** The engine's state cache defaults to a tenth of the memory limit
  (`GOMEMLIMIT`, else the cgroup limit, else 1 GiB); setting `GOMEMLIMIT` bounds it.

#### 3k. Check PostgreSQL Session Time Zones (Ask the User)

v1.48.0 moves sequel to v1.15.0, which starts every PostgreSQL and CockroachDB session in UTC, whatever the server,
database or role default, unless the data source name (or `PGTZ`/`PGOPTIONS`) sets a time zone. This affects any
microservice that opens such a database through sequel, not only the foreman.

```bash
grep -rnIE --exclude-dir=vendor --exclude-dir=.git --exclude-dir=.claude 'postgres(ql)?://|cockroachdb' .
```

If the project uses PostgreSQL or CockroachDB, tell the user. SQL that relied on a non-UTC session time zone, such as
`NOW()` stored into a `TIMESTAMP` column, `date_trunc`, or a `::date` cast, now computes in UTC. To keep the old zone,
add a `timezone` parameter to the data source name.

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
