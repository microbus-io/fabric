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
	method     string
	url        string
	headers    http.Header
	body       io.Reader
	timeBudget time.Duration
	hasBudget  bool
}

// NewRequest constructs a new request from the provided options
func NewRequest(options ...Option) (*Request, error) {
	req := &Request{
		headers: make(http.Header),
	}
	for _, opt := range options {
		err := opt(req)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return req, nil
}

// ToHTTP constructs an HTTP request given the properties set for this request
func (req *Request) ToHTTP() (*http.Request, error) {
	httpReq, err := http.NewRequest(req.method, req.url, req.body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for name, value := range req.headers {
		httpReq.Header[name] = value
	}
	if req.hasBudget {
		frame.Of(httpReq).SetTimeBudget(req.timeBudget)
	}
	return httpReq, nil
}
