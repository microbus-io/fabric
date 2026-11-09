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
SOURCE = v1.49.0
DEST   = v1.50.0
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

### Step 3: Release-Specific Migration (v1.49.0 -> v1.50.0)

*Invoked by Step 2. This is `DEST`'s migration; a release author replaces it when cutting the next release (see "Authoring framework upgrade skills" in the repo-root `CLAUDE.md`). Confine it to source edits - Step 2 owns the per-increment `genservice` + `go vet`, and Step 4 owns the final `go test`.*

v1.50.0 removes the `sequel` skills (`add-sql-microservice` with its `busstop` template, `add-sql-fields`,
`modify-sql-fields`, `remove-sql-fields`, `rename-sql-object` and `rename-sql-table`) and rewrites
`.claude/rules/sequel.txt` around how a SQL-backed microservice uses the sequel package and the patterns that work
well, instead of the template's layout. No source edit is required for that: a microservice scaffolded from the
template is the project's own code, and it keeps compiling and working unchanged.

v1.50.0 also stops the two synchronous tool paths from taking workflow tools: `llm.core`'s `Chat` rejects them, and
the MCP portal no longer lists workflows. A `Chat` call that passes a workflow tool needs rewriting (3c).

#### 3a. Tell the User

Tell the user that the sequel skills are gone, and that microservices built from them are now maintained by hand,
guided by the new `sequel.txt`. The skills were already removed from the project when this release's `.claude` was
installed.

Microservices built from the template have a `CLAUDE.md` whose Agent Instructions say "This microservice is a SQL
CRUD microservice. See `.claude/rules/sequel.txt` for the conventions." The new `sequel.txt` no longer describes the
template's conventions, so offer to reword that line, for example to "This microservice keeps its data in a SQL
database. See `.claude/rules/sequel.txt` for the recommended patterns.":

```bash
grep -rlF --include='CLAUDE.md' --exclude-dir=vendor 'This microservice is a SQL CRUD microservice.' .
```

#### 3b. Optionally Replace `sequel.UnsafeSQL` (Ask First)

Microservices built from the template mark SQL expressions in their column mappings with `sequel.UnsafeSQL`, which
sequel has deprecated: sequel itself never treated it specially, and only the template's own statement builder did.
It still compiles and works, so this step is optional. Find the microservices that use it:

```bash
grep -rlE --include='*.go' --exclude-dir=vendor '\bsequel\.UnsafeSQL\b' . | xargs -r -n1 dirname | sort -u
```

If there are any, ask the user whether to replace it with a private marker type in each of those packages. The
rename is safe only where the package that creates the marked values is also the one whose statement builder checks
for them, as in the template. If a project moved the builder into a shared package, the two would end up with
different types and the check would silently fail, storing the SQL as text; leave such a project on
`sequel.UnsafeSQL`. If yes, in each directory listed, rename it:

```bash
perl -pi -e 's/\bsequel\.UnsafeSQL\b/sqlExpr/g' DIR/*.go
```

Then declare the type once in that package, for example in `service.go`:

```go
// sqlExpr marks a column value that is SQL to write into the statement rather than a value to bind.
type sqlExpr string
```

Step 2's `go vet` reports an unused `sequel` import if the rename left one.

#### 3c. Find `Chat` Calls That Pass a Workflow Tool (Grep-Guided)

`llm.core`'s `Chat` now fails with a 400, `workflow tool requires ChatLoop`, when any of its `toolURLs` is a
workflow's URL. Such a call never ran the workflow (the LLM got the workflow's graph as the tool's result), so no
working behavior is lost, but the call compiles and fails only at run time, so `go vet` does not catch it. Find the
`Chat` calls:

```bash
grep -rlF --include='*.go' --exclude-dir=vendor 'coreservices/llm/llmapi"' . \
  | xargs -r grep -HnE '\.Chat\('
```

Skip a call with no tools (`nil` toolURLs). For each other call, trace its `toolURLs` argument to the URLs it holds.
A URL is a workflow's when:

- it comes from `xapi.Name.URL()` where `Name` is declared with `define.Workflow` in that api package. For a package
  outside the project (Fabric's own, or another module's), look it up in the module cache:

  ```bash
  grep -n 'define.Workflow' "$(go list -f '{{.Dir}}' IMPORT/PATH/xapi)/definition.go"
  ```

- or it is a literal or configured URL on port `:428`, the port workflows and tasks are served on (a task URL fails
  `Chat` anyway).

Rewrite each call that passes a workflow tool. The right shape is a design choice, so ask the user:

- **Called from a workflow task:** replace the `Chat` call with the `ChatLoop` workflow as a subgraph,
  `llmapi.NewSubgraph(flow).ChatLoop(ctx, provider, model, items, toolURLs, options)`, which runs workflow tools as
  subgraphs of its own. The task parks while the conversation runs (`yield`), and re-enters with the result; see
  "Using LLMs in Workflows" in `.claude/rules/workflows.txt`.
- **Called from a function or web handler:** a synchronous request cannot wait on a workflow. Either move the
  operation into a workflow that runs `ChatLoop`, launched by the code that handles the triggering event, or expose
  the logic the LLM needs as a function and pass that function's URL instead of the workflow's.

Tell the user, too, that the MCP portal no longer lists workflows as tools, so an MCP client that used one loses it.

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
