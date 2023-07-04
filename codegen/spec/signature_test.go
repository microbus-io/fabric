/*
Copyright (c) 2023 Microbus LLC and various contributors

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
}
