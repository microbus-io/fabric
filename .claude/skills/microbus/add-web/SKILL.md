---
name: add-web
description: TRIGGER when user asks to add or modify a web handler, HTML page, file download, or raw HTTP endpoint that works with http.ResponseWriter/http.Request.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A web endpoint is declared as a `define.Web` var in `myserviceapi/definition.go` and implemented as a handler in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyWeb` comment on the `define.Web` var.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a web endpoint:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the method and route
- [ ] Step 3: Determine a description
- [ ] Step 4: Determine the required claims
- [ ] Step 5: Declare the endpoint in definition.go
- [ ] Step 6: Generate the boilerplate
- [ ] Step 7: Implement the logic in service.go
- [ ] Step 8: Test the handler
- [ ] Step 9: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Determine the Method and Route

The method of the endpoint determines the HTTP method with which it will be addressable. Use `ANY` to accept requests with any method.

The route of the endpoint is resolved relative to the hostname of the microservice to determine how it is addressed. The common approach is to use the name of the endpoint in kebab-case as its route, e.g. `/my-web`.

To set a port other than the default 443, prefix the route with the port, e.g. `:123/my-web`.

Encase path arguments with `{}` , e.g. `/section/{section}/page/{page...}`.

Prefix the route with `//` to set a hostname other than that of this microservice, e.g. `//another.host.name:1234/on-something`

#### Step 3: Determine a Description

Describe the endpoint starting with its name, in Go doc style: `MyWeb does X`. This becomes the godoc comment on the `define.Web` var.

Describe **what the endpoint does and the effect it produces**, not who is expected to call it. `"Renders a printable summary of the order"` is good; `"page used by the checkout flow"` is not.

#### Step 4: Determine the Required Claims

Determine if the endpoint should be restricted to authorized actors only. Compose a boolean expression over the JWT claims associated with the request that if not met will cause the request to be denied. For example: `roles.manager && level>2`. Default to closed: in a standard ingress configuration an empty `requiredClaims` on a `:443` endpoint (or any port the operator added to `AllowedInternalPorts`) is reachable by the entire internet. Leave it empty only for an intentionally public endpoint; if the endpoint wields a stored secret or a privileged side effect, it must be gated by `requiredClaims` and/or an internal port. See the Ports and Authentication sections of `.claude/rules/microbus.md`.

#### Step 5: Declare the Endpoint in `definition.go`

Append the `define.Web` var to `myserviceapi/definition.go`. A web endpoint has no In/Out structs.

```go
/*
MyWeb does X.
*/
var MyWeb = define.Web{ // MARKER: MyWeb
	Host: Hostname, Method: "ANY", Route: "/my-web",
}
```

- `Host` is always `Hostname`. `Method` and `Route` come from Step 2
- Add the gating and routing fields only when needed:
  - `RequiredClaims: "roles.manager && level>2"` for the claims from Step 4 (omit when public)
  - `TimeBudget: 30 * time.Second` to cap the handler's duration (omit for the default; add the `"time"` import if used)
  - `LoadBalancing: define.None` to multicast to all replicas, or `LoadBalancing: "my-queue"` for a named queue; omit for the default hostname queue

#### Step 6: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `myserviceapi/client.go`, `intermediate.go`, `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

The generated client method's arity depends on the method, which matters when writing the test in Step 8:
- A body method (`POST`, `PUT`, `PATCH`): `MyWeb(ctx, relativeURL string, body any)`
- A non-body method (`GET`, `HEAD`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`): `MyWeb(ctx, relativeURL string)`
- `ANY`: `MyWeb(ctx, method string, relativeURL string, body any)`

Then verify the microservice compiles with `go vet ./...` from the project root.

#### Step 7: Implement the Logic in `service.go`

The previous step generated a placeholder web handler `func (svc *Service) MyWeb(w http.ResponseWriter, r *http.Request) (err error)` in `service.go`, tagged `// MARKER: MyWeb` and holding a `// TODO` body. Fill in that body; leave the generated signature and godoc as they are. Use `r.PathValue("argName")` to obtain path argument values by name, if needed.

#### Step 8: Test the Handler

Skip this step if instructed to be "quick" or to skip tests.

The boilerplate generator created a placeholder test function `TestMyService_MyWeb` in `service_test.go`, tagged with a `// MARKER: MyWeb` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 9: Housekeeping

Follow the `housekeeping` skill.
