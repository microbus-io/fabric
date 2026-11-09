# creditflow.example

## Agent Instructions

This microservice implements agentic workflows. See `.claude/rules/workflows.txt` for the conventions.

## Overview

The creditflow example implements a multi-step credit approval workflow that exercises the core workflow features:
fan-out/fan-in, forEach with a reducer, a subgraph run from a task, goto loops, error transitions, and time budgets.

## Workflows

### CreditApproval

The main workflow. Accepts an `Applicant` and produces `approved`, along with the `creditVerified`,
`employmentFailures`, and `identityVerified` results it was decided from.

```
SubmitCreditApplication -> (fan-out)
  |- VerifyCredit ----------------------------> ReviewJoin
  |    `- (on err) HandleCreditError ---------> ReviewJoin
  |- VerifyEmployment (forEach employer) -----> ReviewJoin
  `- RunIdentityVerification (flow.Subgraph) -> ReviewJoin
                                                   |
                                ReviewJoin -> ReviewCredit -> Decision -> END
                                         \       ^   |
                                   (goto) v      |   v (goto)
                                        RequestMoreInfo
```

- `SubmitCreditApplication` unpacks the applicant into individual state fields
- `VerifyCredit` checks credit score >= 550. On error, routes to `HandleCreditError`
- `HandleCreditError` receives the `TracedError` via `onErr`, sets `creditVerified = false`
- `VerifyEmployment` runs once per employer via forEach; failures summed with `ReducerAdd`
- `RunIdentityVerification` calls `flow.Subgraph` to run the `IdentityVerification` child workflow (see below) and
  adopts its `identityVerified` output
- `ReviewJoin` and `ReviewCredit` are two graph nodes sharing the `ReviewCredit` task. `ReviewJoin` is the fan-in
  that closes the fan-out; `ReviewCredit` is where `RequestMoreInfo` loops back to. The split exists because a fan-in
  needs a cohort to close and a re-entry from the loop has none, so the loop target cannot be the fan-in. llm.core's
  `ChatLoop` splits `FirstLLM`/`NextLLM` for the same reason.
- The review passes through for good scores (650+), approves borderline (580-649), uses goto to `RequestMoreInfo`
  for very borderline (550-579) until two attempts are made and then approves, and rejects below 550
- `Decision` ANDs all verification results

### IdentityVerification (subgraph)

```
InitIdentityVerification -> (fan-out)
  |- VerifySSN
  |- VerifyAddress
  `- VerifyPhoneNumber (1s time budget)
       | (fan-in)
       v
  IdentityDecision -> END
```

## Why no fault injection

creditflow deliberately has no fault-injection hooks (no `faultInjection`-style field that tasks inspect to force error/retry/sleep/interrupt/timeout/goto behaviors). Those workflow mechanisms are stress-tested in isolation by the dwarf workflow engine's own test suite, so this example stays a clean illustration of a *real* workflow shape, without test-only branches.
