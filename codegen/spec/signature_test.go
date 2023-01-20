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
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
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
