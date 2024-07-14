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

func TestSpec_Signature(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("Hello(x int, y string) (ok bool)"), &sig)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "Hello", sig.Name)
	testarossa.SliceLen(t, sig.InputArgs, 2)
	testarossa.Equal(t, "x", sig.InputArgs[0].Name)
	testarossa.Equal(t, "int", sig.InputArgs[0].Type)
	testarossa.Equal(t, "y", sig.InputArgs[1].Name)
	testarossa.Equal(t, "string", sig.InputArgs[1].Type)
	testarossa.SliceLen(t, sig.OutputArgs, 1)
	testarossa.Equal(t, "ok", sig.OutputArgs[0].Name)
	testarossa.Equal(t, "bool", sig.OutputArgs[0].Type)

	err = yaml.Unmarshal([]byte("Hello(x int)"), &sig)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "Hello", sig.Name)
	testarossa.SliceLen(t, sig.InputArgs, 1)
	testarossa.Equal(t, "x", sig.InputArgs[0].Name)
	testarossa.Equal(t, "int", sig.InputArgs[0].Type)
	testarossa.SliceLen(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("Hello() (e string, ok bool)"), &sig)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "Hello", sig.Name)
	testarossa.SliceLen(t, sig.InputArgs, 0)
	testarossa.SliceLen(t, sig.OutputArgs, 2)
	testarossa.Equal(t, "e", sig.OutputArgs[0].Name)
	testarossa.Equal(t, "string", sig.OutputArgs[0].Type)
	testarossa.Equal(t, "ok", sig.OutputArgs[1].Name)
	testarossa.Equal(t, "bool", sig.OutputArgs[1].Type)

	err = yaml.Unmarshal([]byte("Hello()"), &sig)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "Hello", sig.Name)
	testarossa.SliceLen(t, sig.InputArgs, 0)
	testarossa.SliceLen(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("Hello"), &sig)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "Hello", sig.Name)
	testarossa.SliceLen(t, sig.InputArgs, 0)
	testarossa.SliceLen(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("MockMe"), &sig)
	testarossa.Error(t, err)

	err = yaml.Unmarshal([]byte("TestMe"), &sig)
	testarossa.Error(t, err)
}

func TestSpec_HTTPArguments(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("Hello(x int, y string) (ok bool)"), &sig)
	testarossa.NoError(t, err)

	err = yaml.Unmarshal([]byte("Hello(x int, httpResponseBody string) (ok bool)"), &sig)
	testarossa.Error(t, err, "httpResponseBody can't be an input argument")
	err = yaml.Unmarshal([]byte("Hello(x int, httpStatusCode string) (ok bool)"), &sig)
	testarossa.Error(t, err, "httpStatusCode can't be an input argument")
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpRequestBody bool)"), &sig)
	testarossa.Error(t, err, "httpRequestBody can't be an output argument")

	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool)"), &sig)
	testarossa.NoError(t, err)
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, httpStatusCode int)"), &sig)
	testarossa.NoError(t, err)

	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, httpStatusCode bool)"), &sig)
	testarossa.Error(t, err, "httpStatusCode must be of type int")
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, z int, httpStatusCode int)"), &sig)
	testarossa.Error(t, err, "Output argument not allowed alongside httpResponseBody")
}

func TestSpec_TypedHandlers(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("OnFunc(ctx context.Context) (result int)"), &sig)
	testarossa.Error(t, err, "Context type not allowed")
	err = yaml.Unmarshal([]byte("OnFunc(x int) (result int, err error)"), &sig)
	testarossa.Error(t, err, "Error type not allowed")
	err = yaml.Unmarshal([]byte("onFunc(x int) (y int)"), &sig)
	testarossa.Error(t, err, "Endpoint name must start with uppercase")
	err = yaml.Unmarshal([]byte("OnFunc(X int) (y int)"), &sig)
	testarossa.Error(t, err, "Arg name must start with lowercase")
	err = yaml.Unmarshal([]byte("OnFunc(x int) (Y int)"), &sig)
	testarossa.Error(t, err, "Arg name must start with lowercase")
	err = yaml.Unmarshal([]byte("OnFunc(x os.File) (y int)"), &sig)
	testarossa.Error(t, err, "Dot notation not allowed")
	err = yaml.Unmarshal([]byte("OnFunc(x Time) (x Duration)"), &sig)
	testarossa.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(b boolean, x uint64, x int) (y int)"), &sig)
	testarossa.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(x map[string]string) (y int, b bool, y int)"), &sig)
	testarossa.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(m map[int]int)"), &sig)
	testarossa.Error(t, err, "Map keys must ne strings")
	err = yaml.Unmarshal([]byte("OnFunc(m mutex)"), &sig)
	testarossa.Error(t, err, "Primitive type")
	err = yaml.Unmarshal([]byte("OnFunc(m int"), &sig)
	testarossa.Error(t, err, "Missing closing parenthesis")
	err = yaml.Unmarshal([]byte("OnFunc(m int) (x int"), &sig)
	testarossa.Error(t, err, "Missing closing parenthesis")
	err = yaml.Unmarshal([]byte("OnFunc(mint) (x int)"), &sig)
	testarossa.Error(t, err, "Missing argument type")
	err = yaml.Unmarshal([]byte("OnFunc(m int) (xint)"), &sig)
	testarossa.Error(t, err, "Missing argument type")
}
