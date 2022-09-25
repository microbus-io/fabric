package pub

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// Option is used to construct a request
type Option func(req *Request) error

// Method sets the method of the request
func Method(method string) Option {
	method = strings.ToUpper(method)
	return func(req *Request) error {
		req.method = method
		return nil
	}
}

// URL sets the URL of the request
func URL(url string) Option {
	return func(req *Request) error {
		req.url = url
		return nil
	}
}

// GET sets the method and URL of the request
func GET(url string) Option {
	return func(req *Request) error {
		req.method = "GET"
		req.url = url
		return nil
	}
}

// POST sets the method and URL of the request
func POST(url string) Option {
	return func(req *Request) error {
		req.method = "POST"
		req.url = url
		return nil
	}
}

// Header add the header to the request.
// The same header may have multiple values
func Header(name, value string) Option {
	return func(req *Request) error {
		if value != "" {
			req.headers.Add(name, value)
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
			req.body = v
		case []byte:
			req.body = bytes.NewReader(v)
		case string:
			req.body = strings.NewReader(v)
		default:
			j, err := json.Marshal(body)
			if err != nil {
				return err
			}
			req.body = bytes.NewReader(j)
			req.headers.Set("Content-Type", "application/json")
		}
		return nil
	}
}

// TimeBudget sets the time budget of the request.
// Once a time budget is set, it is only possible to shorten it, not extend it
func TimeBudget(budget time.Duration) Option {
	if budget < 0 {
		budget = 0
	}
	return func(req *Request) error {
		if !req.hasBudget || req.timeBudget > budget {
			req.hasBudget = true
			req.timeBudget = budget
		}
		return nil
	}
}
