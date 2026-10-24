---
name: add-python-task
description: TRIGGER when user asks to add a Python-backed workflow task to an existing Python-backed microservice.
---

**CRITICAL**: Read `.claude/rules/python.txt` and `.claude/rules/workflows.txt` before proceeding. The former covers the Go-Python bridge; the latter covers task lifecycle, flow state, retries, and OnTimeout/OnError routing.

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

## Workflow

Copy this checklist and track your progress:

```
Adding a Python-backed workflow task:
- [ ] Step 1: Verify the microservice is Python-backed
- [ ] Step 2: Determine the task signature
- [ ] Step 3: Run add-task with Manual + Tags("python")
- [ ] Step 4: Replace the task body with the Call+Poll durability pattern
- [ ] Step 5: Add the Python function to service.py
- [ ] Step 6: Housekeeping
```

#### Step 1: Verify the Microservice Is Python-Backed

The microservice must already have `python.go` and a `service.py` at its root. If not, run `add-python-microservice` first (it works both for fresh microservices and for extending an existing one without overwriting business logic).

#### Step 2: Determine the Task Signature

Determine the Go task signature. Task signatures include `flow *workflow.Flow` after `ctx` but before the typed inputs. Outputs that share a name with an input must use the `Out` suffix on the output side (read-modify-write pattern).

```go
func MyTask(ctx context.Context, flow *workflow.Flow, input1 string, input2 int) (output1 float64, err error)
```

Python sees the inputs as a dict keyed by the field names in `MyTaskIn`. Python does not see `ctx` or `flow`.

#### Step 3: Run `add-task` with `Manual` + `Tags("python")`

Run the `add-task` skill, with these overrides applied as you reach each of its steps:

- **Step 7 (Declare the Task in definition.go)**: add `Manual: true` and `Tags: []string{"python"}` to the `define.Task` var, so the task is registered as a manual subscription tagged `python` and stays off the bus until the venv liveness callback activates the `python`-tagged group when Python is ready:

  ```go
  var MyTask = define.Task{ // MARKER: MyTask
      Host: Hostname, Method: "POST", Route: ":428/my-task",
      Manual: true, Tags: []string{"python"},
      In: MyTaskIn{}, Out: MyTaskOut{},
  }
  ```

- **Step 8 (Generate the Boilerplate)**: run normally. `genservice` emits the `sub.Manual()` and `sub.Tag("python")` wiring into the generated `intermediate.go`, and scaffolds the task stub in `service.go` and the test in `service_test.go`.
- **Step 9 (Implement the Logic in service.go)**: skip. Step 4 below replaces the generated stub's body.
- **Step 10 (Test the Task)**: run normally, then add a one-line opt-in HINT immediately after the `app.RunInTest(t)` line so a future reader can switch the test from mock-only to real-Python without hunting through docs:

  ```go
  app.RunInTest(t)

  // HINT: Uncomment to spin up real Python and exercise actual execution (slow on first run)
  // svc.StartPyVenv(ctx)
  ```

- **Step 11 (Housekeeping)**: skip; housekeeping runs once at the end of this skill instead.

When `add-task` finishes, return here for Step 4.

#### Step 4: Replace the Task Body with the Call+Poll Durability Pattern

In `service.go`, replace the task body that `add-task` left as a stub with the durable Call+Poll pattern. The callID is persisted in flow state under the key `pyCallID`. `Poll` waits for most of the task's remaining time budget; if the call is still running when that wait ends, the task arms an unlimited `flow.Retry` and the next dispatch polls the same callID rather than issuing a fresh call. This lets a Python computation that exceeds the framework's 15-minute hop ceiling survive across task retries.

```go
func (svc *Service) MyTask(ctx context.Context, flow *workflow.Flow, input1 string, input2 int) (output1 float64, err error) { // MARKER: MyTask
    if svc.venv == nil || !svc.venv.Ready() {
        return 0, errors.New("venv not ready", http.StatusServiceUnavailable)
    }

    callID := flow.GetString("pyCallID")
    if callID == "" {
        in := myserviceapi.MyTaskIn{
            Input1: input1,
            Input2: input2,
        }
        callID, err = svc.venv.Call(ctx, "my_task", in)
        if err != nil {
            return 0, errors.Trace(err)
        }
        flow.SetString("pyCallID", callID)
    }

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
}
```

How the pattern composes:

- **First entry**: `pyCallID` is empty, so the task issues `Call`, persists the returned callID, then polls. If the call completes within the wait, the result is returned and `pyCallID` is deleted.
- **Retry entry**: `Poll` waits for 4/5 of the task's remaining time budget and returns `done=false` if the call is still running, leaving the rest of the budget to arm the retry and return. The `pyCallID` it set is kept, and the next dispatch skips `Call` and polls again, which completes immediately if Python has finished.
- **Terminal error**: an error discards the writes of the run that returns it, but a `pyCallID` set by an earlier run remains in the state. A workflow that routes the failure to an `onError` handler and carries on should `flow.Del("pyCallID")` there.

#### Step 5: Add the Python Function to `service.py`

Append a Python function to `service.py` (at the microservice's root) whose name is the snake_case form of the Go task name, accepting a dict and returning a dict. The docstring is the same description text as the task's godoc on the Go side.

```python
def my_task(args):  # MARKER: MyTask
    """MyTask does X."""
    input1 = args["input1"]
    input2 = args["input2"]
    # ... compute ...
    return {"output1": 42.0}
```

#### Step 6: Housekeeping

Follow the `housekeeping` skill.
