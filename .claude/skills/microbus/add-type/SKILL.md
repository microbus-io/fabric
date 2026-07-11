---
name: add-type
description: TRIGGER when defining a domain type (struct) in a microservice's api package, or when adding validation rules or a Validate method to an existing api type. Referenced by add-function, add-config, add-outbound-event, and add-task.
---

**CRITICAL**: Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

A domain type is a struct declared in the microservice's api package (`myserviceapi`) and referenced by an
endpoint signature, a structured config, or an event payload. This skill covers the anatomy of such a type:
its file placement, its field tags, and its optional validation.

## File Placement and Ownership

Place each type in a separate file in the `myserviceapi` directory, named after the type in lowercase, e.g.
`myserviceapi/mystruct.go`. The type must live in the api package, not the service package, so that clients,
the generated boilerplate, and tests can all name it.

If the type is owned by this microservice, define its struct explicitly. If it is owned by another
microservice or package, define an alias to it instead and do not redefine its fields:

```go
package myserviceapi

import (
	"github.com/path/to/thirdparty"
)

// ThirdPartyStruct is X.
type ThirdPartyStruct = thirdparty.ThirdPartyStruct
```

## Field Tags

Every field of an owned type carries up to three tags:

1. **`json`** (required): camelCase name with the `omitzero` option, e.g. `json:"fooField,omitzero"`.
2. **`jsonschema_description`** (recommended): a short description of the field. It enriches the OpenAPI
   document and improves LLM tool-calling accuracy. Use this dedicated tag, not the `jsonschema:"description=..."`
   subtag form, which truncates at the first comma.
3. **`dv8`** (optional): validation directives, enforced automatically at the framework's trust boundary
   (see Validation below).

```go
package myserviceapi

// Person is a contact in the directory.
type Person struct {
	Name  string `json:"name,omitzero" jsonschema_description:"Name is the person's full name" dv8:"notzero,len<=64"`
	Email string `json:"email,omitzero" jsonschema_description:"Email is the person's email address" dv8:"trim,tolower,regexp ^.+@.+$"`
	Age   int    `json:"age,omitzero" jsonschema_description:"Age is the person's age in years" dv8:"val>=0,val<=120"`
}
```

## Validation With `dv8` Tags

The [`dv8`](https://github.com/microbus-io/dv8) directives most commonly used (see its README for the full set):

- `notzero` - requires a value that is not the type's zero value (non-nil for pointers, slices and maps; `true` for booleans)
- `len<=64`, `len>0` - length of a string (in runes), slice, or map
- `val>=0`, `val<=120` - value range for numeric, duration, and time fields
- `oneof S|M|L` - one of a set of values
- `regexp ^[0-9]{5}$` - pattern match; a comma inside the pattern is escaped `\\,` (e.g. `{2\\,5}`)
- `trim`, `tolower`, `toupper`, `default=x` - normalizing directives that rewrite the value before checks run
- `each <directive>` - applies a directive to the elements of a slice or the values of a map, e.g. `each len>0`
- `key <directive>` - applies a directive to the keys of a map

Where the tags are enforced:

- **In structs of functions and events**: validated automatically after decoding, before the handler runs.
  An invalid payload is rejected `400 Bad Request`; the handler never sees it.
- **Structured configs**: a value that fails validation is rejected before it is committed; the getter
  returns the normalized value.
- **Out structs**: tags are inert on output. Outputs are produced by trusted code and are not validated.
- **Direct Go calls**: a handler calling another handler in-process bypasses the framework boundary. When
  such internal composition needs validation, call `dv8.Validate(ctx, &v)` explicitly.

A malformed directive (a typo, a misapplied rule) fails the microservice at startup, not at request time.

## Custom Validation: the `Validate` Method

For cross-field invariants that tags cannot express, give the type a `Validate` method. The framework
discovers and calls it automatically wherever the type is validated, including when nested inside another
validated type:

```go
// Validate errors when the person record is internally inconsistent.
func (p *Person) Validate(ctx context.Context) error {
	if p.Retired && p.Age < 50 {
		return errors.New("retirement age is 50")
	}
	return nil
}
```

- The signature is `Validate(ctx context.Context) error`, or `Validate() error` when the context is not
  needed. A type can implement one or the other, never both.
- **The method must be a pure check of its fields**: no I/O, no downstream calls, no state mutation. It runs
  on the startup path and on every config refresh, so a validator that calls out turns validation into a
  distributed dependency.
- Validation runs after the tag directives, so normalized values (trimmed, defaulted) are what the method sees.
