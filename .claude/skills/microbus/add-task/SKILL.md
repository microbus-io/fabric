---
name: add-task
description: TRIGGER when user asks to add a workflow step, agent step, agentic task, task endpoint, or workflow phase. "Agent step" and "workflow step" are interchangeable - a task is a single step of a workflow / agent, whether or not the surrounding workflow calls an LLM.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A task is declared as a `define.Task` var in `myserviceapi/definition.go` and implemented as a handler in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyTask` comment on the `define.Task` var and on its In/Out structs.

**IMPORTANT**: Read `.claude/rules/workflows.txt` for workflow and task conventions before proceeding.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a task endpoint:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the signature
- [ ] Step 3: Determine the route
- [ ] Step 4: Determine a description
- [ ] Step 5: Determine the required claims
- [ ] Step 6: Define complex types
- [ ] Step 7: Declare the task in definition.go
- [ ] Step 8: Generate the boilerplate
- [ ] Step 9: Implement the logic in service.go
- [ ] Step 10: Test the task
- [ ] Step 11: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

Ensure the local `CLAUDE.md` advertises that this microservice implements agentic workflows. The Agent Instructions block holds one short paragraph per instruction so multiple instructions (workflows, SQL, auth) can coexist as separate paragraphs. The paragraph to add is:

```markdown
This microservice implements agentic workflows. See `.claude/rules/workflows.txt` for the conventions.
```

How to apply:

- If the file does not exist, create it with the hostname as an H1 heading, then add an `## Agent Instructions` section containing the paragraph above.
- If the file exists and already has an `## Agent Instructions` section, append the paragraph (separated by a blank line) after the existing instructions; skip if a workflows-related paragraph is already present.
- If the file exists but has no `## Agent Instructions` section, insert one as the first section after the H1 hostname heading and add the paragraph.

#### Step 2: Determine the Signature

Determine the Go signature of the task endpoint. A task always receives `ctx context.Context` and `flow *workflow.Flow` as its first two arguments, followed by state fields it reads as input. It returns state fields it writes as output, plus `err error`.

```go
func MyTask(ctx context.Context, flow *workflow.Flow, input1 string, input2 float64) (output1 bool, err error)
```

Constraints:
- The first argument must be `ctx context.Context`
- The second argument must be `flow *workflow.Flow`
- The function must return an `err error`
- All input arguments (after `flow`) represent state fields read from the workflow state
- All output arguments (except `err`) represent state fields written to the workflow state
- To read and modify the same state field, use the `Out` suffix on the return value - the generator strips `Out` to map back to the same state key (e.g. input `counter int` and output `counterOut int` both map to state key `"counter"`)
- Complex types (structs) are allowed by value or by reference
- All arguments must be serializable into JSON
- Arguments must not be named `t` or `svc`
- Argument names must start with a lowercase letter
- The function name must start with an uppercase letter

Naming for fan-in: argument names carry no execution semantics. A fan-in field's reducer is set explicitly at graph-build time with `graph.SetReducer(field, reducer)`; without one, the default is `Replace` (last write wins). Pick names that describe what the field *is* (e.g. `failures`, `messages`, `score`), not the merge strategy. Tasks writing to a reducer-managed field must produce only the **delta** for this branch, not the full accumulated value - otherwise fan-in produces duplicates. For example, a `VerifyEmployment` task running once per employer with `graph.SetReducer("failures", workflow.ReducerAdd)` should return `failuresOut: 0 or 1` (its own count), not the running total.

**Prefer typed input/output arguments over `flow.Get` / `flow.Set`.** Inputs and outputs are auto-bound to state by name; the signature is the task's state contract, mocks get typed handlers, and a reader sees what the task reads and produces without scanning the body. Reserve `flow.Get` / `flow.Set` for keys whose names are dynamic or for internal types not in the API package. See the Best Practices section of `.claude/rules/workflows.txt` for the rationale.

**`forEach` branches see auto-injected per-element fields.** When the task runs as the target of `AddTransitionForEach(..., "items", "item")`, the branch's state contains `item` (the element), `itemIndex` (0-based position), and `itemCount` (cohort size). Take any of them as a typed argument by name - no lookup code needed.

#### Step 3: Determine the Route

The route of the task endpoint is resolved relative to the hostname of the microservice. Tasks use the dedicated port `:428` to prevent external access. Use the name of the task in kebab-case as its route, e.g. `:428/my-task`.

#### Step 4: Determine a Description

Describe the task starting with its name, in Go doc style: `MyTask does X`. This becomes the godoc comment on the `define.Task` var.

Describe **what the task does and the effect it produces**, not who or what is expected to invoke it. `"Computes the credit score from the applicant's history"` is good; `"called by the credit-review workflow"` or `"used by the LLM as a tool"` is not.

#### Step 5: Determine the Required Claims

Determine if the task endpoint should be restricted to authorized actors only. Compose a boolean expression over the JWT claims associated with the request that if not met will cause the request to be denied. For example: `roles.manager && level>2`. Leave empty if the task should be accessible by all.

#### Step 6: Define Complex Types

Identify the struct types in the signature and define each in the `myserviceapi` directory following the
`add-type` skill: one file per type, camelCase `json` tags with `omitzero`, `jsonschema_description` tags,
optional `dv8` validation tags, and an optional pure `Validate` method for cross-field invariants. A type
owned by another microservice is aliased rather than redefined. Skip this step if there are no complex types.

