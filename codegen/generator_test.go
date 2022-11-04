package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestCodeGen_FullGeneration(t *testing.T) {
	t.Parallel()

	serviceYaml := `
general:
  host: test.full.generation
  description: Testing full generation.
configs:
  - signature: Config1() (b bool)
    description: Config1 is a bool.
    default: true
    validation: bool
    callback: true
  - signature: Config2() (dur Duration)
    description: Config2 is a duration.
    default: 2m
    validation: dur [1m,5m]
functions:
  - signature: Func1(x Type1, y Type1) (result float)
    description: Func1 returns the distance between two points.
    path: :1234/distance
  - signature: Func2(term string) (count int)
    description: Func2 is a distributed search.
    path: :1234/count-occurrences
    queue: none
types:
  - name: Type1
    define:
      x: float
      y: float
  - name: Type2
    import: from/somewhere/else
webs:
  - signature: Web1()
  - signature: Web2()
    queue: none
tickers:
  - signature: Ticker1()
    description: Ticker1 runs once a minute.
    interval: 1m
    timeBudget: 30s
  - signature: Ticker2()
    description: Ticker1 runs once an hour.
    interval: 1h
`

	// Create a temp directory with the service.yaml file
	dir := "testing-" + rand.AlphaNum32(12)
	os.Mkdir(dir, os.ModePerm)
	defer os.RemoveAll(dir)
	err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(serviceYaml), 0666)
	assert.NoError(t, err)

	// Generate
	gen := NewGenerator()
	gen.WorkDir = dir
	err = gen.Run()
	assert.NoError(t, err)

	// Validate
	fileContains := func(fileName string, terms ...string) {
		b, err := os.ReadFile(filepath.Join(dir, fileName))
		assert.NoError(t, err, "%s", fileName)
		body := string(b)
		for _, term := range terms {
			assert.Contains(t, body, term, "%s", fileName)
		}
	}

	fileContains(
		filepath.Join(dir+"api", "clients-gen.go"),
		"Func1(ctx", "Func2(ctx", "Web1(ctx", "Web2(ctx",
	)
	fileContains(
		filepath.Join(dir+"api", "types-gen.go"),
		"type Type1", "type Type2", "from/somewhere/else",
	)
	fileContains(
		filepath.Join("intermediate", "intermediate-gen.go"),
		"svc.Subscribe(",
		"svc.impl.Func1", "svc.impl.Func1",
		") doFunc1(w", ") doFunc2(w",
		"svc.impl.Web1", "svc.impl.Web2",
		"svc.StartTicker(",
		"svc.impl.Ticker1", "svc.impl.Ticker2",
		"svc.DefineConfig(",
		"svc.impl.OnChangedConfig1",
		") Config1() (b bool)", ") Config2() (dur time.Duration)",
		"func Config1(b bool)", "func Config2(dur time.Duration)",
	)
	fileContains(
		filepath.Join("resources", "embed-gen.go"),
		"go:embed",
	)
	fileContains(
		"service.go",
		") Func1(ctx", ") Func2(ctx",
		") Web1(w", ") Web2(w",
		") Ticker1(ctx", ") Ticker2(ctx",
		") OnChangedConfig1(ctx",
	)
	fileContains(
		"service-gen.go",
		"intermediate.Config1", "intermediate.Config2",
	)
	fileContains(
		"version-gen.go",
		"Version", "SourceCodeSHA256",
	)
	fileContains(
		"version-gen_test.go",
		"SourceCodeSHA256",
	)

	ver, err := gen.currentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 1, ver.Version)

	// Second run should be a no op
	err = gen.Run()
	assert.NoError(t, err)

	ver, err = gen.currentVersion()
	assert.NoError(t, err)
	assert.Equal(t, 1, ver.Version)
}

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
