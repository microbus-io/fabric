/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"io"
	"net/http"

	"github.com/microbus-io/fabric/errors"
)

// Copy writes the HTTP response to the HTTP response writer.
func Copy(w http.ResponseWriter, res *http.Response) error {
	for k, v := range res.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(res.StatusCode)
	_, err := io.Copy(w, res.Body)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
