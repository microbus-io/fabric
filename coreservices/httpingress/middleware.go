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

package httpingress

import (
	"net/http"

	"github.com/microbus-io/fabric/connector"
)

// MiddlewareFunc is a processor that can be added to pre- or post-process a request.
// It should call the next function in the chain.
type MiddlewareFunc func(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error)

// Middleware is a processor that can be added to pre- or post-process a request.
// It should call the next function in the chain.
type Middleware interface {
	Serve(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error)
}

// simpleMiddleware converts a function to a middleware interface.
type simpleMiddleware struct {
	f MiddlewareFunc
}

// Serve pre- or post-processes a request.
func (fmw *simpleMiddleware) Serve(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error) {
	return fmw.f(w, r, next)
}
