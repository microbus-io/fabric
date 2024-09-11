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

// Cors returns a middleware that returns a 403 error for disallowed CORS origins.
func Cors(isAllowed func(origin string) bool) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
			origin := r.Header.Get("Origin")
			if origin != "" {
				// Block disallowed origins
				if !isAllowed(origin) {
					return errors.Newcf(http.StatusForbidden, "disallowed origin '%s'", origin)
				}
				// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "*")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Expose-Headers", "*")
				if r.Method == "OPTIONS" {
					// CORS preflight requests are returned empty
					w.WriteHeader(http.StatusNoContent)
					return nil
				}
			}
			return next(w, r) // No trace
		}
	}
}
