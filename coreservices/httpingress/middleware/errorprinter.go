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
	"fmt"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"go.opentelemetry.io/otel/trace"
)

// ErrorPrinter returns a middleware that outputs any error to the response body.
// It should typically be the first middleware.
// Error details and stack trace are only printed on localhost.
func ErrorPrinter() Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			err = next(w, r) // No trace
			if err == nil {
				return nil
			}
			// Headers
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Cache-Control", "no-store")
			// Status code
			statusCode := errors.StatusCode(err)
			if statusCode <= 0 || statusCode >= 1000 {
				statusCode = http.StatusInternalServerError
			}
			w.WriteHeader(statusCode)
			// Error message with trace ID
			traceID := ""
			span := trace.SpanFromContext(r.Context())
			if span != nil {
				traceID = span.SpanContext().TraceID().String()
			}
			if ww, ok := w.(*httpx.ResponseRecorder); ok { // Always true
				ww.ClearBody()
			}
			if httpx.IsLocalhostAddress(r) {
				w.Write([]byte(fmt.Sprintf("%+v", err)))
				if traceID != "" {
					w.Write([]byte("\n\n{trace/"))
					w.Write([]byte(traceID))
					w.Write([]byte("}"))
				}
			} else {
				w.Write([]byte(http.StatusText(statusCode) + " " + traceID))
				if traceID != "" {
					w.Write([]byte(" {trace/"))
					w.Write([]byte(traceID))
					w.Write([]byte("}"))
				}
			}
			return nil
		}
	}
}
