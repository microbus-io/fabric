## Create CreditFlow Microservice

Create an example microservice at hostname `creditflow.example` that demonstrates agentic workflow features of the
Microbus framework: fan-out/fan-in, forEach with a reducer, a subgraph run from a task, goto loops, error transitions,
and time budgets. Keep it a clean illustration of a real workflow shape, with no fault-injection inputs or test-only
branches.

## Applicant Type

Define an `Applicant` struct in the `creditflowapi` package with fields `ApplicantName string`, `SSN string`,
`Address string`, `Phone string`, `Employers []string`, and `CreditScore int` (all with camelCase JSON tags,
`omitzero`).

## CreditApproval Workflow Graph

The main workflow at `:428/credit-approval` accepts input `applicant Applicant` and produces outputs `approved bool`,
`creditVerified bool`, `employmentFailures int`, `identityVerified bool`.

Graph structure:

```
SubmitCreditApplication -> (fan-out)
  |- VerifyCredit -------------------------------> ReviewJoin
  |    `- (onErr) HandleCreditError -------------> ReviewJoin
  |- VerifyEmployment (forEach employers) -------> ReviewJoin
  `- IdentityVerification (RunIdentityVerification) -> ReviewJoin
                                                         |
                                     ReviewJoin -> ReviewCredit -> Decision -> END
                                              \       ^   |
                                        (goto) v      |   v (goto)
                                             RequestMoreInfo
```

Construction details:
- `ReviewJoin` and `ReviewCredit` are two graph nodes sharing the `ReviewCredit` task URL. `ReviewJoin` is the fan-in
  (`graph.SetFanIn("ReviewJoin")`) that closes the fan-out; `ReviewCredit` is the plain node that `RequestMoreInfo`
  loops back to. A fan-in must have a cohort to close, so the loop target cannot also be the fan-in.
- The `IdentityVerification` node's URL is the `RunIdentityVerification` task, which runs the `IdentityVerification`
  workflow as a subgraph.
- `graph.AddTransitionOnError("VerifyCredit", "HandleCreditError")` routes errors from VerifyCredit to
  HandleCreditError, which then transitions to `ReviewJoin`.
- `graph.AddTransitionForEach("SubmitCreditApplication", "VerifyEmployment", "employers", "employerName")` creates
  one VerifyEmployment invocation per element of the `employers` array, binding each element to `employerName`.
- `graph.SetReducer("employmentFailures", workflow.ReducerAdd)` sums the `employmentFailures` values from each
  forEach branch at the fan-in.
- `graph.AddTransitionGoto("ReviewJoin", "RequestMoreInfo")` and `graph.AddTransitionGoto("ReviewCredit",
  "RequestMoreInfo")` declare that either review node may dynamically goto RequestMoreInfo, and
  `graph.AddTransition("RequestMoreInfo", "ReviewCredit")` closes the loop. Node names are PascalCase;
  `employers`/`employerName` are camelCase state fields.

## IdentityVerification Workflow Graph (Subgraph)

The workflow at `:428/identity-verification` accepts inputs `applicantName`, `ssn`, `address`, `phone` and produces
output `identityVerified bool`.

```
InitIdentityVerification -> (fan-out)
  |- VerifySSN
  |- VerifyAddress
  `- VerifyPhoneNumber  [time budget: 1s]
       | (fan-in)
       v
  IdentityDecision -> END
