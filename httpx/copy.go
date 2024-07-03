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

package httpx

import (
	"bytes"
	"io"
	"net/http"

	"github.com/microbus-io/fabric/errors"
)

// Copy writes the [http.Response] to the [http.ResponseWriter].
// Do not modify the body of the [http.Response] after calling this methog.
// [http.ResponseWriter] is assumed to be empty.
func Copy(w http.ResponseWriter, res *http.Response) error {
	// Optimize for ResponseRecorder and BodyReader
	rr, ok1 := w.(*ResponseRecorder)
	br, ok2 := res.Body.(*BodyReader)
	if ok1 && ok2 {
		rr.header = res.Header
		rr.statusCode = res.StatusCode
		if br.bytes != nil {
			rr.body = bytes.NewBuffer(br.bytes)
			// This is somewhat risky: bytes are now owned by the buffer.
			// Also, the behavior here is to overwrite while the non-optimized case appends.
			// The memory savings are appealing to take this risk.
		} else {
			rr.body = nil
		}
		return nil
	}

	for k, v := range res.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(res.StatusCode)
	if res.Body != nil {
		_, err := io.Copy(w, res.Body)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
