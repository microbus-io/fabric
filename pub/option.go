package pub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/utils"
)

// Option is used to construct a request in Connector.Publish.
type Option func(req *Request) error

// Method sets the method of the request.
func Method(method string) Option {
	method = strings.ToUpper(method)
	return func(req *Request) error {
		req.Method = method
		return nil
	}
}

// URL sets the URL of the request.
func URL(url string) Option {
	return func(req *Request) error {
		u, err := utils.ParseURL(url)
		if err != nil {
			return errors.Trace(err)
		}
		u.RawQuery += req.queryArgs
		req.URL = u.String()
		return nil
	}
}

// GET sets the method and URL of the request.
func GET(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "GET"
		return nil
	}
}

// DELETE sets the method and URL of the request.
func DELETE(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "DELETE"
		return nil
	}
}

// HEAD sets the method and URL of the request.
func HEAD(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "HEAD"
		return nil
	}
}

// POST sets the method and URL of the request.
func POST(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "POST"
		return nil
	}
}

// PUT sets the method and URL of the request.
func PUT(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "PUT"
		return nil
	}
}

// PATCH sets the method and URL of the request.
func PATCH(url string) Option {
	return func(req *Request) error {
		err := req.Apply(URL(url))
		if err != nil {
			return errors.Trace(err)
		}
		req.Method = "PATCH"
		return nil
	}
}

// Header adds the header to the request.
// The same header may have multiple values.
func Header(name, value string) Option {
	return func(req *Request) error {
		if value != "" {
			req.Header.Add(name, value)
		}
		return nil
	}
}

// ContentLength sets the Content-Length header of the request.
func ContentLength(len int) Option {
	return func(req *Request) error {
		req.Header.Set("Content-Length", strconv.Itoa(len))
		return nil
	}
}

// QueryArg adds the query argument to the request.
// The same argument may have multiple values.
func QueryArg(name string, value any) Option {
	return func(req *Request) error {
		if value != "" {
			if len(req.queryArgs) > 0 {
				req.queryArgs += "&"
			}
			v := fmt.Sprintf("%v", value)
			req.queryArgs += url.QueryEscape(name) + "=" + url.QueryEscape(v)
			if req.URL != "" {
				u, err := utils.ParseURL(req.URL)
				if err != nil {
					return errors.Trace(err)
				}
				if len(u.RawQuery) > 0 {
					u.RawQuery += "&"
				}
				u.RawQuery += url.QueryEscape(name) + "=" + url.QueryEscape(v)
				req.URL = u.String()
			}
		}
		return nil
	}
}

// Query adds the escaped query arguments to the request.
// The same argument may have multiple values.
func Query(escapedQueryArgs string) Option {
	return func(req *Request) error {
		if escapedQueryArgs != "" {
			if len(req.queryArgs) > 0 {
				req.queryArgs += "&"
			}
			req.queryArgs += escapedQueryArgs
			if req.URL != "" {
				u, err := utils.ParseURL(req.URL)
				if err != nil {
					return errors.Trace(err)
				}
				if len(u.RawQuery) > 0 {
					u.RawQuery += "&"
				}
				u.RawQuery += escapedQueryArgs
				req.URL = u.String()
			}
		}
		return nil
	}
}

// Body sets the body of the request.
// Arguments of type io.Reader, []byte and string are serialized in binary form.
// url.Values is serialized as form data.
// All other types are serialized as JSON.
// The Content-Type and Content-Length headers may be set by this function.
func Body(body any) Option {
	return func(req *Request) error {
		if body == nil {
			return nil
		}
		switch v := body.(type) {
		case io.Reader:
			req.Body = v
		case []byte:
			req.Body = httpx.NewBodyReader(v)
			req.Header.Set("Content-Length", strconv.Itoa(len(v)))
		case string:
			b := []byte(v)
			req.Body = httpx.NewBodyReader(b)
			req.Header.Set("Content-Length", strconv.Itoa(len(b)))
		case url.Values:
			b := []byte(v.Encode())
			req.Body = httpx.NewBodyReader(b)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Content-Length", strconv.Itoa(len(b)))
		default:
			j, err := json.Marshal(body)
			if err != nil {
				return errors.Trace(err)
			}
			req.Body = httpx.NewBodyReader(j)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Content-Length", strconv.Itoa(len(j)))
		}
		return nil
	}
}

// ContentType sets the Content-Type header.
func ContentType(contentType string) Option {
	return func(req *Request) error {
		req.Header.Set("Content-Type", contentType)
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

// Unicast indicates that a single response is expected from this request.
func Unicast() Option {
	return func(req *Request) error {
		req.Multicast = false
		return nil
	}
}

// Multicast indicates that a multiple responses are expected from this request.
func Multicast() Option {
	return func(req *Request) error {
		req.Multicast = true
		return nil
	}
}