```

`IdentityDecision` is the fan-in. Declare a 1-second time budget on the VerifyPhoneNumber task with
`TimeBudget: time.Second` in its `define.Task`, so the task is cancelled if it exceeds the deadline.

## Task Endpoints

All task endpoints are on port `:428`. Each receives and returns typed state fields via function arguments and named
return values.

- **SubmitCreditApplication** `:428/submit-credit-application` - input `applicant Applicant`; unpacks it into
  individual state fields: returns `applicantName`, `ssn`, `address`, `phone`, `employers []string`,
  `creditScore int`.

- **VerifyCredit** `:428/verify-credit` - input `creditScore int`; returns `creditVerified bool`, which is
  `creditScore >= 550`.

- **HandleCreditError** `:428/handle-credit-error` - input `onErr *errors.TracedError`; returns
  `creditVerified bool`. Logs the error as a warning and returns `false`.

- **VerifyEmployment** `:428/verify-employment` - inputs `applicantName string`, `employerName string`; returns
  `employmentFailuresOut int` (JSON `employmentFailures`). Returns `1` if either argument is empty, otherwise `0`.

- **RunIdentityVerification** `:428/run-identity-verification` - inputs `applicantName`, `ssn`, `address`, `phone`;
  returns `identityVerified bool`. Calls `flow.Subgraph(creditflowapi.IdentityVerification.URL(),
  creditflowapi.IdentityVerificationIn{...}, &out)`. When it yields, returns `false` and the foreman parks the step
  while the child runs; on re-entry, returns `out.IdentityVerified`. Returns a traced error if the subgraph cannot
  run.

- **InitIdentityVerification** `:428/init-identity-verification` - inputs `applicantName`, `ssn`, `address`,
  `phone`; no outputs. The subgraph's entry point; the inputs are already in its state.

- **VerifySSN** `:428/verify-ssn` - input `ssn string`; returns `ssnVerified bool`. Validates pattern
  `^\d{3}-\d{2}-\d{4}$` and rejects if the last 4 digits are `0000`.

- **VerifyAddress** `:428/verify-address` - input `address string`; returns `addressVerified bool`. Passes if the
  address is non-empty and does not contain `"Nowhere"`.

- **VerifyPhoneNumber** `:428/verify-phone-number` - input `phone string`; returns `phoneVerified bool`. Validates
  pattern `^\d{3}-\d{3}-\d{4}$` or `^\(\d{3}\) \d{3}-\d{4}$`. Has the 1s time budget.

- **IdentityDecision** `:428/identity-decision` - inputs `ssnVerified`, `addressVerified`, `phoneVerified`; returns
  `identityVerified bool`. Returns `true` only if all three are true.

- **RequestMoreInfo** `:428/request-more-info` - input `reviewAttempts int`; returns `reviewAttemptsOut int` (JSON
  `reviewAttempts`, the read-modify-write pair of the input). Returns `reviewAttempts + 1`.

- **ReviewCredit** `:428/review-credit` - inputs `creditScore int`, `creditVerified bool`, `reviewAttempts int`;
  returns `creditVerifiedOut bool` (JSON `creditVerified`). Logic:
  - Score >= 650: pass `creditVerified` through unchanged
  - Score 580-649: approve (`return true`)
  - Score 550-579 and `reviewAttempts < 2`: call `flow.Goto("RequestMoreInfo")`, return the current value
  - Score 550-579 and `reviewAttempts >= 2`: approve
  - Score < 550: return unchanged (reject)

- **Decision** `:428/decision` - inputs `creditVerified bool`, `employmentFailures int`, `identityVerified bool`;
  returns `approved bool`, which is `creditVerified && employmentFailures == 0 && identityVerified`.

## Demo Web Endpoint

- `Demo` on `ANY /demo` - an interactive HTML page backed by `resources/demo.html`. On GET with no params, pre-fills
  defaults (name "Alice", SSN "123-45-6789", address "123 Main St", phone "555-123-4567", employer "Acme Corp",
  score 750). On POST, builds an `Applicant` from the form fields (employers comma-separated, trimmed) and calls a
  `runWorkflow` helper that:
  1. `foremanClient.Create(ctx, creditflowapi.CreditApproval.URL(), creditflowapi.CreditApprovalIn{...}, nil)`
     creates the flow, which starts it
  2. `foremanClient.AwaitAndParse(ctx, flowKey, &out)` waits for it to stop and parses the outputs into a
     `CreditApprovalOut`; the outcome's status is shown on the page
  3. Fetches the history (`foremanClient.History`) and the Mermaid diagram
     (`foremanClient.HistoryMermaid(ctx, "?flowKey="+flowKey+"&format=raw")`) in parallel using `svc.Parallel`,
     ignoring failures of either
  Renders the template with the status, the approval results, the step history, and the Mermaid diagram, or the
  error if the run failed.

A `flattenSteps` helper recursively flattens `[]foremanapi.FlowStep` into `[]demoStep` structs (with `TaskName`
trimmed to its last path segment, `Status`, `Changes` as a JSON string, and `Indent bool`), indenting subgraph steps
from `SubHistory`.

## Downstream Dependency

Imports `foremanapi` to run and inspect the demo's flow (`Create`, `AwaitAndParse`, `History`, `HistoryMermaid`).
