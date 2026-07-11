---
name: add-inbound-event
description: TRIGGER when user asks to listen for, subscribe to, or handle an event emitted by another microservice.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: An inbound event sink is declared as a `define.InboundEvent` var in `myserviceapi/definition.go` referencing the source microservice's `OutboundEvent`, and implemented as a handler in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: OnMyEvent` comment on the `define.InboundEvent` var.

**CRITICAL**: Inbound event sinks are not exposed via OpenAPI. The connector's built-in `:888/openapi.json` handler enforces this filter automatically.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying an inbound event sink:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Locate the source outbound event and determine the signature
- [ ] Step 3: Determine a description
- [ ] Step 4: Declare the inbound event in definition.go
- [ ] Step 5: Generate the boilerplate
- [ ] Step 6: Implement the sink in service.go
- [ ] Step 7: Test the inbound event sink
- [ ] Step 8: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Locate the Source Outbound Event and Determine the Signature

Locate the `define.OutboundEvent` var in the api package of the microservice that is the source of the event. Note its package import path (e.g. `github.com/company/project/eventsource/eventsourceapi`) and its In/Out structs - the handler's signature is the source event's signature.

```go
func OnMyEvent(ctx context.Context, input1 string, input2 ThirdPartyStruct) (output1 map[string]MyStruct, err error)
```

#### Step 3: Determine a Description

Take the event's description from the source's `OutboundEvent` godoc. Use it for the godoc on this `define.InboundEvent` var.

#### Step 4: Declare the Inbound Event in `definition.go`

Append the `define.InboundEvent` var to `myserviceapi/definition.go`, referencing the source's `OutboundEvent` var. Add the import for the source api package.

```go
import (
	"github.com/microbus-io/fabric/define"

	"github.com/company/project/eventsource/eventsourceapi"
)

/*
OnMyEvent is triggered when X.
*/
var OnMyEvent = define.InboundEvent{ // MARKER: OnMyEvent
	Source: eventsourceapi.OnMyEvent,
}
```

- The var name is the handler method name generated on this service. Name it after the source event. If this service sinks two same-named events from different sources, give the vars distinct names (e.g. `OnAUpdated`, `OnBUpdated`) so the handler methods differ
- `Source` is the typed reference to the source's `OutboundEvent` var, so a renamed or removed source event becomes a compile error here
- Add the gating and routing fields only when needed:
  - `LoadBalancing: define.None` to process the event on every replica of this service (broadcast, e.g. cache invalidation), or `LoadBalancing: "my-queue"` for a named queue; omit for the default queue (one replica processes each delivery)
  - `RequiredClaims: "roles.user"` to process the event only when its carried actor satisfies the expression; omit to accept all
  - `TimeBudget: 30 * time.Second` to cap the handler's duration; add the `"time"` import if used

#### Step 5: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `intermediate.go` (the hook wiring and `ToDo` entry), `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then verify the microservice compiles with `go vet ./...` from the project root.

#### Step 6: Implement the Sink in `service.go`

The previous step generated a placeholder sink handler `func (svc *Service) OnMyEvent(...)` in `service.go`, with its signature (the source event's signature, complex types referenced through the source api package) and godoc projected from `definition.go`, tagged `// MARKER: OnMyEvent` and holding a `// TODO` body. Fill in that body; leave the generated signature and godoc as they are. Add imports for any packages the body references that are not already present.

#### Step 7: Test the Inbound Event Sink

Skip this step if instructed to be "quick" or to skip tests.

The boilerplate generator created a placeholder test function `TestMyService_OnMyEvent` in `service_test.go`, tagged with a `// MARKER: OnMyEvent` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 8: Housekeeping

Follow the `housekeeping` skill.
