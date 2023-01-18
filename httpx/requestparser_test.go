/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

package httpx

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_RequestParserQueryArgs(t *testing.T) {
	t.Parallel()

	var data struct {
		X struct {
			A int
			B int
		}
		Y struct {
			A int
			B int
		}
		S string
		A []int
		B bool
		E string
	}
	r, err := http.NewRequest("GET", `/path?x.a=5&x[b]=3&y={"a":1,"b":2}&s="str"&a=[1,2,3]&b=true&e=`, nil)
	assert.NoError(t, err)
	err = ParseRequestData(r, &data)
	assert.NoError(t, err)
	assert.Equal(t, 5, data.X.A)
	assert.Equal(t, 3, data.X.B)
	assert.Equal(t, 1, data.Y.A)
	assert.Equal(t, 2, data.Y.B)
	assert.Equal(t, "str", data.S)
	assert.Equal(t, []int{1, 2, 3}, data.A)
	assert.Equal(t, true, data.B)
	assert.Equal(t, "", data.E)
}

func TestHttpx_RequestParserOverrideJSON(t *testing.T) {
	t.Parallel()

	var data struct {
		X int
		Y int
	}
	var buf bytes.Buffer
	buf.WriteString(`{"x":1,"y":1}`)

	r, err := http.NewRequest("POST", `/path`, &buf)
	r.Header.Set("Content-Type", "application/json")
	assert.NoError(t, err)
	err = ParseRequestData(r, &data)
	assert.NoError(t, err)
	assert.Equal(t, 1, data.X)
	assert.Equal(t, 1, data.Y)

	r, err = http.NewRequest("POST", `/path?x=2`, &buf)
	assert.NoError(t, err)
	err = ParseRequestData(r, &data)
	assert.NoError(t, err)
	assert.Equal(t, 2, data.X)
	assert.Equal(t, 1, data.Y)
}

func TestHttpx_RequestParserOverrideFormData(t *testing.T) {
	t.Parallel()

	var data struct {
		X int
		Y int
	}
	var buf bytes.Buffer
	buf.WriteString(`x=1&y=1`)

	r, err := http.NewRequest("POST", `/path`, &buf)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	assert.NoError(t, err)
	err = ParseRequestData(r, &data)
	assert.NoError(t, err)
	assert.Equal(t, 1, data.X)
	assert.Equal(t, 1, data.Y)

	r, err = http.NewRequest("POST", `/path?x=2`, &buf)
	assert.NoError(t, err)
	err = ParseRequestData(r, &data)
	assert.NoError(t, err)
	assert.Equal(t, 2, data.X)
	assert.Equal(t, 1, data.Y)
}
