---
name: add-config
description: TRIGGER when user asks to add or modify a configuration property or setting, or to make a value configurable.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**CRITICAL**: A config property is declared as a `define.Config` var in `myserviceapi/definition.go`; its callback, if any, is implemented in `service.go`. Add the declaration and run `cmd/genservice`.

**CRITICAL**: Keep the `// MARKER: MyConfig` comment on the `define.Config` var.

## Workflow

Copy this checklist and track your progress:

```
Creating or modifying a configuration property:
- [ ] Step 1: Read local CLAUDE.md file
- [ ] Step 2: Determine the type
- [ ] Step 3: Determine the properties
- [ ] Step 4: Declare the config in definition.go
- [ ] Step 5: Generate the boilerplate
- [ ] Step 6: Implement the callback
- [ ] Step 7: Use the config
- [ ] Step 8: Test the callback
- [ ] Step 9: Add to config file
- [ ] Step 10: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Determine the Type

Determine the type of the configuration property.

A **scalar** config is one of `string`, `int`, `bool`, `float64`, or `time.Duration`. These are stored and converted natively.

A config may also be a **structured (JSON) value**: a struct, a slice (`[]int`), a map (`map[string]bool`), or any JSON-serializable composite (including slices/maps of structs). Its value is stored as JSON text; the generated getter unmarshals it into the typed value and the setter marshals it back. Reach for a structured config when the knob is a list, a set, or a group of related fields rather than a single scalar.

#### Step 3: Determine the Properties

Determine the properties of the configuration property:
- **Description**: explains the purpose of the property. It becomes the godoc on the `define.Config` var and starts with the property name
- **Default value**: an optional default, always written as a string (it flows through the same validate-and-convert path as a configured value)
- **Validation**: an optional validation rule (see below)
- **Secret**: whether the value is a secret that should not be logged
- **Callback**: whether `OnChangedMyConfig` should fire when the value changes, for example to reopen a connection to an external resource

Validation rules can be any of the following:
- `str` followed by a regexp: `str ^[a-zA-Z0-9]+$`
- `bool`
- `int` followed by an open, closed or mixed interval: `int [0,60]`
- `float` followed by an open, closed or mixed interval: `float [0.0,1.0)`
- `dur` followed by an open, closed or mixed interval of Go durations: `dur (0s,24h]`
- `set` followed by a pipe-separated list of values: `set Red|Green|Blue`
- `url`
- `email`
- `json`

#### Step 4: Declare the Config in `definition.go`

Append the `define.Config` var to `myserviceapi/definition.go`. The godoc is the description from Step 3.

```go
/*
MyConfig is X.
*/
var MyConfig = define.Config{ // MARKER: MyConfig
	Value:      int(0),
	Default:    "1",
	Validation: "int (1,100]",
	Secret:     true,
	Callback:   true,
}
```

- `Value` is a type carrier declaring the property's type: `string("")`, `int(0)`, `bool(false)`, `float64(0)`, or `time.Duration(0)`. The generator reads its type, never the value
- For a `time.Duration` config, add the `"time"` import to `definition.go`
- `Default` is the optional default as a string; omit when there is none
- `Validation` is the optional rule from Step 3; omit when there is none
- `Secret: true` marks a value that is never logged; omit when false
- `Callback: true` makes `OnChangedMyConfig` fire on change; omit when false

For a **structured (JSON) config**, the `Value` carrier is a composite literal of the type, and `Validation` is `json`:

```go
// RetryPolicy is the structured value of the Retry config.
type RetryPolicy struct {
	MaxRetries int    `json:"maxRetries,omitzero"`
	Backoff    string `json:"backoff,omitzero"`
}

