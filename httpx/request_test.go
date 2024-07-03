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
	"context"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_Request(t *testing.T) {
	req, err := NewRequest("GET", "https://example.com", nil)
	if assert.NoError(t, err) {
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
	}

	req, err = NewRequest("POST", "https://example.com", []byte("hello"))
	if assert.NoError(t, err) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "text/plain; charset=utf-8", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "hello", string(body))
	}

	req, err = NewRequest("POST", "https://example.com", "<html><body>hello</body></html>")
	if assert.NoError(t, err) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "text/html; charset=utf-8", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "<html><body>hello</body></html>", string(body))
	}

	req, err = NewRequest("POST", "https://example.com", `{"foo":"bar"}`)
	if assert.NoError(t, err) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, `{"foo":"bar"}`, string(body))
	}

	req, err = NewRequest("POST", "https://example.com", []byte(`[1,2,3,4]`))
	if assert.NoError(t, err) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, `[1,2,3,4]`, string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", strings.NewReader("hello"))
	if assert.NoError(t, err) {
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "hello", string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", url.Values{
		"a": []string{"a1"},
		"b": []string{"b1", "b2"},
		"c": []string{"c1"},
	})
	if assert.NoError(t, err) {
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "a=a1&b=b1&b=b2&c=c1", string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", QArgs{
		"a": "a1",
		"b": "b1",
		"c": "c1",
	})
	if assert.NoError(t, err) {
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "a=a1&b=b1&c=c1", string(body))
	}

	j := struct {
		S string `json:"s"`
		I int    `json:"i"`
		B bool   `json:"b"`
	}{
		S: "String",
		I: 123,
		B: true,
	}
	req, err = NewRequest("PUT", "https://example.com", &j)
	if assert.NoError(t, err) {
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, `{"s":"String","i":123,"b":true}`, string(body))
	}
}

func TestHttpx_MustRequest(t *testing.T) {
	ctx := context.Background()

	req := MustNewRequest("POST", "https://example.com", nil)
	assert.NotNil(t, req)
	assert.Panics(t, func() { MustNewRequest("POST", "@$^%&", nil) })

	req = MustNewRequestWithContext(ctx, "POST", "https://example.com", nil)
	assert.NotNil(t, req)
	assert.Panics(t, func() { MustNewRequestWithContext(ctx, "POST", "@$^%&", nil) })
}
