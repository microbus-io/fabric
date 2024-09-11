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
	"strings"

	"github.com/microbus-io/fabric/connector"
)

// RewriteRootPath returns a middleware that rewrites the root path "/" with one that can be routed to such as "/root".
func RewriteRootPath(rootPath string) Middleware {
	if !strings.HasPrefix(rootPath, "/") {
		rootPath = "/" + rootPath
	}
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			if r.URL.Path == "/" {
				r.URL.Path = rootPath
			}
			return next(w, r) // No trace
		}
	}
}
