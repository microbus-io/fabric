package pub

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
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
		if err := utils.ValidateURL(url); err != nil {
			return errors.Trace(err)
		}
		req.URL = url
		return nil
	}
}

// GET sets the method and URL of the request
func GET(url string) Option {
	return func(req *Request) error {
		if err := utils.ValidateURL(url); err != nil {
			return errors.Trace(err)
		}
		req.Method = "GET"
		req.URL = url
		return nil
	}
}

// POST sets the method and URL of the request
func POST(url string) Option {
	return func(req *Request) error {
		if err := utils.ValidateURL(url); err != nil {
			return errors.Trace(err)
		}
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
			req.Body = utils.NewBodyReader(v)
		case string:
			req.Body = utils.NewBodyReader([]byte(v))
		default:
			j, err := json.Marshal(body)
			if err != nil {
				return errors.Trace(err)
			}
			req.Body = utils.NewBodyReader(j)
			req.Header.Set("Content-Type", "application/json")
		}
		return nil
	}
}

// TimeBudget sets a timeout for the request.
// The default time budget is 20 seconds.
func TimeBudget(timeout time.Duration) Option {
	if timeout < 0 {
		timeout = 0
	}
	return func(req *Request) error {
		req.TimeBudget = timeout
		return nil
	}
}

// Unicast indicates that a single response is expected from this request
func Unicast() Option {
	return func(req *Request) error {
		req.Multicast = false
		return nil
	}
}

// Multicast indicates that a multiple responses are expected from this request
func Multicast() Option {
	return func(req *Request) error {
		req.Multicast = true
		return nil
	}
}
