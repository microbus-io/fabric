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
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/microbus-io/fabric/connector"
)

// RequestTimeout returns a middleware that applies a timeout to the request.
// The value of the Request-Timeout header is used if provided in the request. Otherwise, the time budget is used.
func RequestTimeout(budget func() time.Duration) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			ctx := r.Context()
			timeBudget := budget()
			if r.Header.Get("Request-Timeout") != "" {
				headerTimeoutSecs, err := strconv.Atoi(r.Header.Get("Request-Timeout"))
				if err == nil && headerTimeoutSecs > 0 {
					timeBudget = time.Duration(headerTimeoutSecs) * time.Second
				}
			}
			if timeBudget > 0 {
				var cancel context.CancelFunc
				delegateCtx, cancel := context.WithTimeout(ctx, timeBudget)
				defer cancel()
				r = r.WithContext(delegateCtx)
			}
			return next(w, r) // No trace
		}
	}
}
