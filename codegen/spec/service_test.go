/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
	"testing"

	"github.com/microbus-io/testarossa"
	"gopkg.in/yaml.v3"
)

func TestSpec_ErrorsInFunctions(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s []*int)
    path: :BAD/...
`), &svc)
	testarossa.ErrorContains(t, err, "invalid port")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    queue: skip
`), &svc)
	testarossa.ErrorContains(t, err, "invalid queue")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: //bad.ho$t
`), &svc)
	testarossa.ErrorContains(t, err, "invalid hostname")
}

func TestSpec_ErrorsInPathArguments(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{ }
`), &svc)
	testarossa.ErrorContains(t, err, "must be an identifier")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{p$}
`), &svc)
	testarossa.ErrorContains(t, err, "must be an identifier")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{p +}
`), &svc)
	testarossa.ErrorContains(t, err, "must be an identifier")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{ +}
`), &svc)
	testarossa.ErrorContains(t, err, "must be an identifier")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{$+}
`), &svc)
	testarossa.ErrorContains(t, err, "must be an identifier")

	err = yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(s string)
    path: /{+}/hello
`), &svc)
	testarossa.ErrorContains(t, err, "must end path")
}

func TestSpec_ErrorsInEvents(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
events:
  - signature: Func(s []*int)
`), &svc)
	testarossa.ErrorContains(t, err, "must start with 'On'")

	err = yaml.Unmarshal([]byte(general+`
events:
  - signature: OnFunc(s []*int)
    path: :BAD/...
`), &svc)
	testarossa.ErrorContains(t, err, "invalid port")

	err = yaml.Unmarshal([]byte(general+`
events:
  - signature: OnFunc(s []*int)
    path: :0/...
`), &svc)
	testarossa.ErrorContains(t, err, "invalid port")
}

func TestSpec_ErrorsInSinks(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
sinks:
  - signature: Func(s []*int)
    source: from/somewhere/else
`), &svc)
	testarossa.ErrorContains(t, err, "must start with 'On'")

	err = yaml.Unmarshal([]byte(general+`
sinks:
  - signature: OnFunc(s []*int)
`), &svc)
	testarossa.ErrorContains(t, err, "invalid source")

	err = yaml.Unmarshal([]byte(general+`
sinks:
  - signature: OnFunc(s []*int)
    source: https://www.example.com
`), &svc)
	testarossa.ErrorContains(t, err, "invalid source")

	err = yaml.Unmarshal([]byte(general+`
sinks:
  - signature: OnFunc(s []*int)
    source: from/somewhere/else
    forHost: invalid.ho$t
`), &svc)
	testarossa.ErrorContains(t, err, "invalid hostname")
}

func TestSpec_ErrorsInConfigs(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
configs:
  - signature: func() (b bool)
`), &svc)
	testarossa.ErrorContains(t, err, "start with uppercase")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func()
`), &svc)
	testarossa.ErrorContains(t, err, "single return value")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func() (x int, y int)
`), &svc)
	testarossa.ErrorContains(t, err, "single return value")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func(x int) (b bool)
`), &svc)
	testarossa.ErrorContains(t, err, "arguments not allowed")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func() (b byte)
`), &svc)
	testarossa.ErrorContains(t, err, "invalid return type")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func() (b string)
    validation: xyz
`), &svc)
	testarossa.ErrorContains(t, err, "invalid validation rule")

	err = yaml.Unmarshal([]byte(general+`
configs:
  - signature: Func() (b string)
    validation: str ^[a-z]+$
    default: 123
`), &svc)
	testarossa.ErrorContains(t, err, "doesn't validate against rule")
}

func TestSpec_ErrorsInTickers(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
tickers:
  - signature: func()
`), &svc)
	testarossa.ErrorContains(t, err, "start with uppercase")

	err = yaml.Unmarshal([]byte(general+`
tickers:
  - signature: Func(x int)
`), &svc)
	testarossa.ErrorContains(t, err, "arguments or return values not allowed")

	err = yaml.Unmarshal([]byte(general+`
tickers:
  - signature: Func() (x int)
`), &svc)
	testarossa.ErrorContains(t, err, "arguments or return values not allowed")

	err = yaml.Unmarshal([]byte(general+`
tickers:
  - signature: Func()
`), &svc)
	testarossa.ErrorContains(t, err, "non-positive interval")

	err = yaml.Unmarshal([]byte(general+`
tickers:
  - signature: Func()
    interval: "-2m"
`), &svc)
	testarossa.ErrorContains(t, err, "non-positive interval")
}

func TestSpec_ErrorsInWebs(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
webs:
  - signature: func()
`), &svc)
	testarossa.ErrorContains(t, err, "start with uppercase")

	err = yaml.Unmarshal([]byte(general+`
webs:
  - signature: Func(x int)
`), &svc)
	testarossa.ErrorContains(t, err, "arguments or return values not allowed")

	err = yaml.Unmarshal([]byte(general+`
webs:
  - signature: Func() (x int)
`), &svc)
	testarossa.ErrorContains(t, err, "arguments or return values not allowed")

	err = yaml.Unmarshal([]byte(general+`
webs:
  - signature: Func()
    path: :BAD/...
`), &svc)
	testarossa.ErrorContains(t, err, "invalid port")

	err = yaml.Unmarshal([]byte(general+`
webs:
  - signature: Func()
    queue: skip
`), &svc)
	testarossa.ErrorContains(t, err, "invalid queue")
}

func TestSpec_ErrorsInService(t *testing.T) {
	t.Parallel()

	var svc Service
	general := `
general:
  host: ok.host
`

	err := yaml.Unmarshal([]byte(general+`
functions:
  - signature: Func(x int) (y int)
webs:
  - signature: Func()
`), &svc)
	testarossa.ErrorContains(t, err, "duplicate")
}

func TestSpec_HandlerInAndOut(t *testing.T) {
	t.Parallel()

	code := `signature: Func(i integer, b boolean, s string, f  float64)  (m map[string]string, a []int)`
	var h Handler
	err := yaml.Unmarshal([]byte(code), &h)
	testarossa.NoError(t, err)
	testarossa.Equal(t, h.In(), "ctx context.Context, i int, b bool, s string, f float64")
	testarossa.Equal(t, h.Out(), "m map[string]string, a []int, err error")
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
	testarossa.NoError(t, err)
	svc.Package = "from/my"

	testarossa.Equal(t, "Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	testarossa.Equal(t, "Imported", svc.Functions[0].Signature.OutputArgs[0].Type)

	svc.FullyQualifyTypes()

	testarossa.Equal(t, "myapi.Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	testarossa.Equal(t, "myapi.Imported", svc.Functions[0].Signature.OutputArgs[0].Type)

	svc.ShorthandTypes()

	testarossa.Equal(t, "Defined", svc.Functions[0].Signature.InputArgs[0].Type)
	testarossa.Equal(t, "Imported", svc.Functions[0].Signature.OutputArgs[0].Type)
}
