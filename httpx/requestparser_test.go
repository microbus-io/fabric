/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
