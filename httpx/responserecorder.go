package httpx

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
)

// ResponseRecorder is used to record HTTP responses
type ResponseRecorder struct {
	header     http.Header
	body       *bytes.Buffer
	bytes      []byte
	statusCode int
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		header:     make(http.Header),
		statusCode: 0,
	}
}

// Header enables setting headers.
// It implements the http.ResponseWriter interface
func (rr *ResponseRecorder) Header() http.Header {
	return rr.header
}

// Write writes bytes to the body of the response.
// It implements the http.ResponseWriter interface
func (rr *ResponseRecorder) Write(b []byte) (int, error) {
	if rr.bytes == nil && rr.body == nil {
		rr.bytes = b
		return len(b), nil
	}

	if rr.body == nil {
		rr.body = &bytes.Buffer{}
		rr.body.Write(rr.bytes)
		rr.bytes = nil
	}
	return rr.body.Write(b)
}

// WriteHeader writes the header to the response.
// It implements the http.ResponseWriter interface
func (rr *ResponseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
}

// Result returns the response generated by the recorder
func (rr *ResponseRecorder) Result() *http.Response {
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: rr.statusCode,
		Header:     rr.header,
	}
	if res.StatusCode == 0 {
		res.StatusCode = http.StatusOK
	}
	res.Status = fmt.Sprintf("%03d %s", res.StatusCode, http.StatusText(res.StatusCode))
	if rr.bytes != nil {
		res.Body = NewBodyReader(rr.bytes)
		res.ContentLength = int64(len(rr.bytes))
	} else if rr.body != nil {
		res.Body = NewBodyReader(rr.body.Bytes())
		res.ContentLength = int64(rr.body.Len())
	}
	rr.header.Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
	return res
}