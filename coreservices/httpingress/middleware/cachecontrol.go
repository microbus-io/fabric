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

// CacheControl returns a middleware that sets the Cache-Control header if not otherwise specified.
func CacheControl(defaultValue string) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			err = next(w, r)
			if w.Header().Get("Cache-Control") == "" {
				w.Header().Set("Cache-Control", defaultValue)
			}
			return err // No trace
		}
	}
}
