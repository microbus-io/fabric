/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
