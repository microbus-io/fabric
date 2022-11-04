package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestCodeGen_ErrorsInYAML(t *testing.T) {
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
  - signature: Func(x int) (Y int)
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
  - signature: Func(x int) (x int)
`,
		"duplicate arg",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(b bool, x int, x int) (y int)
`,
		"duplicate arg",
		// --------------------
		`
general:
  host: super.service
functions:
  - signature: Func(x int) (y int, b bool, y int)
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
  - signature: Func(s string)
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

	// Create a temp directory
	dir := "testing-" + rand.AlphaNum32(12)
	os.Mkdir(dir, os.ModePerm)
	defer os.RemoveAll(dir)

	for i := 0; i < len(testCases); i += 2 {
		err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(testCases[i]), 0666)
		assert.NoError(t, err)

		gen := NewGenerator()
		gen.WorkDir = dir
		err = gen.Run()
		if assert.Error(t, err, "%s", testCases[i]) {
			assert.Contains(t, err.Error(), testCases[i+1], "%s", testCases[i])
		}
	}
}
