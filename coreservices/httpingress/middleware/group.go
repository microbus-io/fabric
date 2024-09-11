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

package middleware

import (
	"net/http"

	"github.com/microbus-io/fabric/connector"
)

// Group returns a middleware that chains the nested middleware together and executes them sequentially in order.
// It can be used in conjunction with the [OnRoute] middleware to apply a group of middleware to a specific route.
func Group(nested ...Middleware) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		handler := next
		for n := len(nested) - 1; n >= 0; n-- {
			handler = nested[n](handler)
		}
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			return handler(w, r) // No trace
		}
	}
}
