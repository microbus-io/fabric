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
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
)

// Compress returns a middleware that compresses textual responses using brotli, gzip or deflate.
func Compress() Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			acceptEncoding := r.Header.Get("Accept-Encoding")
			if acceptEncoding == "" {
				// Client does not accept compression
				return next(w, r) // No trace
			}

			// Delegate the request downstream
			ww := httpx.NewResponseRecorder()
			err = next(ww, r)
			if err != nil {
				return err // No trace
			}
			res := ww.Result()

			if res.Body == nil || res.ContentLength < (1<<10) {
				// Don't bother with less than 1KB
				err = httpx.Copy(w, res)
				return errors.Trace(err)
			}

			contentEncoding := res.Header.Get("Content-Encoding")
			if contentEncoding != "" && contentEncoding != "identity" {
				// Already compressed?
				err = httpx.Copy(w, res)
				return errors.Trace(err)
			}

			contentType := res.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
				// Images, videos and audio are already compressed
				err = httpx.Copy(w, res)
				return errors.Trace(err)
			}

			// Compress the body
			var compressor io.WriteCloser
			encoding := ""
			switch {
			case strings.Contains(acceptEncoding, "br"):
				encoding = "br"
				compressor = brotli.NewWriter(w)
			case strings.Contains(acceptEncoding, "deflate"):
				encoding = "deflate"
				compressor, _ = flate.NewWriter(w, flate.DefaultCompression)
			case strings.Contains(acceptEncoding, "gzip"):
				encoding = "gzip"
				compressor = gzip.NewWriter(w)
			default:
				// No compression
				err = httpx.Copy(w, res)
				return errors.Trace(err)
			}

			// Copy status code and headers, without the body
			body := res.Body
			res.Body = nil
			httpx.Copy(w, res)
			// Set content headers
			w.Header().Del("Content-Length")
			w.Header().Set("Content-Encoding", encoding)
			// Compress the body
			_, err = io.Copy(compressor, body)
			compressor.Close()
			return errors.Trace(err)
		}
	}
}
