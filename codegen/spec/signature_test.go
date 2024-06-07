/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpec_Signature(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("Hello(x int, y string) (ok bool)"), &sig)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", sig.Name)
	assert.Len(t, sig.InputArgs, 2)
	assert.Equal(t, "x", sig.InputArgs[0].Name)
	assert.Equal(t, "int", sig.InputArgs[0].Type)
	assert.Equal(t, "y", sig.InputArgs[1].Name)
	assert.Equal(t, "string", sig.InputArgs[1].Type)
	assert.Len(t, sig.OutputArgs, 1)
	assert.Equal(t, "ok", sig.OutputArgs[0].Name)
	assert.Equal(t, "bool", sig.OutputArgs[0].Type)

	err = yaml.Unmarshal([]byte("Hello(x int)"), &sig)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", sig.Name)
	assert.Len(t, sig.InputArgs, 1)
	assert.Equal(t, "x", sig.InputArgs[0].Name)
	assert.Equal(t, "int", sig.InputArgs[0].Type)
	assert.Len(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("Hello() (e string, ok bool)"), &sig)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", sig.Name)
	assert.Len(t, sig.InputArgs, 0)
	assert.Len(t, sig.OutputArgs, 2)
	assert.Equal(t, "e", sig.OutputArgs[0].Name)
	assert.Equal(t, "string", sig.OutputArgs[0].Type)
	assert.Equal(t, "ok", sig.OutputArgs[1].Name)
	assert.Equal(t, "bool", sig.OutputArgs[1].Type)

	err = yaml.Unmarshal([]byte("Hello()"), &sig)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", sig.Name)
	assert.Len(t, sig.InputArgs, 0)
	assert.Len(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("Hello"), &sig)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", sig.Name)
	assert.Len(t, sig.InputArgs, 0)
	assert.Len(t, sig.OutputArgs, 0)

	err = yaml.Unmarshal([]byte("MockMe"), &sig)
	assert.Error(t, err)

	err = yaml.Unmarshal([]byte("TestMe"), &sig)
	assert.Error(t, err)
}

func TestSpec_HTTPArguments(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("Hello(x int, y string) (ok bool)"), &sig)
	assert.NoError(t, err)

	err = yaml.Unmarshal([]byte("Hello(x int, httpResponseBody string) (ok bool)"), &sig)
	assert.Error(t, err, "httpResponseBody can't be an input argument")
	err = yaml.Unmarshal([]byte("Hello(x int, httpStatusCode string) (ok bool)"), &sig)
	assert.Error(t, err, "httpStatusCode can't be an input argument")
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpRequestBody bool)"), &sig)
	assert.Error(t, err, "httpRequestBody can't be an output argument")

	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool)"), &sig)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, httpStatusCode int)"), &sig)
	assert.NoError(t, err)

	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, httpStatusCode bool)"), &sig)
	assert.Error(t, err, "httpStatusCode must be of type int")
	err = yaml.Unmarshal([]byte("Hello(x int, y string) (httpResponseBody bool, z int, httpStatusCode int)"), &sig)
	assert.Error(t, err, "Output argument not allowed alongside httpResponseBody")
}

func TestSpec_TypedHandlers(t *testing.T) {
	t.Parallel()

	var sig Signature

	err := yaml.Unmarshal([]byte("OnFunc(ctx context.Context) (result int)"), &sig)
	assert.Error(t, err, "Context type not allowed")
	err = yaml.Unmarshal([]byte("OnFunc(x int) (result int, err error)"), &sig)
	assert.Error(t, err, "Error type not allowed")
	err = yaml.Unmarshal([]byte("onFunc(x int) (y int)"), &sig)
	assert.Error(t, err, "Endpoint name must start with uppercase")
	err = yaml.Unmarshal([]byte("OnFunc(X int) (y int)"), &sig)
	assert.Error(t, err, "Arg name must start with lowercase")
	err = yaml.Unmarshal([]byte("OnFunc(x int) (Y int)"), &sig)
	assert.Error(t, err, "Arg name must start with lowercase")
	err = yaml.Unmarshal([]byte("OnFunc(x os.File) (y int)"), &sig)
	assert.Error(t, err, "Dot notation not allowed")
	err = yaml.Unmarshal([]byte("OnFunc(x Time) (x Duration)"), &sig)
	assert.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(b boolean, x uint64, x int) (y int)"), &sig)
	assert.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(x map[string]string) (y int, b bool, y int)"), &sig)
	assert.Error(t, err, "Duplicate arg name")
	err = yaml.Unmarshal([]byte("OnFunc(m map[int]int)"), &sig)
	assert.Error(t, err, "Map keys must ne strings")
	err = yaml.Unmarshal([]byte("OnFunc(m mutex)"), &sig)
	assert.Error(t, err, "Primitive type")
	err = yaml.Unmarshal([]byte("OnFunc(m int"), &sig)
	assert.Error(t, err, "Missing closing parenthesis")
	err = yaml.Unmarshal([]byte("OnFunc(m int) (x int"), &sig)
	assert.Error(t, err, "Missing closing parenthesis")
	err = yaml.Unmarshal([]byte("OnFunc(mint) (x int)"), &sig)
	assert.Error(t, err, "Missing argument type")
	err = yaml.Unmarshal([]byte("OnFunc(m int) (xint)"), &sig)
	assert.Error(t, err, "Missing argument type")
}
