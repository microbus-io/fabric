/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
