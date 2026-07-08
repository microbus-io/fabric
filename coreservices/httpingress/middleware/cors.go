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

package middleware

import (
	"net/http"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
)

// Cors returns a middleware that responds to the CORS origin OPTION request and blocks requests from disallowed origins.
// allowedOrigin returns the value to set in Access-Control-Allow-Origin for the given request and Origin header,
// and whether the origin is trusted to make credentialed requests. An empty value rejects the request with 403.
// The literal "*" value allows any origin; it is never credentialed.
func Cors(allowedOrigin func(r *http.Request, origin string) (allowed string, credentialed bool)) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed, credentialed := allowedOrigin(r, origin)
				if allowed == "" {
					return errors.New("disallowed origin '%s'", origin, http.StatusForbidden)
				}
				// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", allowed)
				if allowed != "*" {
					// A reflected origin varies by request; a shared cache must not serve it across origins
					h.Add("Vary", "Origin")
				}
				if credentialed && allowed != "*" {
					// Credentialed mode: a * is taken literally rather than as a wildcard,
					// so the preflight's requested method and headers are reflected instead
					h.Set("Access-Control-Allow-Credentials", "true")
					if r.Method == "OPTIONS" {
						reqMethod := r.Header.Get("Access-Control-Request-Method")
						if reqMethod != "" {
							h.Set("Access-Control-Allow-Methods", reqMethod)
							h.Add("Vary", "Access-Control-Request-Method")
						}
						reqHeaders := r.Header.Get("Access-Control-Request-Headers")
						if reqHeaders != "" {
							h.Set("Access-Control-Allow-Headers", reqHeaders)
							h.Add("Vary", "Access-Control-Request-Headers")
						}
					}
				} else {
					// Uncredentialed mode: the browser blocks credentials, and the * values below act
					// as true wildcards.
					// Authorization is carved out of the Allow-Headers wildcard by spec and must be named.
					h.Set("Access-Control-Allow-Methods", "*")
					h.Set("Access-Control-Allow-Headers", "*, Authorization")
					h.Set("Access-Control-Expose-Headers", "*")
				}
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
