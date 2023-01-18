package httpingress

import "net/http"

// PassThrough wraps calls to ResponseWriter, collecting metrics in the process.
type PassThrough struct {
	W  http.ResponseWriter
	N  int
	SC int
}

func (pt *PassThrough) Header() http.Header {
	return pt.W.Header()
}

func (pt *PassThrough) Write(b []byte) (int, error) {
	n, err := pt.W.Write(b)
	pt.N += n
	return n, err // No trace
}

func (pt *PassThrough) WriteHeader(statusCode int) {
	pt.SC = statusCode
	pt.W.WriteHeader(statusCode)
}
