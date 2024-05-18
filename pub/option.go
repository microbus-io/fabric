/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package pub

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
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
			// Invalid URLs are often received by hacker scanning tools,
			// for example, requests to .env
			return errors.Newcf(http.StatusNotFound, "invalid URL '%s", url)
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

// Header sets the header of the request. It overwrites any previously set value.
func Header(name string, value string) Option {
	return func(req *Request) error {
		if value != "" {
			req.Header.Set(name, value)
		} else {
			req.Header.Del(name)
		}
		return nil
	}
}

// CopyHeaders copies all non-Microbus headers from an upstream request.
func CopyHeaders(upstream *http.Request) Option {
	return func(req *Request) error {
		for h := range upstream.Header {
			if !strings.HasPrefix(h, frame.HeaderPrefix) {
				req.Header[h] = upstream.Header[h]
			}
		}
		return nil
	}
}

// Baggage sets a baggage header of the request. It overwrites any previously set value.
func Baggage(name string, value string) Option {
	return func(req *Request) error {
		if value != "" {
			req.Header.Set(frame.HeaderBaggagePrefix+name, value)
		} else {
			req.Header.Del(frame.HeaderBaggagePrefix + name)
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

// Query adds the encoded query arguments to the request.
// The same argument may have multiple values.
func QueryString(encodedQueryArgs string) Option {
	return func(req *Request) error {
		if encodedQueryArgs != "" {
			if len(req.queryArgs) > 0 {
				req.queryArgs += "&"
			}
			req.queryArgs += encodedQueryArgs
			if req.URL != "" {
				u, err := utils.ParseURL(req.URL)
				if err != nil {
					return errors.Trace(err)
				}
				if len(u.RawQuery) > 0 {
					u.RawQuery += "&"
				}
				u.RawQuery += encodedQueryArgs
				req.URL = u.String()
			}
		}
		return nil
	}
}

// Query adds arguments to the request.
func Query(args url.Values) Option {
	return QueryString(args.Encode())
}

// Body sets the body of the request.
// Arguments of type io.Reader, io.ReadCloser, []byte and string are serialized in binary form.
// url.Values is serialized as form data.
// All other types are serialized as JSON.
// The Content-Type Content-Length headers will be set to match the body if they can be determined and unless already set.
func Body(body any) Option {
	return func(req *Request) error {
		if body == nil {
			return nil
		}
		r, _ := http.NewRequest("POST", "", nil)
		if req.Header.Get("Content-Type") != "" {
			r.Header.Set("Content-Type", req.Header.Get("Content-Type"))
		}
		err := httpx.SetRequestBody(r, body)
		if err != nil {
			return errors.Trace(err)
		}
		req.Body = r.Body
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		req.Header.Set("Content-Length", r.Header.Get("Content-Length"))
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

// Noop does nothing.
func Noop() Option {
	return func(r *Request) error {
		return nil
	}
}
