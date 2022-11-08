// Code generated by Microbus. DO NOT EDIT.

package eventsourceapi

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

const ServiceName = "eventsource.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Client is an interface to calling the endpoints of the eventsource.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the eventsource.example microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "eventsource.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the eventsource.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the eventsource.example microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "eventsource.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// MulticastTrigger is an interface to trigger the events of the eventsource.example microservice.
type MulticastTrigger struct {
	svc  Service
	host string
}

// NewMulticastTrigger creates a new multicast trigger to the eventsource.example microservice.
func NewMulticastTrigger(caller Service) *MulticastTrigger {
	return &MulticastTrigger{
		svc:  caller,
		host: "eventsource.example",
	}
}

// ForHost replaces the default host name of this trigger.
func (_c *MulticastTrigger) ForHost(host string) *MulticastTrigger {
	_c.host = host
	return _c
}

// RegisterIn are the input arguments of Register.
type RegisterIn struct {
	Email string `json:"email"`
}

// RegisterOut are the return values of Register.
type RegisterOut struct {
	Data struct {
		Allowed bool `json:"allowed"`
	}
	HTTPResponse *http.Response
	Err error
}

// Get retrieves the return values.
func (_out *RegisterOut) Get() (allowed bool, err error) {
	allowed = _out.Data.Allowed
	err = _out.Err
	return
}

/*
Register attempts to register a new user.
*/
func (_c *Client) Register(ctx context.Context, email string) (allowed bool, err error) {
	_in := RegisterIn{
		email,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/register`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out RegisterOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&(_out.Data))
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	allowed = _out.Data.Allowed
	return
}

/*
Register attempts to register a new user.
*/
func (_c *MulticastClient) Register(ctx context.Context, email string, _options ...pub.Option) <-chan *RegisterOut {
	_in := RegisterIn{
		email,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *RegisterOut, 1)
		_res <- &RegisterOut{Err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/register`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *RegisterOut, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r RegisterOut
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

// OnAllowRegisterHandler is a handler of the OnAllowRegister event.
type OnAllowRegisterHandler func (ctx context.Context, email string) (allow bool, err error)

// PathOfOnAllowRegister is the URL path of the OnAllowRegister event.
const PathOfOnAllowRegister = "/on-allow-register"

// OnAllowRegisterIn are the input arguments of OnAllowRegister.
type OnAllowRegisterIn struct {
	Email string `json:"email"`
}

// OnAllowRegisterOut are the return values of OnAllowRegister.
type OnAllowRegisterOut struct {
	Data struct {
		Allow bool `json:"allow"`
	}
	HTTPResponse *http.Response
	Err error
}

// Get retrieves the return values.
func (_out *OnAllowRegisterOut) Get() (allow bool, err error) {
	allow = _out.Data.Allow
	err = _out.Err
	return
}

/*
OnAllowRegister is called before a user is allowed to register.
Event sinks are given the opportunity to block the registration.
*/
func (_c *MulticastTrigger) OnAllowRegister(ctx context.Context, email string, _options ...pub.Option) <-chan *OnAllowRegisterOut {
	_in := OnAllowRegisterIn{
		email,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *OnAllowRegisterOut, 1)
		_res <- &OnAllowRegisterOut{Err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/on-allow-register`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *OnAllowRegisterOut, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r OnAllowRegisterOut
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

// OnRegisteredHandler is a handler of the OnRegistered event.
type OnRegisteredHandler func (ctx context.Context, email string) (err error)

// PathOfOnRegistered is the URL path of the OnRegistered event.
const PathOfOnRegistered = "/on-registered"

// OnRegisteredIn are the input arguments of OnRegistered.
type OnRegisteredIn struct {
	Email string `json:"email"`
}

// OnRegisteredOut are the return values of OnRegistered.
type OnRegisteredOut struct {
	Data struct {
	}
	HTTPResponse *http.Response
	Err error
}

// Get retrieves the return values.
func (_out *OnRegisteredOut) Get() (err error) {
	err = _out.Err
	return
}

/*
OnRegistered is called when a user is successfully registered.
*/
func (_c *MulticastTrigger) OnRegistered(ctx context.Context, email string, _options ...pub.Option) <-chan *OnRegisteredOut {
	_in := OnRegisteredIn{
		email,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *OnRegisteredOut, 1)
		_res <- &OnRegisteredOut{Err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/on-registered`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *OnRegisteredOut, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r OnRegisteredOut
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