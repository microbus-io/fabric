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

package httpx

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/microbus-io/testarossa"
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
	testarossa.NoError(t, err)
	err = ParseRequestData(r, &data)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, data.X)
	testarossa.Equal(t, 1, data.Y)

	r, err = http.NewRequest("POST", `/path?x=2`, &buf)
	testarossa.NoError(t, err)
	err = ParseRequestData(r, &data)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 2, data.X)
	testarossa.Equal(t, 1, data.Y)
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
	testarossa.NoError(t, err)
	err = ParseRequestData(r, &data)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, data.X)
	testarossa.Equal(t, 1, data.Y)

	r, err = http.NewRequest("POST", `/path?x=2`, &buf)
	testarossa.NoError(t, err)
	err = ParseRequestData(r, &data)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 2, data.X)
	testarossa.Equal(t, 1, data.Y)
}
