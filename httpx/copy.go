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
// Headers are added, the body is appended, and the status code is overwritten.
// It is possible for this function to take ownership of the bytes of the body of the [http.Response] so do not modify them.
func Copy(w http.ResponseWriter, res *http.Response) error {
	// Optimize for ResponseRecorder and BodyReader
	rr, ok1 := w.(*ResponseRecorder)
	br, ok2 := res.Body.(*BodyReader)
	if ok1 && ok2 {
		if len(rr.header) == 0 {
			rr.header = res.Header
		} else {
			for k, vv := range res.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
		}
		rr.statusCode = res.StatusCode
		if br.bytes != nil {
			if rr.body == nil || rr.body.Len() == 0 {
				// This is somewhat risky: bytes are now owned by the buffer.
				// The memory savings are appealing to take this risk.
				rr.body = bytes.NewBuffer(br.bytes)
			} else {
				_, err := io.Copy(w, res.Body)
				if err != nil {
					return errors.Trace(err)
				}
			}
		}
		return nil
	}

	for k, vv := range res.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
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
