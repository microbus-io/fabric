---
name: add-function
description: TRIGGER when user asks to add, create, or modify an API endpoint, function, or RPC, or a route that accepts typed arguments and returns typed results.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A functional endpoint is declared as a `define.Function` var in `myserviceapi/definition.go` and implemented as a handler in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyFunction` comment on the `define.Function` var and on its In/Out structs. They are waypoints for future edits.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a functional endpoint:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the signature
- [ ] Step 3: Determine the method and route
- [ ] Step 4: Determine a description
- [ ] Step 5: Determine the required claims
- [ ] Step 6: Define complex types
- [ ] Step 7: Declare the endpoint in definition.go
- [ ] Step 8: Generate the boilerplate
- [ ] Step 9: Implement the logic in service.go
- [ ] Step 10: Test the function
- [ ] Step 11: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Determine the Signature

Determine the Go signature of the functional endpoint.

```go
func MyFunction(ctx context.Context, input1 string, input2 ThirdPartyStruct) (output1 map[string]MyStruct, err error)
```

Constraints:
- The first input argument must be `ctx context.Context`
- The function must return an `err error`
- Maps must be keyed by string, e.g. `map[string]any`
- Complex types (structs) are allowed by value or by reference, e.g. `MyStruct` or `*MyStruct`
- All input or output arguments must be serializable into JSON, including complex types
- Arguments must not be named `t` or `svc`
- Argument names must start with a lowercase letter
- The function name must start with an uppercase letter
- A return argument named `httpStatusCode` must be of type `int`
- If the return argument `httpResponseBody` is present, no other return argument other than `httpStatusCode` and `error` can be present
- The magic HTTP argument names `httpRequestBody`, `httpResponseBody` and `httpStatusCode` are documented in the rules file under "Magic HTTP Arguments"

#### Step 3: Determine the Method and Route

The method of the endpoint determines the HTTP method with which it will be addressable. Unless there's a reason to use a specific method, like for a REST API, use `ANY` to accept requests with any method.

The route of the endpoint is resolved relative to the hostname of the microservice to determine how it is addressed. The common approach is to use the name of the endpoint in kebab-case as its route, e.g. `/my-function`.

To set a port other than the default 443, prefix the route with the port, e.g. `:1234/my-function`.

Encase path arguments with `{}` , e.g. `/section/{section}/page/{page...}`.

Prefix the route with `//` to set a hostname other than that of this microservice, e.g. `//another.host.name:1234/on-something`

#### Step 4: Determine a Description

Describe the endpoint starting with its name, in Go doc style: `MyFunction does X`. This becomes the godoc comment on the `define.Function` var.

Describe **what the endpoint does and the effect it produces**, not who is expected to call it. `"Charges the card and returns a receipt id"` is good; `"called by the LLM as a tool"` or `"used by the checkout page"` is not.

Do not write per-argument descriptions in the godoc. Put them in `jsonschema_description:"..."` tags on the In/Out struct fields (Step 7).

#### Step 5: Determine the Required Claims

Determine if the endpoint should be restricted to authorized actors only. Compose a boolean expression over the JWT claims associated with the request that if not met will cause the request to be denied. For example: `roles.manager && level>2`. Default to closed: in a standard ingress configuration an empty `requiredClaims` on a `:443` endpoint (or any port the operator added to `AllowedInternalPorts`) is reachable by the entire internet. Leave it empty only for an intentionally public endpoint; if the endpoint wields a stored secret or a privileged side effect, it must be gated by `requiredClaims` and/or an internal port. See the Ports and Authentication sections of `.claude/rules/microbus.md`.

#### Step 6: Define Complex Types

Identify the struct types in the signature and define each in the `myserviceapi` directory following the
`add-type` skill: one file per type, camelCase `json` tags with `omitzero`, `jsonschema_description` tags,
optional `dv8` validation tags, and an optional pure `Validate` method for cross-field invariants. A type
owned by another microservice is aliased rather than redefined. Skip this step if there are no complex types.

#### Step 7: Declare the Endpoint in `definition.go`

Append the `define.Function` var and its In/Out structs to `myserviceapi/definition.go`.

```go
/*
MyFunction does X.
*/
var MyFunction = define.Function{ // MARKER: MyFunction
	Host: Hostname, Method: "ANY", Route: "/my-function",
	In: MyFunctionIn{}, Out: MyFunctionOut{},
}

// MyFunctionIn are the input arguments of MyFunction.
type MyFunctionIn struct { // MARKER: MyFunction
	Input1 string           `json:"input1,omitzero" jsonschema_description:"Input1 is X"`
	Input2 ThirdPartyStruct `json:"input2,omitzero" jsonschema_description:"Input2 is X"`
}

// MyFunctionOut are the output arguments of MyFunction.
type MyFunctionOut struct { // MARKER: MyFunction
	Output1 map[string]MyStruct `json:"output1,omitzero" jsonschema_description:"Output1 is X"`
}
```

- `Host` is always `Hostname`. `Method` and `Route` come from Step 3. Set `In` and `Out` to the In/Out struct literals (`MyFunctionIn{}`, `MyFunctionOut{}`)
- The In struct holds the input arguments excluding `ctx`; the Out struct holds the output arguments excluding `err`. Use PascalCase field names and camelCase `json` tags with `omitzero`
- In struct fields may carry `dv8` validation tags (see the `add-type` skill); they are enforced automatically after decoding, rejecting an invalid payload with `400 Bad Request` before the handler runs. Tags on Out struct fields are inert
- For a magic HTTP argument (`httpRequestBody`, `httpResponseBody`, `httpStatusCode`), set the field's `json` tag to `-`. A `jsonschema_description` tag still applies to a body field (`HTTPRequestBody`/`HTTPResponseBody`) and describes the whole body payload in the OpenAPI doc, e.g. `` `json:"-" jsonschema_description:"The object to create"` ``
- If an In/Out field's type comes from another package (e.g. a `time.Time` field needs `"time"`), add that import to `definition.go`
- Add the gating and routing fields only when needed:
  - `RequiredClaims: "roles.manager && level>2"` for the claims from Step 5 (omit when public)
  - `TimeBudget: 30 * time.Second` to cap the handler's duration (omit for the default; add the `time` import if used)
  - `LoadBalancing: define.None` to multicast to all replicas, or `LoadBalancing: "my-queue"` for a named queue; omit for the default hostname queue (load-balanced among peers)

#### Step 8: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `myserviceapi/client.go`, `intermediate.go`, `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then verify the microservice compiles with `go vet ./...` from the project root.

#### Step 9: Implement the Logic in `service.go`

The previous step generated a placeholder handler `func (svc *Service) MyFunction(...)` in `service.go`, with the signature and godoc projected from `definition.go`, tagged `// MARKER: MyFunction` and holding a `// TODO: Implement MyFunction` body. Replace that body with the handler's logic. Leave the generated signature and godoc as they are: they are the contract from `definition.go`, so if the signature is wrong, fix `definition.go` and regenerate rather than editing `service.go`. Complex types refer to their definition in `myserviceapi`. Add imports for any packages the body references that are not already imported (e.g. `"time"` for a `time.Time` value).

#### Step 10: Test the Function

Skip this step if instructed to be "quick" or to skip tests.

The boilerplate generator created a placeholder test function `TestMyService_MyFunction` in `service_test.go`, tagged with a `// MARKER: MyFunction` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 11: Housekeeping

Follow the `housekeeping` skill.
