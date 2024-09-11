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
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/connector"
)

// OnRoute returns a middleware that applies the conditional middleware only for URL paths that match the predicate.
func OnRoute(applyToPath func(path string) bool, conditional Middleware) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		nextOnRoute := conditional(next)
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			if applyToPath(r.URL.Path) {
				return nextOnRoute(w, r) // No trace
			} else {
				return next(w, r) // No trace
			}
		}
	}
}

/*
OnRoutePrefix returns a middleware that applies the conditional middleware only for URL paths that start with the prefix.

	httpIngress.Middleware().Append(middleware.OnRoutePrefix("/images/", middleware.CacheControl("private")))
*/
func OnRoutePrefix(pathPrefix string, conditional Middleware) Middleware {
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	return OnRoute(func(path string) bool {
		return strings.HasPrefix(path, pathPrefix)
	}, conditional)
}

// OnRouteRegex returns a middleware that applies the conditional middleware only for URL paths that match the regexp.
func OnRouteRegex(re *regexp.Regexp, conditional Middleware) Middleware {
	return OnRoute(func(path string) bool {
		return re.MatchString(path)
	}, conditional)
}
