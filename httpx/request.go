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
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/microbus-io/fabric/errors"
)

// SetRequestBody sets the body of the request.
// Arguments of type [io.Reader], [io.ReadCloser], []byte and string are serialized in binary form.
// [url.Values] and [QArgs] are serialized as form data.
// All other types are serialized as JSON.
// The Content-Type Content-Length headers will be set to match the body if they can be determined and unless already set.
func SetRequestBody(r *http.Request, body any) error {
	if body == nil {
		return nil
	}
	hasContentType := r.Header.Get("Content-Type") != ""
	switch v := body.(type) {
	case io.ReadCloser:
		r.Body = v
	case io.Reader:
		r.Body = io.NopCloser(v)
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
	case url.Values:
		b := []byte(v.Encode())
		r.Body = NewBodyReader(b)
		if !hasContentType {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
	case QArgs:
		b := []byte(v.Encode())
		r.Body = NewBodyReader(b)
		if !hasContentType {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
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
