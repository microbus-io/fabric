---
name: add-workflow
description: TRIGGER when user asks to define a workflow graph, create an agent or agentic workflow, orchestrate tasks, or build a multi-step process. "Agent", "agentic workflow", and "workflow" are interchangeable here - an LLM is not required for a workflow to be an agent (rule-based, deterministic, and LLM-driven workflows are all agents).
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A workflow is declared as a `define.Workflow` var in `myserviceapi/definition.go` and implemented as a graph builder in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyWorkflow` comment on the `define.Workflow` var and on its In/Out structs.

**IMPORTANT**: Read `.claude/rules/workflows.txt` for workflow and task conventions before proceeding.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a workflow graph:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the signature
- [ ] Step 3: Determine the route
- [ ] Step 4: Determine a description
- [ ] Step 5: Declare the workflow in definition.go
- [ ] Step 6: Generate the boilerplate
- [ ] Step 7: Implement the graph builder in service.go
- [ ] Step 8: Test the workflow
- [ ] Step 9: Housekeeping
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

Determine the input and output fields of the workflow. Inputs are the state fields the workflow expects from its caller. Outputs are the state fields the workflow produces as its result. The signature is documentation only - it describes the workflow's expected state contract but does not generate typed code. The actual state is `map[string]any`.

```go
MyWorkflow(inputField1 string, inputField2 float64) (outputField1 bool, outputField2 int)
```

Constraints:
- The name `status` is reserved
- All fields must be serializable into JSON
- Field names must start with a lowercase letter
- The workflow name must start with an uppercase letter

#### Step 3: Determine the Route

The route of the workflow graph endpoint is resolved relative to the hostname of the microservice. Workflows use the dedicated port `:428` to prevent external access. Use the name of the workflow in kebab-case as its route, e.g. `:428/my-workflow`. The method is `GET`.

#### Step 4: Determine a Description

Describe the workflow starting with its name, in Go doc style: `MyWorkflow does X`. This becomes the godoc comment on the `define.Workflow` var.

Describe **what the workflow does and the outcome it produces**, not who is expected to run it. `"Underwrites a credit application: gathers identity, employment and credit history, then approves or declines"` is good; `"called by the customer portal"` or `"used by the LLM as a tool"` is not.

#### Step 5: Declare the Workflow in `definition.go`

Append the `define.Workflow` var and its In/Out structs to `myserviceapi/definition.go`. Workflows always use the `GET` method.

```go
/*
MyWorkflow does X.
*/
var MyWorkflow = define.Workflow{ // MARKER: MyWorkflow
	Host: Hostname, Method: "GET", Route: ":428/my-workflow",
	In: MyWorkflowIn{}, Out: MyWorkflowOut{},
}

// MyWorkflowIn are the input arguments of MyWorkflow.
type MyWorkflowIn struct { // MARKER: MyWorkflow
	InputField1 string  `json:"inputField1,omitzero"`
	InputField2 float64 `json:"inputField2,omitzero"`
}

// MyWorkflowOut are the output arguments of MyWorkflow.
type MyWorkflowOut struct { // MARKER: MyWorkflow
	OutputField1 bool `json:"outputField1,omitzero"`
	OutputField2 int  `json:"outputField2,omitzero"`
}
```

- `Host` is always `Hostname`. `Method` is always `GET`. `Route` comes from Step 3
- The In/Out structs are documentation (OpenAPI, the runner UI, the generated Executor signature), not runtime contracts; state flows through the graph unfiltered
- When a workflow both reads and returns the same state field (a read-modify-write field, e.g. a `counter` it increments), give the Out struct field the `Out` suffix and strip that suffix from the JSON tag so it maps to the same state key as the input (e.g. `CounterOut int` with `json:"counter,omitzero"`), exactly as a task does in `add-task`. Without the suffix the input and output share a name and the generated `Executor`/`Subgraph` signature collides (`counter redeclared in this block`) in a file you must not hand-edit
- If an In/Out field's type comes from another package (e.g. a `time.Time` field needs `"time"`), add that import to `definition.go`
- Add `RequiredClaims: "..."` only when the workflow must be gated; omit when open

#### Step 6: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `myserviceapi/client.go` (the `Executor` and `Subgraph` methods), `intermediate.go` (the marshaler that validates the graph, the `ToDo` entry, and the subscription), `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then, from the project root, bring the module's dependencies up to date and verify the microservice compiles:

```shell
go mod tidy
go vet ./...
```

Run `go mod tidy` first: the workflow test imports the foreman, which pulls transitive dependencies (the dwarf engine, sequel, throttle) that may not yet be in `go.sum`, which makes `go vet` fail with `missing go.sum entry` until the module is tidied.

#### Step 7: Implement the Graph Builder in `service.go`

The previous step generated a placeholder graph builder `func (svc *Service) MyWorkflow(ctx context.Context) (graph *workflow.Graph, err error)` in `service.go`, tagged `// MARKER: MyWorkflow`, with a `graph = workflow.NewGraph("MyWorkflow")` starter that returns an empty graph (which fails validation until you define it). Build out the graph with the `workflow.NewGraph` builder API, referencing task endpoints from this or other microservices by their `URL()` method. Leave the generated godoc as it is.

