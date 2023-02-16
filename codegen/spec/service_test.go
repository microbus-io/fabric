/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSpec_ErrorsInYAML(t *testing.T) {
	t.Parallel()

	tcGeneral := []string{
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
	}

	tcTypedHandlers := []string{
		`
xxx:
  - signature: OnFunc(ctx context.Context) (result int)
`,
		"context type not allowed",
		// --------------------
		`
xxx:
  - signature: OnFunc(x int) (result int, err error)
`,
		"error type not allowed",
		// --------------------
		`
xxx:
  - signature: onFunc(x int) (y int)
`,
		"start with uppercase",
		// --------------------
		`
xxx:
  - signature: OnFunc(X int) (y int)
`,
		"start with lowercase",
		// --------------------
		`
xxx:
  - signature: OnFunc(x float64) (Y float64)
`,
		"start with lowercase",
		// --------------------
		`
xxx:
  - signature: OnFunc(x os.File) (y int)
`,
		"dot notation",
		// --------------------
		`
xxx:
  - signature: OnFunc(x Time) (x Duration)
`,
		"duplicate arg",
		// --------------------
		`
xxx:
  - signature: OnFunc(b boolean, x uint64, x int) (y int)
`,
		"duplicate arg",
		// --------------------
		`
xxx:
  - signature: OnFunc(x map[string]string) (y int, b bool, y int)
`,
		"duplicate arg",
		// --------------------
		`
xxx:
  - signature: OnFunc(m map[int]int)
`,
		"map keys",
		// --------------------
		`
xxx:
  - signature: OnFunc(m mutex)
`,
		"primitive type",
		// --------------------
		`
xxx:
  - signature: OnFunc(m int
`,
		"closing parenthesis",
		// --------------------
		`
xxx:
  - signature: OnFunc(m int) (x int
`,
		"closing parenthesis",
		// --------------------
		`
xxx:
  - signature: OnFunc(mint) (x int)
`,
		"invalid argument",
		// --------------------
		`
xxx:
  - signature: OnFunc(x int) (mint)
`,
		"invalid argument",
	}

	tcFunctions := []string{
		`
functions:
  - signature: Func(s []*int)
    path: :99999/...
`,
		"invalid path",
		// --------------------
		`
functions:
  - signature: Func(s string)
    queue: skip
`,
		"invalid queue",
	}

	tcEvents := []string{
		`
events:
  - signature: Func(s []*int)
`,
		"must start with 'On'",
		// --------------------
		`
events:
  - signature: OnFunc(s []*int)
    path: :99999/...
`,
		"invalid path",
	}

	tcSinks := []string{
		`
sinks:
  - signature: Func(s []*int)
    source: from/somewhere/else
`,
		"must start with 'On'",
		// --------------------
		`
sinks:
  - signature: OnFunc(s []*int)
`,
		"invalid source",
		// --------------------
		`
sinks:
  - signature: OnFunc(s []*int)
    source: https://www.example.com
`,
		"invalid source",
		// --------------------
		`
sinks:
  - signature: OnFunc(s []*int)
    source: from/somewhere/else
    forHost: invalid.ho$t
`,
		"invalid host name",
	}

	tcConfigs := []string{
		`
configs:
  - signature: func() (b bool)
`,
		"start with uppercase",
		// --------------------
		`
configs:
  - signature: Func()
`,
		"single return value",
		// --------------------
		`
configs:
  - signature: Func(x int) (b bool)
`,
		"arguments not allowed",
		// --------------------
		`
configs:
  - signature: Func() (b byte)
`,
		"invalid return type",
		// --------------------
		`
configs:
  - signature: Func() (b string)
    validation: xyz
`,
		"invalid validation rule",
		// --------------------
		`
configs:
  - signature: Func() (b string)
    validation: str ^[a-z]+$
    default: 123
`,
		"doesn't validate against rule",
	}

	tcTickers := []string{
		`
tickers:
  - signature: func()
`,
		"start with uppercase",
		// --------------------
		`
tickers:
  - signature: Func(x int)
`,
		"arguments or return values not allowed",
		// --------------------
		`
tickers:
  - signature: Func() (x string)
`,
		"arguments or return values not allowed",
		// --------------------
		`
tickers:
  - signature: Func()
`,
		"non-positive interval",
		// --------------------
		`
tickers:
  - signature: Func()
    interval: -2m
`,
		"non-positive interval",
		// --------------------
		`
tickers:
  - signature: Func()
    interval: 2m
    timeBudget: -1m
`,
		"negative time budget",
	}

	tcWebs := []string{
		`
webs:
  - signature: func()
`,
		"start with uppercase",
		// --------------------
		`
webs:
  - signature: Func(x int)
`,
		"arguments or return values not allowed",
		// --------------------
		`
webs:
  - signature: Func() (x string)
`,
		"arguments or return values not allowed",
		// --------------------
		`
webs:
  - signature: Func()
    path: :99999/...
`,
		"invalid path",
		// --------------------
		`
webs:
  - signature: Func()
    queue: skip
`,
		"invalid queue",
	}

	tcService := []string{
		`
functions:
  - signature: Func(x int) (y int)
webs:
  - signature: Func()
`,
		"duplicate",
	}

	testCases := []string{}
	testCases = append(testCases, tcConfigs...)
	testCases = append(testCases, tcEvents...)
	testCases = append(testCases, tcFunctions...)
	testCases = append(testCases, tcGeneral...)
	testCases = append(testCases, tcService...)
	testCases = append(testCases, tcSinks...)
	testCases = append(testCases, tcTickers...)
	testCases = append(testCases, tcWebs...)
	for i := 0; i < len(testCases); i += 2 {
		tc := testCases[i]
		if !strings.Contains(tc, "general:") {
			tc = `
general:
  host: any.host
` + tc
		}
		var svc Service
		err := yaml.Unmarshal([]byte(tc), &svc)
		if assert.Error(t, err, "%s", tc) {
			assert.Contains(t, err.Error(), testCases[i+1], "%s", tc)
		}
	}

	for i := 0; i < len(tcTypedHandlers); i += 2 {
		for _, x := range []string{"functions:", "events:", "sinks:"} {
			tc := `
general:
  host: any.host
` + strings.ReplaceAll(tcTypedHandlers[i], "xxx:", x) +
				`    source: this/path/is/ok`
			var svc Service
			err := yaml.Unmarshal([]byte(tc), &svc)
			if assert.Error(t, err, "%s", tc) {
				assert.Contains(t, err.Error(), tcTypedHandlers[i+1], "%s", tc)
			}
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
