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
	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/service"
)

// Logger returns a middleware that logs requests.
func Logger(logger service.Logger) Middleware {
	pathAndQuery := func(r *http.Request, maxLen int) string {
		s := r.URL.Path
		if len(r.URL.RawQuery) != 0 {
			s += "?" + r.URL.RawQuery
		}
		if len(s) > maxLen {
			return s[:maxLen] + "..."
		}
		return s
	}

	return func(next connector.HTTPHandler) connector.HTTPHandler {
		metricsPrefix := "/" + metricsapi.Hostname
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			if !strings.HasPrefix(r.URL.Path, metricsPrefix) {
				logger.LogInfo(r.Context(), "Request received",
					"path", pathAndQuery(r, 512),
				)
			}
			err = next(w, r) // No trace
			if err != nil {
				statusCode := errors.StatusCode(err)
				if statusCode <= 0 || statusCode >= 1000 {
					statusCode = http.StatusInternalServerError
				}
				logFunc := logger.LogError
				maxLen := 2048
				if statusCode == http.StatusNotFound {
					logFunc = logger.LogInfo
					maxLen = 512
				} else if statusCode < 500 {
					logFunc = logger.LogWarn
					maxLen = 512
				}
				logFunc(r.Context(), "Serving",
					"error", err,
					"path", pathAndQuery(r, maxLen),
					"status", statusCode,
				)
			}
			return err // No trace
		}
	}
}
