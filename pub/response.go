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

package pub

import "net/http"

// Response is a union of an http.Response and an error.
// Only one or the other is valid
type Response struct {
	res *http.Response
	err error
}

// Get returns the http.Response or error stored in the composite Response
func (r *Response) Get() (*http.Response, error) {
	return r.res, r.err
}

// NewErrorResponse creates a new response containing an error
func NewErrorResponse(err error) *Response {
	return &Response{err: err}
}

// NewResponse creates a new response containing an http.Response
func NewHTTPResponse(res *http.Response) *Response {
	return &Response{res: res}
}
