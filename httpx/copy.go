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
