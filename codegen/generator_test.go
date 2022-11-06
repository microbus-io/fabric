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