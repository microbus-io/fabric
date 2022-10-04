package pub

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
)

// Option is used to construct a request in Connector.Publish
type Option func(req *Request) error

// Method sets the method of the request
func Method(method string) Option {
	method = strings.ToUpper(method)
	return func(req *Request) error {
		req.Method = method
		return nil
	}
}

// URL sets the URL of the request
func URL(url string) Option {
	return func(req *Request) error {
		req.URL = url
		return nil
	}
}

// GET sets the method and URL of the request
func GET(url string) Option {
	return func(req *Request) error {
		req.Method = "GET"
		req.URL = url
		return nil
	}
}

// POST sets the method and URL of the request
func POST(url string) Option {
	return func(req *Request) error {
		req.Method = "POST"
		req.URL = url
		return nil
	}
}

// Header add the header to the request.
// The same header may have multiple values
func Header(name, value string) Option {
	return func(req *Request) error {
		if value != "" {
			req.Header.Add(name, value)
		}
		return nil
	}
}

// Body sets the body of the request.
// Arguments of type io.Reader, []byte and string are serialized in binary form.
// All other types are serialized as JSON
func Body(body any) Option {
	return func(req *Request) error {
		if body == nil {
			return nil
		}
		switch v := body.(type) {
		case io.Reader:
			req.Body = v
		case []byte:
			req.Body = bytes.NewReader(v)
		case string:
			req.Body = strings.NewReader(v)
		default:
			j, err := json.Marshal(body)
			if err != nil {
				return errors.Trace(err)
			}
			req.Body = bytes.NewReader(j)
			req.Header.Set("Content-Type", "application/json")
		}
		return nil
	}
}

// Deadline sets the deadline of the request.
// Once a deadline is set, it is only possible to shorten it, not extend it
func Deadline(deadline time.Time) Option {
	return func(req *Request) error {
		if req.Deadline.IsZero() || req.Deadline.After(deadline) {
			req.Deadline = deadline
		}
		return nil
	}
}

// TimeBudget sets the deadline of the request to a time in the future.
// Once a deadline is set, it is only possible to shorten it, not extend it
func TimeBudget(timeout time.Duration) Option {
	return Deadline(time.Now().Add(timeout))
}
