package pub

import "net/http"

// Response is a union of an http.Response and an error.
// Only one or the other is valid
type Response struct {
	res *http.Response
	err error
}

// Get returns the http.Response or error stored in the composite Response
func (r *Response) Get() (*http.Response, error) {
	return r.res, r.err
}

// NewErrorResponse creates a new response containing an error
func NewErrorResponse(err error) *Response {
	return &Response{err: err}
}

// NewResponse creates a new response containing an http.Response
func NewHTTPResponse(res *http.Response) *Response {
	return &Response{res: res}
}
