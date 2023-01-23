/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by Microbus. DO NOT EDIT.

/*
Package eventsourceapi implements the public API of the eventsource.example microservice,
including clients and data structures.

The event source microservice fires events that are caught by the event sink microservice.
*/
package eventsourceapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ strings.Reader
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.BodyReader
	_ pub.Option
	_ sub.Option
)

// The default host name addressed by the clients is eventsource.example.
const HostName = "eventsource.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
	Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
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

// NewMulticastTrigger creates a new multicast trigger of the eventsource.example microservice.
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

// Hook assists in the subscription to the events of the eventsource.example microservice.
type Hook struct {
	svc  Service
	host string
}

// NewHook creates a new hook to the events of the eventsource.example microservice.
func NewHook(listener Service) *Hook {
	return &Hook{
		svc:  listener,
		host: "eventsource.example",
	}
}

// ForHost replaces the default host name of this hook.
func (_c *Hook) ForHost(host string) *Hook {
	_c.host = host
	return _c
}

// RegisterIn are the input arguments of Register.
type RegisterIn struct {
	Email string `json:"email"`
}

// RegisterOut are the return values of Register.
type RegisterOut struct {
	Allowed bool `json:"allowed"`
}

// RegisterResponse is the response to Register.
type RegisterResponse struct {
	data RegisterOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *RegisterResponse) Get() (allowed bool, err error) {
	allowed = _out.data.Allowed
	err = _out.err
	return
}

/*
Register attempts to register a new user.
*/
func (_c *MulticastClient) Register(ctx context.Context, email string, _options ...pub.Option) <-chan *RegisterResponse {
	_in := RegisterIn{
		email,
	}
	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/register`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *RegisterResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r RegisterResponse
			_httpRes, _err := _i.Get()
			_r.HTTPResponse = _httpRes
			if _err != nil {
				_r.err = errors.Trace(_err)
			} else {
				_err = json.NewDecoder(_httpRes.Body).Decode(&(_r.data))
				if _err != nil {
					_r.err = errors.Trace(_err)
				}
			}
			_res <- &_r
		}
		close(_res)
	}()
	return _res
}

// OnAllowRegisterIn are the input arguments of OnAllowRegister.
type OnAllowRegisterIn struct {
	Email string `json:"email"`
}

// OnAllowRegisterOut are the return values of OnAllowRegister.
type OnAllowRegisterOut struct {
	Allow bool `json:"allow"`
}

// OnAllowRegisterResponse is the response to OnAllowRegister.
type OnAllowRegisterResponse struct {
	data OnAllowRegisterOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *OnAllowRegisterResponse) Get() (allow bool, err error) {
	allow = _out.data.Allow
	err = _out.err
	return
}

/*
OnAllowRegister is called before a user is allowed to register.
Event sinks are given the opportunity to block the registration.
*/
func (_c *MulticastTrigger) OnAllowRegister(ctx context.Context, email string, _options ...pub.Option) <-chan *OnAllowRegisterResponse {
	_in := OnAllowRegisterIn{
		email,
	}
	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:417/on-allow-register`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *OnAllowRegisterResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r OnAllowRegisterResponse
			_httpRes, _err := _i.Get()
			_r.HTTPResponse = _httpRes
			if _err != nil {
				_r.err = errors.Trace(_err)
			} else {
				_err = json.NewDecoder(_httpRes.Body).Decode(&(_r.data))
				if _err != nil {
					_r.err = errors.Trace(_err)
				}
			}
			_res <- &_r
		}
		close(_res)
	}()
	return _res
}

// OnRegisteredIn are the input arguments of OnRegistered.
type OnRegisteredIn struct {
	Email string `json:"email"`
}

// OnRegisteredOut are the return values of OnRegistered.
type OnRegisteredOut struct {
}

// OnRegisteredResponse is the response to OnRegistered.
type OnRegisteredResponse struct {
	data OnRegisteredOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *OnRegisteredResponse) Get() (err error) {
	err = _out.err
	return
}

/*
OnRegistered is called when a user is successfully registered.
*/
func (_c *MulticastTrigger) OnRegistered(ctx context.Context, email string, _options ...pub.Option) <-chan *OnRegisteredResponse {
	_in := OnRegisteredIn{
		email,
	}
	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:417/on-registered`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *OnRegisteredResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r OnRegisteredResponse
			_httpRes, _err := _i.Get()
			_r.HTTPResponse = _httpRes
			if _err != nil {
				_r.err = errors.Trace(_err)
			} else {
				_err = json.NewDecoder(_httpRes.Body).Decode(&(_r.data))
				if _err != nil {
					_r.err = errors.Trace(_err)
				}
			}
			_res <- &_r
		}
		close(_res)
	}()
	return _res
}

/*
Register attempts to register a new user.
*/
func (_c *Client) Register(ctx context.Context, email string) (allowed bool, err error) {
	_in := RegisterIn{
		email,
	}
	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/register`)),
		pub.Body(_in),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out RegisterOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	allowed = _out.Allowed
	return
}

/*
OnAllowRegister is called before a user is allowed to register.
Event sinks are given the opportunity to block the registration.
*/
func (_c *Hook) OnAllowRegister(handler func(ctx context.Context, email string) (allow bool, err error), options ...sub.Option) error {
	doOnAllowRegister := func(w http.ResponseWriter, r *http.Request) error {
		var i OnAllowRegisterIn
		var o OnAllowRegisterOut
		err := httpx.ParseRequestData(r, &i)
		if err!=nil {
			return errors.Trace(err)
		}
		o.Allow, err = handler(
			r.Context(),
			i.Email,
		)
		if err != nil {
			return errors.Trace(err)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(o)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}
	path := httpx.JoinHostAndPath(_c.host, `:417/on-allow-register`)
	if handler == nil {
		return _c.svc.Unsubscribe(path)
	}
	return _c.svc.Subscribe(path, doOnAllowRegister, options...)
}

/*
OnRegistered is called when a user is successfully registered.
*/
func (_c *Hook) OnRegistered(handler func(ctx context.Context, email string) (err error), options ...sub.Option) error {
	doOnRegistered := func(w http.ResponseWriter, r *http.Request) error {
		var i OnRegisteredIn
		var o OnRegisteredOut
		err := httpx.ParseRequestData(r, &i)
		if err!=nil {
			return errors.Trace(err)
		}
		err = handler(
			r.Context(),
			i.Email,
		)
		if err != nil {
			return errors.Trace(err)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(o)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}
	path := httpx.JoinHostAndPath(_c.host, `:417/on-registered`)
	if handler == nil {
		return _c.svc.Unsubscribe(path)
	}
	return _c.svc.Subscribe(path, doOnRegistered, options...)
}