Each node in the graph carries both a short **name** (used in transitions) and a **URL** (used to dispatch the task at runtime). Register nodes with `graph.SetEndpoint("Name", taskURL)`, then write transitions in terms of names. Use **PascalCase** node names matching the task endpoint (e.g. `"TaskA"` for `TaskA`) - they are graph-topology identifiers, kept visually distinct from the lowercased dispatch URLs and the camelCase state field names. The graph display name passed to `NewGraph` is PascalCase too (conventionally the workflow's own name). Child workflows are not graph nodes; they are launched dynamically from a task body via `flow.Subgraph(childWorkflow.URL(), input)` - see the "Subgraphs and Interrupts" section in `.claude/rules/workflows.txt`.

```go
// MyWorkflow does X.
func (svc *Service) MyWorkflow(ctx context.Context) (graph *workflow.Graph, err error) { // MARKER: MyWorkflow
	graph = workflow.NewGraph("MyWorkflow")
	graph.SetEndpoint("TaskA", myserviceapi.TaskA.URL())
	graph.SetEndpoint("TaskB", myserviceapi.TaskB.URL())
	graph.SetEndpoint("TaskC", myserviceapi.TaskC.URL())
	// graph.AddTransition("TaskA", "TaskB")
	// graph.AddTransitionSwitch("TaskB", "TaskC", "score >= 50") // first-match-wins routing; end with a "true" arm for the default
	// graph.AddTransitionSwitch("TaskB", workflow.END, "true")
	// graph.AddTransitionGoto("TaskB", "TaskC")
	return graph, nil
}
```

If a subgraph call needs its input or output adapted across a contract boundary, do it in any ordinary task immediately upstream or downstream of the subgraph using `flow.Transform`/`Delete`/`Clear` - see the "State Transformation Around a Subgraph" section in `.claude/rules/workflows.txt`. If the workflow's terminal state needs scrubbing before it lands in `final_state`, the last task calls `flow.Delete`/`Transform`.

Naming the same task URL twice with different names is the supported way to reuse a task at multiple positions in the graph (each position keeps its own node identity for fan-in tracking).

**Every fan-out requires a matching fan-in.** Any node with two or more normal outgoing transitions, or an `AddTransitionForEach` transition, is a fan-out: its branches run in parallel. Every fan-out must converge on a single node that you mark with `graph.SetFanIn("name")`. Add the `SetFanIn` call in the same edit as the fan-out transitions, and route every parallel branch into the marked node before the graph reaches `workflow.END`. A graph that fans out without a `SetFanIn` node still compiles and passes `go vet`, but `graph.Validate()` (run by the generated marshaler) rejects it at run time, so the test in Step 8 fails. This is the single most common workflow-graph mistake. The fan-out/fan-in section in `.claude/rules/workflows.txt` has the full rule and a worked example.

**Reducers for fan-in fields.** When parallel branches converge, each field that needs anything other than last-write-wins must be explicitly wired with `graph.SetReducer(field, reducer)` at graph-build time. Fields without a registered reducer use `Replace` (last write wins). No name-driven inference - a field named `messages` and a field named `total` are treated identically until `SetReducer` says otherwise.

```go
graph.SetReducer("messages", workflow.ReducerAppend) // accumulate per-branch deltas
graph.SetReducer("total",    workflow.ReducerAdd)    // sum numeric contributions
graph.SetReducer("seen",     workflow.ReducerUnion)  // dedupe across branches
graph.SetReducer("attrs",    workflow.ReducerMerge)  // merge per-branch objects
graph.SetReducer("approved", workflow.ReducerAnd)    // all branches must approve
graph.SetReducer("flagged",  workflow.ReducerOr)     // any branch flags
graph.SetReducer("notes",    workflow.ReducerConcat) // join string deltas
```

Pick the reducer by what the merge means, not by what the field is named. The default `Replace` is correct when only one branch ever writes the field.

**Prefer `AddTransitionSwitch` > `AddTransitionWhen` > `AddTransitionGoto` for branching.** `AddTransitionSwitch` is the default for any routing where exactly one branch should run: siblings are first-match-wins in declaration order (use `when="true"` as the last arm for a default), the validator enforces mutual exclusivity, and no `SetFanIn` is needed because only one branch ever fires. Drop to `AddTransitionWhen` only when the source genuinely fans out under multiple independent predicates that may all match - that's parallel work, not routing, and it needs a matching `SetFanIn`. Two `When` predicates meant to be mutually exclusive (`x>0` and `x<=0`) should be a `Switch` so the validator enforces it. Reserve `AddTransitionGoto` for runtime control-flow the task body decides via `flow.Goto`, the canonical case being a fan-in node looping back ("ask for more info, retry the review"). The "Switch Transitions" and "When to choose Switch vs. When vs. Goto" sections in `.claude/rules/workflows.txt` carry the full rules.

To embed this workflow as a subgraph from inside another task body, use the generated `Subgraph` client (`otherapi.NewSubgraph(flow).MyWorkflow(ctx, ...)`), never the `Executor` - the `Executor` is test-only.

#### Step 8: Test the Workflow

Skip this step if instructed to be "quick" or to skip tests.

The boilerplate generator created a placeholder test function `TestMyService_MyWorkflow` in `service_test.go`, tagged with a `// MARKER: MyWorkflow` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 9: Housekeeping

Follow the `housekeeping` skill.