/*
Retry configures how failed operations are retried.
*/
var Retry = define.Config{ // MARKER: Retry
	Value:      RetryPolicy{},
	Default:    `{"maxRetries":3,"backoff":"1s"}`,
	Validation: "json",
}
```

- The `Value` carrier may be a struct (`RetryPolicy{}`), a slice (`[]int{}`), a map (`map[string]bool{}`), or a slice/map of structs (`[]RetryPolicy{}`)
- Set `Validation: "json"` so the stored value is checked as valid JSON; when present, `Default` is the JSON text of the value (e.g. `[80,443]` for a `[]int`, `{"beta":true}` for a `map[string]bool`)
- Define any new struct type in the **api package**, in a separate file beside `definition.go` (e.g. `myserviceapi/retrypolicy.go`), following the `add-type` skill. It must live in the api package, not the service package, so the generated getter/setter (which live in the service package) and tests can name it. The type's `dv8` validation tags and optional pure `Validate` method are enforced on every configured value: a value that does not unmarshal into the type or that fails validation is rejected before it is committed
- The generated getter returns the typed value with normalizing `dv8` directives (`default=`, `trim`) applied; a missing or malformed stored value yields the type's zero value

#### Step 5: Generate the Boilerplate

From the microservice's directory, run the generator. It regenerates `intermediate.go` (the getter, setter, validation, and change dispatcher), `mock.go`, `mock_test.go`, and `manifest.yaml` from the updated `definition.go`. It also scaffolds a placeholder handler in `service.go` and a placeholder test in `service_test.go` for any new feature that lacks one, each ready for you to fill in.

```shell
go run github.com/microbus-io/fabric/cmd/genservice .
```

Then verify the microservice compiles with `go vet ./...` from the project root.

#### Step 6: Implement the Callback

Skip this step if the config does not have a callback.

For a config with a callback, the previous step generated a placeholder `func (svc *Service) OnChangedMyConfig(ctx context.Context) (err error)` in `service.go`, tagged `// MARKER: MyConfig` and holding a `// TODO` body. Fill in that body to handle the new value; leave the generated signature and godoc as they are.

#### Step 7: Use the Config

The config has no implementation of its own. Read its value from within other endpoints using the generated getter.

```go
myConfig := svc.MyConfig()
```

Use `svc.SetMyConfig(value)` to set the value programmatically, for example in tests.

For a structured config the getter returns the typed value (`retry := svc.Retry()` yields a `RetryPolicy`) and the setter takes it. When you need to name the type to construct a value, it is in the api package: `svc.SetRetry(myserviceapi.RetryPolicy{MaxRetries: 5})`.

#### Step 8: Test the Callback

Skip this step if the config does not have a callback, or if instructed to be "quick" or to skip tests.

When present, the boilerplate generator created a placeholder test function `TestMyService_OnChangedMyConfig` in `service_test.go`, tagged with a `// MARKER: MyConfig` comment and a `HINT` block. Add one or more test cases at the bottom of that function, following the pattern shown in its `HINT` comment. Do not remove the `HINT` comment.

`SetMyConfig` runs the value through the same validation as a configured value and returns an error when it fails. If the config has a `Validation` rule, cover both paths: a valid value that sets cleanly (`assert.NoError`, then assert the getter returns it), and an out-of-range or malformed value that is rejected (`assert.Error`). The `HINT` block only shows the happy path.

If the feature or microservice cannot or should not be tested - for example, it starts a daemon or requires a
resource unavailable in the test environment - do not delete the generated test. The generator keys on the test
function's name and would scaffold it again on the next regeneration. Instead, replace the entire body of the
test with a `t.Skip("reason")` explaining why.

#### Step 9: Add to Config File

Add a commented-out entry for the new configuration property to the appropriate config file at the root of the project, nested under the hostname of the microservice. Use the default value if one was defined, or leave it blank otherwise.

If the config is secret, add it to `config.local.yaml`. If the config is not secret, add it to `config.yaml`. Create the file if it does not exist.

If a section for the hostname already exists in the file, add the new property to that section. Otherwise, create a new section.

```yaml
my.service.hostname:
  # MyConfig: default
```

A structured config's value may be written either as native nested YAML or as a quoted JSON string; both
canonicalize to the same stored value:

```yaml
my.service.hostname:
  # Retry:
  #   maxRetries: 3
  #   backoff: 1s
```

#### Step 10: Housekeeping

Follow the `housekeeping` skill.
