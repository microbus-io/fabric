// Code generated by Microbus. DO NOT EDIT.

/*
Package eventsinkapi implements the public API of the eventsink.example microservice,
including clients and data structures.

The event sink microservice handles events that are fired by the event source microservice.
*/
package eventsinkapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

var (
	_ context.Context
	_ json.Decoder
	_ http.Request
	_ strings.Reader
	_ time.Duration

	_ errors.TracedError
	_ pub.Request
	_ sub.Subscription
)

const ServiceName = "eventsink.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Client is an interface to calling the endpoints of the eventsink.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the eventsink.example microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "eventsink.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the eventsink.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the eventsink.example microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "eventsink.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// RegisteredIn are the input arguments of Registered.
type RegisteredIn struct {
}

// RegisteredOut are the return values of Registered.
type RegisteredOut struct {
	Data struct {
		Emails []string `json:"emails"`
	}
	HTTPResponse *http.Response
	Err error
}

// Get retrieves the return values.
func (_out *RegisteredOut) Get() (emails []string, err error) {
	emails = _out.Data.Emails
	err = _out.Err
	return
}

/*
Registered returns the list of registered users.
*/
func (_c *Client) Registered(ctx context.Context) (emails []string, err error) {
	_in := RegisteredIn{
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/registered`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out RegisteredOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&(_out.Data))
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	emails = _out.Data.Emails
	return
}

/*
Registered returns the list of registered users.
*/
func (_c *MulticastClient) Registered(ctx context.Context, _options ...pub.Option) <-chan *RegisteredOut {
	_in := RegisteredIn{
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *RegisteredOut, 1)
		_res <- &RegisteredOut{Err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/registered`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *RegisteredOut, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r RegisteredOut
			_httpRes, _err := _i.Get()
			_r.HTTPResponse = _httpRes
			if _err != nil {
				_r.Err = errors.Trace(_err)
			} else {
				_err = json.NewDecoder(_httpRes.Body).Decode(&(_r.Data))
				if _err != nil {
					_r.Err = errors.Trace(_err)
				}
			}
			_res <- &_r
		}
		close(_res)
	}()
	return _res
}
