/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/testarossa"
)

func TestHttpx_Request(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	req, err := NewRequest("GET", "https://example.com", nil)
	if assert.NoError(err) {
		assert.Equal("GET", req.Method)
		assert.Equal("https://example.com", req.URL.String())
	}

	req, err = NewRequest("POST", "https://example.com", []byte("hello"))
	if assert.NoError(err) {
		assert.Equal("POST", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("text/plain; charset=utf-8", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal("hello", string(body))
	}

	req, err = NewRequest("POST", "https://example.com", "<html><body>hello</body></html>")
	if assert.NoError(err) {
		assert.Equal("POST", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("text/html; charset=utf-8", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal("<html><body>hello</body></html>", string(body))
	}

	req, err = NewRequest("POST", "https://example.com", `{"foo":"bar"}`)
	if assert.NoError(err) {
		assert.Equal("POST", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(`{"foo":"bar"}`, string(body))
	}

	req, err = NewRequest("POST", "https://example.com", []byte(`[1,2,3,4]`))
	if assert.NoError(err) {
		assert.Equal("POST", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(`[1,2,3,4]`, string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", strings.NewReader("hello"))
	if assert.NoError(err) {
		assert.Equal("PUT", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal("hello", string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", url.Values{
		"a": []string{"a1"},
		"b": []string{"b1", "b2"},
		"c": []string{"c1"},
	})
	if assert.NoError(err) {
		assert.Equal("PUT", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal("a=a1&b=b1&b=b2&c=c1", string(body))
	}

	req, err = NewRequest("PUT", "https://example.com", QArgs{
		"a": "a1",
		"b": "b1",
		"c": "c1",
	})
	if assert.NoError(err) {
		assert.Equal("PUT", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal("a=a1&b=b1&c=c1", string(body))
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
	if assert.NoError(err) {
		assert.Equal("PUT", req.Method)
		assert.Equal("https://example.com", req.URL.String())
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(`{"s":"String","i":123,"b":true}`, string(body))
	}
}

func TestHttpx_MustRequest(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := context.Background()

	req := MustNewRequest("POST", "https://example.com", nil)
	assert.NotNil(req)
	err := errors.CatchPanic(func() error {
		MustNewRequest("POST", "@$^%&", nil)
		return nil
	})
	assert.Error(err)

	req = MustNewRequestWithContext(ctx, "POST", "https://example.com", nil)
	assert.NotNil(req)
	err = errors.CatchPanic(func() error {
		MustNewRequestWithContext(ctx, "POST", "@$^%&", nil)
		return nil
	})
	assert.Error(err)
}

func TestHttpx_ReadRequest(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	body := []byte("hello world body")
	req, err := http.NewRequest("POST", "https://example.com/path?q=1", bytes.NewReader(body))
	assert.NoError(err)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Custom", "value")
	req.ContentLength = int64(len(body))

	var buf bytes.Buffer
	err = req.Write(&buf)
	assert.NoError(err)

	parsed, err := ReadRequest(buf.Bytes())
	assert.NoError(err)
	assert.Equal("POST", parsed.Method)
	assert.Equal("/path", parsed.URL.Path)
	assert.Equal("q=1", parsed.URL.RawQuery)
	assert.Equal("value", parsed.Header.Get("X-Custom"))

	// The body is wrapped as a zero-copy, re-readable BodyReader over the tail of the buffer
	br, ok := parsed.Body.(*BodyReader)
	assert.True(ok)

	got, err := io.ReadAll(parsed.Body)
	assert.NoError(err)
	assert.Equal(body, got)

	br.Reset()
	got, err = io.ReadAll(parsed.Body)
	assert.NoError(err)
	assert.Equal(body, got)
}

func TestHttpx_ReadRequestEmptyBody(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	req, err := http.NewRequest("GET", "https://example.com/ping", nil)
	assert.NoError(err)

	var buf bytes.Buffer
	err = req.Write(&buf)
	assert.NoError(err)

	parsed, err := ReadRequest(buf.Bytes())
	assert.NoError(err)
	assert.Equal("GET", parsed.Method)

	got, err := io.ReadAll(parsed.Body)
	assert.NoError(err)
	assert.Len(got, 0)
}
