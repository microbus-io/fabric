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
	_ "embed"
	"net/http"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
)

//go:embed defaultfavicon.ico
var busIcon []byte

// DefaultFavIcon returns a middleware that responds to /favicon.ico, if the app does not.
func DefaultFavIcon() Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			err = next(w, r)
			if err != nil && r.URL.Path == "/favicon.ico" && errors.StatusCode(err) == http.StatusNotFound {
				w.Header().Set("Content-Type", "image/x-icon")
				w.Header().Set("Cache-Control", "public, max-age=86400") // 24hr
				if ww, ok := w.(*httpx.ResponseRecorder); ok {           // Always true
					ww.ClearBody()
				}
				w.Write(busIcon)
				err = nil
			}
			return err // No trace
		}
	}
}