A task's inputs are read from the workflow's shared state, which is populated by the workflow's caller, by
LLMs, and by tasks possibly hosted in other microservices. `dv8` tags on the task's In struct are enforced
when the state is parsed; a violation fails the flow at this step rather than propagating invalid state.

#### Step 7: Declare the Task in `definition.go`

Append the `define.Task` var and its In/Out structs to `myserviceapi/definition.go`. Tasks always use the `POST` method.

```go
/*
MyTask does X.
*/
var MyTask = define.Task{ // MARKER: MyTask
	Host: Hostname, Method: "POST", Route: ":428/my-task",
	In: MyTaskIn{}, Out: MyTaskOut{},
}

// MyTaskIn are the input arguments of MyTask.
type MyTaskIn struct { // MARKER: MyTask
	Input1 string  `json:"input1,omitzero"`
	Input2 float64 `json:"input2,omitzero"`
}

// MyTaskOut are the output arguments of MyTask.
type MyTaskOut struct { // MARKER: MyTask
	Output1 bool `json:"output1,omitzero"`
}
```

- `Host` is always `Hostname`. `Method` is always `POST`. `Route` comes from Step 3
- The In struct holds the input arguments excluding `ctx` and `flow`; the Out struct holds the output arguments excluding `err`
- For an output field with the `Out` suffix, strip the suffix from the JSON tag so it maps to the same state key as the input (e.g. `CounterOut int` with `json:"counter,omitzero"`)
- If an In/Out field's type comes from another package (e.g. a `time.Time` field needs `"time"`), add that import to `definition.go`
- Add the gating fields only when needed:
  - `RequiredClaims: "roles.manager && level>2"` for the claims from Step 5 (omit when open)
  - `TimeBudget: 5 * time.Minute` if the task has a known runtime ceiling shorter than the foreman default (2m, hard ceiling 15m); add the `"time"` import. For work that does not fit within 15m, use the Interrupt-and-Resume or Polling-with-Retry patterns from `.claude/rules/workflows.txt`

#### Step 8: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `myserviceapi/client.go` (the `Executor` and `Subgraph` methods), `intermediate.go` (the marshaler, `ToDo` entry, and subscription), `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then, from the project root, bring the module's dependencies up to date and verify the microservice compiles:

```shell
go mod tidy
go vet ./...
```

Run `go mod tidy` first: a task that introduces a new import (a downstream client, or the foreman for subflows and workflow tests) can pull transitive dependencies that are not yet in `go.sum`, which makes `go vet` fail with `missing go.sum entry` until the module is tidied.

#### Step 9: Implement the Logic in `service.go`

The previous step generated a placeholder task `func (svc *Service) MyTask(ctx context.Context, flow *workflow.Flow, ...)` in `service.go`, with the signature and godoc projected from `definition.go`, tagged `// MARKER: MyTask` and holding a `// TODO` body. Fill in that body; leave the generated signature and godoc as they are. Complex types refer to their definition in `myserviceapi`.

The task receives state fields as input arguments and returns state fields as output. It also has access to `flow` for control operations (`flow.Goto()`, `flow.Interrupt()`, `flow.Subgraph()`, `flow.Retry()`, `flow.Sleep()`) and for field-based state access (`flow.GetString()`, `flow.Set()`) when needed. `Interrupt` and `Subgraph` both park the step and return `(data, yield, err)`; the task must `return nil` when `yield` is true and may read `data` / branch on `err` once it resolves (see "Subgraphs and Interrupts" in `.claude/rules/workflows.txt`).

To invoke another microservice's workflow as an isolated child flow (a subgraph) from inside this task body, use that microservice's generated `Subgraph` client (only the explicit inputs cross in, only the explicit outputs cross back). Do not use its `Executor` from a task body - the `Executor` is test-only.

```go
output1, yield, err := otherapi.NewSubgraph(flow).OtherWorkflow(ctx, input1, input2)
if yield || err != nil {
	return err
}
// use output1
```

**Idempotency.** Tasks may be replayed: `flow.Retry`, worker-death recovery, and Subgraph re-entry all re-run the task body from the top. A task that fires an external side effect (charge a card, send an email, write to a non-transactional store) must carry its own dedupe key or check first whether the effect has already happened. The framework does not deduplicate side effects for you. Pure computation over state needs no special treatment.

**State hygiene.** If this task consumes large intermediates (LLM response, parsed payload, raw API body, image bytes) that downstream tasks do not need, drop them before returning. Three primitives compose for any cleanup pattern:

- `flow.Delete(names...)` - drop the listed fields.
- `flow.Clear()` - drop every field; typical in a task that is about to build a fresh subgraph input from scratch.
- `flow.Transform("newKey", "oldKey", ...)` - clear all state, then re-introduce the listed fields under new names. Doubles as a "keep these" primitive when called with `("name", "name")` pairs.

Each records JSON null in the step's changes for dropped fields, so the cleanup is preserved in the audit trail; downstream merged state is absent the field (Replace reducer) or sees no contribution (Add/Append/Union/Merge/And/Or/Concat short-circuit to their identity when a branch's value is JSON null).

#### Step 10: Test the Task

Skip this step if instructed to be "quick" or to skip tests.

The boilerplate generator created a placeholder test function `TestMyService_MyTask` in `service_test.go`, tagged with a `// MARKER: MyTask` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 11: Housekeeping

Follow the `housekeeping` skill.
