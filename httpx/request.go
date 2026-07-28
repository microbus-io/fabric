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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/utils"
)

var headerTerminator = []byte("\r\n\r\n")

// SetRequestBody sets the body of the request.
// Arguments of type [io.Reader], [io.ReadCloser], []byte and string are serialized in binary form.
// [url.Values] and [QArgs] are serialized as form data.
// All other types are serialized as JSON.
// The Content-Type Content-Length headers will be set to match the body if they can be determined and unless already set.
func SetRequestBody(r *http.Request, body any) error {
	if utils.IsNil(body) {
		return nil
	}
	hasContentType := r.Header.Get("Content-Type") != ""
	switch v := body.(type) {
	case io.Reader:
		// Buffer the reader so that the body can be read more than once
		if bodyReader, ok := v.(*BodyReader); ok {
			return SetRequestBody(r, bodyReader.Bytes())
		}
		b, err := io.ReadAll(v)
		if err != nil {
			return errors.Trace(err)
		}
		if closer, ok := v.(io.Closer); ok {
			_ = closer.Close()
		}
		return SetRequestBody(r, b)
	case []byte:
		r.Body = NewBodyReader(v)
		if !hasContentType {
			detected := ""
			if len(v) >= 2 && v[0] == '{' && v[len(v)-1] == '}' {
				err := json.Unmarshal(v, &map[string]any{})
				if err == nil {
					detected = "application/json"
				}
			}
			if len(v) >= 2 && v[0] == '[' && v[len(v)-1] == ']' {
				err := json.Unmarshal(v, &[]any{})
				if err == nil {
					detected = "application/json"
				}
			}
			if detected == "" {
				detected = http.DetectContentType(v)
			}
			r.Header.Set("Content-Type", detected)
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(v)))
		r.ContentLength = int64(len(v))
	case string:
		b := []byte(v)
		r.Body = NewBodyReader(b)
		if !hasContentType {
			detected := ""
			if len(b) >= 2 && b[0] == '{' && b[len(b)-1] == '}' {
				err := json.Unmarshal(b, &map[string]any{})
				if err == nil {
					detected = "application/json"
				}
			}
			if len(b) >= 2 && b[0] == '[' && b[len(b)-1] == ']' {
				err := json.Unmarshal(b, &[]any{})
				if err == nil {
					detected = "application/json"
				}
			}
			if detected == "" {
				detected = http.DetectContentType(b)
			}
			r.Header.Set("Content-Type", detected)
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		r.ContentLength = int64(len(b))
	case url.Values:
		b := []byte(v.Encode())
		r.Body = NewBodyReader(b)
		if !hasContentType {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		r.ContentLength = int64(len(b))
	case QArgs:
		b := []byte(v.Encode())
		r.Body = NewBodyReader(b)
		if !hasContentType {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		r.ContentLength = int64(len(b))
	default:
		j, err := json.Marshal(body)
		if err != nil {
			return errors.Trace(err)
		}
		r.Body = NewBodyReader(j)
		if !hasContentType {
			r.Header.Set("Content-Type", "application/json")
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(j)))
		r.ContentLength = int64(len(j))
	}
	return nil
}

// NewRequestWithContext returns a new [http.Request] given a method, URL, and optional body.
// Arguments of type [io.Reader], [io.ReadCloser], []byte and string are serialized in binary form.
// [url.Values] and [QArgs] are serialized as form data.
// All other types are serialized as JSON.
// The Content-Type Content-Length headers will be set to match the body if they can be determined and unless already set.
func NewRequestWithContext(ctx context.Context, method string, url string, body any) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = SetRequestBody(r, body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return r, nil
}

// MustNewRequestWithContext returns a new [http.Request] given a method, URL, and optional body. It panics on error.
// Arguments of type [io.Reader], [io.ReadCloser], []byte and string are serialized in binary form.
// [url.Values] and [QArgs] are serialized as form data.
// All other types are serialized as JSON.
// The Content-Type Content-Length headers will be set to match the body if they can be determined and unless already set.
func MustNewRequestWithContext(ctx context.Context, method string, url string, body any) *http.Request {
	r, err := NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		panic(err)
	}
	return r
}

// NewRequest wraps [NewRequestWithContext] with the background context.
func NewRequest(method string, url string, body any) (*http.Request, error) {
	return NewRequestWithContext(context.Background(), method, url, body)
}

// MustNewRequest wraps [NewRequestWithContext] with the background context. It panics on error.
func MustNewRequest(method string, url string, body any) *http.Request {
	r, err := NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		panic(err)
	}
	return r
}

// ReadRequest parses an HTTP request from a byte buffer emitted by the Microbus transport. It parses only the
// header section and wraps the remaining bytes as a zero-copy [BodyReader], rather than letting the standard
// parser buffer the body through a reader. This makes the body reusable ([BodyReader.Reset]) and hands its raw
// bytes to the fast paths in [Copy] and [NewFragRequest] without a copy.
//
// The buffer must hold a complete message framed by Content-Length, as the Microbus serializer emits (never
// chunked transfer-encoding). A buffer with no header terminator falls back to the standard parser.
func ReadRequest(data []byte) (*http.Request, error) {
	eoh := bytes.Index(data, headerTerminator)
	if eoh < 0 {
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
		return req, errors.Trace(err)
	}
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data[:eoh+len(headerTerminator)])))
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Body = NewBodyReader(data[eoh+len(headerTerminator):])
	return req, nil
}
