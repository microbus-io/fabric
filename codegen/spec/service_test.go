package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSpec_ErrorsInYAML(t *testing.T) {
	t.Parallel()

	testCases := []string{
		// -------------------- GENERAL
		`
general:
`,
		"invalid host",
		// --------------------
		`
general:
  host: $uper.$ervice
`,
		"invalid host",
		// -------------------- FUNCTIONS
		`
general:
  host: super.service
functions:
  - signature: Func(ctx context.Context) (result int)
`,
		"context type not allowed",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x int) (result int, err error)
`,
		"error type not allowed",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: func(x int) (y int)
`,
		"start with uppercase",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(X int) (y int)
`,
		"start with lowercase",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x float64) (Y float64)
`,
		"start with lowercase",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x os.File) (y int)
`,
		"dot notation",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Since(x Time) (x Duration)
`,
		"duplicate arg",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(b boolean, x uint64, x int) (y int)
`,
		"duplicate arg",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x map[string]string) (y int, b bool, y int)
`,
		"duplicate arg",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(m map[int]int)
`,
		"map keys",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(m mutex)
`,
		"primitive type",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(m int
`,
		"closing parenthesis",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(m int) (x int
`,
		"closing parenthesis",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(mint) (x int)
`,
		"invalid argument",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x int) (mint)
`,
		"invalid argument",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(user User) (ok bool)
`,
		"undeclared",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(id string) (user User)
`,
		"undeclared",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(s []*int)
    path: :99999/...
`,
		"invalid path",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(s string)
    queue: skip
`,
		"invalid queue",
		// -------------------- TYPES
		`
general:
  host: super.service
types:
  - name: User
`,
		"missing type specification",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User
    define:
      x: int
    import: package/path/of/another/microservice
`,
		"ambiguous type specification",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User_Record
`,
		"invalid type name",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User.Record
`,
		"invalid type name",
		// --------------------
		`
general:
  host: super.service
types:
  - name: user
`,
		"invalid type name",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User
    define:
      FirstName: string
`,
		"start with lowercase",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User
    import: https://www.example.com
`,
		"invalid import path",
		// --------------------
		`
general:
  host: super.service
types:
  - name: User
    define:
      id: UUID
`,
		"undeclared",
		// -------------------- CONFIGS
		`
general:
  host: super.service
configs:
  - signature: func() (b bool)
`,
		"start with uppercase",
		// --------------------
		`
general:
  host: super.service
configs:
  - signature: Func()
`,
		"single return value",
		// --------------------
		`
general:
  host: super.service
configs:
  - signature: Func(x int) (b bool)
`,
		"arguments not allowed",
		// --------------------
		`
general:
  host: super.service
configs:
  - signature: Func() (b byte)
`,
		"invalid return type",
		// --------------------
		`
general:
  host: super.service
configs:
  - signature: Func() (b string)
    validation: xyz
`,
		"invalid validation rule",
		// --------------------
		`
general:
  host: super.service
configs:
  - signature: Func() (b string)
    validation: str ^[a-z]+$
    default: 123
`,
		"doesn't validate against rule",
		// -------------------- TICKERS
		`
general:
  host: super.service
tickers:
  - signature: func()
`,
		"start with uppercase",
		// --------------------
		`
general:
  host: super.service
tickers:
  - signature: Func(x int)
`,
		"arguments or return values not allowed",
		// --------------------
		`
general:
  host: super.service
tickers:
  - signature: Func() (x string)
`,
		"arguments or return values not allowed",
		// --------------------
		`
general:
  host: super.service
tickers:
  - signature: Func()
`,
		"non-positive interval",
		// --------------------
		`
general:
  host: super.service
tickers:
  - signature: Func()
    interval: -2m
`,
		"non-positive interval",
		// --------------------
		`
general:
  host: super.service
tickers:
  - signature: Func()
    interval: 2m
    timeBudget: -1m
`,
		"negative time budget",
		// -------------------- WEBS
		`
general:
  host: super.service
webs:
  - signature: func()
`,
		"start with uppercase",
		// --------------------
		`
general:
  host: super.service
webs:
  - signature: Func(x int)
`,
		"arguments or return values not allowed",
		// --------------------
		`
general:
  host: super.service
webs:
  - signature: Func() (x string)
`,
		"arguments or return values not allowed",
		// --------------------
		`
general:
  host: super.service
webs:
  - signature: Func()
    path: :99999/...
`,
		"invalid path",
		// --------------------
		`
general:
  host: super.service
webs:
  - signature: Func()
    queue: skip
`,
		"invalid queue",
		// --------------------
	}

	for i := 0; i < len(testCases); i += 2 {
		var svc Service
		err := yaml.Unmarshal([]byte(testCases[i]), &svc)
		if assert.Error(t, err, "%s", testCases[i]) {
			assert.Contains(t, err.Error(), testCases[i+1], "%s", testCases[i])
		}
	}
}

func TestSpec_HandlerInAndOut(t *testing.T) {
	t.Parallel()

	code := `signature: Func(i integer, b boolean, s string, f  float64)  (m map[string]string, a []int)`
	var h Handler
	err := yaml.Unmarshal([]byte(code), &h)
	assert.NoError(t, err)
	assert.Equal(t, h.In(), "ctx context.Context, i int, b bool, s string, f float64")
	assert.Equal(t, h.Out(), "m map[string]string, a []int, err error")
}

func TestSpec_QualifyTypes(t *testing.T) {
	t.Parallel()

	code := `
general:
  host: example.com
functions:
  - signature: Func(d Defined) (i Imported)
types:
  - name: Defined
    define:
      x: int
      y: int
  - name: Imported
    import: from/somewhere/else
`
	var svc Service
	err := yaml.Unmarshal([]byte(code), &svc)
	assert.NoError(t, err)
	svc.Package = "from/my"

	assert.Equal(t, "Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	assert.Equal(t, "Imported", svc.Functions[0].Signature.OutputArgs[0].Type)

	svc.FullyQualifyTypes()

	assert.Equal(t, "myapi.Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	assert.Equal(t, "myapi.Imported", svc.Functions[0].Signature.OutputArgs[0].Type)

	svc.ShorthandTypes()

	assert.Equal(t, "Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	assert.Equal(t, "Imported", svc.Functions[0].Signature.OutputArgs[0].Type)
}
