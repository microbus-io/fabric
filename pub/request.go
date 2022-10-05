package pub

import (
	"io"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
)

// Request is used to construct an HTTP request that can be sent over the bus.
// Although technically public, it is used internally and should not be constructed by microservices directly
type Request struct {
	Method    string
	URL       string
	Header    http.Header
	Body      io.Reader
	Deadline  time.Time
	Multicast bool
}

// NewRequest constructs a new request from the provided options
func NewRequest(options ...Option) (*Request, error) {
	req := &Request{
		Header: make(http.Header),
	}
	err := req.Apply(options...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return req, nil
}

// Apply the provided options to the request
func (req *Request) Apply(options ...Option) error {
	for _, opt := range options {
		err := opt(req)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// ToHTTP constructs an HTTP request given the properties set for this request
func (req *Request) ToHTTP() (*http.Request, error) {
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	if !req.Deadline.IsZero() {
		frame.Of(httpReq).SetTimeBudget(time.Until(req.Deadline))
	}
	return httpReq, nil
}
