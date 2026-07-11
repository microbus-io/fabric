---
name: add-metric
description: TRIGGER when user asks to add or modify a metric, counter, gauge, or histogram, or to track/measure an operation.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A metric is declared as a `define.Metric` var in `myserviceapi/definition.go`; its observe callback, if any, is implemented in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyMetric` comment on the `define.Metric` var.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a metric:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the kind
- [ ] Step 3: Determine if observable just in time
- [ ] Step 4: Determine the value type and labels
- [ ] Step 5: Determine the OpenTelemetry name
- [ ] Step 6: Declare the metric in definition.go
- [ ] Step 7: Record the metric
- [ ] Step 8: Generate the boilerplate
- [ ] Step 9: Test the callback
- [ ] Step 10: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Determine the Kind

Determine if the metric is a:
- **Counter** - an ever increasing number
- **Gauge** - a number that can go up or down
- **Histogram** - a distribution of cumulative values over time

If a histogram, determine the boundaries that determine its buckets.

#### Step 3: Determine If Observable Just in Time

For some metrics, and in particular gauges, it is enough to observe just in time before recording. For example, there's no reason to sample available system memory constantly if only the last sampling will be reported. Determine if this metric is observable.

#### Step 4: Determine the Value Type and Labels

The value type is the type of the number being recorded: `int`, `float64`, or `time.Duration`.

The labels are the dimensions the metric is broken down by, named like Go arguments (e.g. `status`, `method`). They are recorded as strings. A metric may have no labels.

#### Step 5: Determine the OpenTelemetry Name

The OpenTelemetry name of the metric:
- Should have a (single-word) application prefix relevant to the domain the metric belongs to. The prefix is usually the application name itself
- Should have a suffix describing the unit, in plural form

For example, `myapplication_my_metric_units`.

Do **not** add a `_total` suffix to a counter's OpenTelemetry name. `_total` is a Prometheus naming convention, not an OpenTelemetry one.

#### Step 6: Declare the Metric in `definition.go`

Append the `define.Metric` var to `myserviceapi/definition.go`. The godoc is the metric's description.

```go
/*
MyMetric counts X.
*/
var MyMetric = define.Metric{ // MARKER: MyMetric
	Kind: define.Counter, Value: int(0), Labels: []string{"status"},
	OTelName: "myapplication_my_metric_units",
}
```

- `Kind` is `define.Counter`, `define.Gauge`, or `define.Histogram`
- `Value` is a type carrier declaring the recorded value's type: `int(0)`, `float64(0)`, or `time.Duration(0)`. For `time.Duration`, add the `"time"` import
- `Labels` is the list of label argument names; omit when there are none
- `Buckets: []float64{0.1, 0.5, 1, 5}` is required for a histogram and omitted otherwise
- `OTelName` is the name from Step 5
- `Observable: true` for a metric observed just in time (Step 3); omit when false

#### Step 7: Record the Metric

The generated recorder is `IncrementMyMetric(ctx, value, labels...)` for a counter and `RecordMyMetric(ctx, value, labels...)` for a gauge or histogram.

If the metric is **observable just in time**, the boilerplate generator (Step 8) creates a placeholder `OnObserveMyMetric` handler in `service.go`, tagged `// MARKER: MyMetric` and holding a `// TODO` body. Fill in that body to record the current value, for example:

```go
v, err := svc.calculateMyMetric()
if err != nil {
	return errors.Trace(err)
}
err = svc.RecordMyMetric(ctx, v, "ok")
return errors.Trace(err)
```

Otherwise, call the recorder from the place in `service.go` where the event occurs.

```go
svc.IncrementMyMetric(ctx, 1, "ok")
```

#### Step 8: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `intermediate.go` (the recorder, the `Describe*` registration, and the observe dispatcher), `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then verify the microservice compiles with `go vet ./...` from the project root.

#### Step 9: Test the Callback

Skip this step if the metric is not observable just in time, or if instructed to be "quick" or to skip tests.

When present, the boilerplate generator created a placeholder test function `TestMyService_OnObserveMyMetric` in `service_test.go`, tagged with a `// MARKER: MyMetric` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 10: Housekeeping

Follow the `housekeeping` skill.
