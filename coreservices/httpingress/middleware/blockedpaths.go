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
	"github.com/microbus-io/fabric/errors"
)

// BlockedPaths returns a middleware that returns a 404 error for paths matching the predicate
// The path passed to the matcher is the full path of the URL, without query arguments.
func BlockedPaths(isBlocked func(path string) bool) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		notFound := errors.Newc(http.StatusNotFound, "")
		return func(w http.ResponseWriter, r *http.Request) error {
			if isBlocked(r.URL.Path) {
				w.WriteHeader(http.StatusNotFound)
				return notFound
			}
			return next(w, r) // No trace
		}
	}
}
