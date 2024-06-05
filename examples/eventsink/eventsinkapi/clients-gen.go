/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

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
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
)

var (
	_ context.Context
	_ *json.Decoder
	_ io.Reader
	_ *http.Request
	_ *url.URL
	_ strings.Reader
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.BodyReader
	_ pub.Option
	_ sub.Option
)

// Hostname is the default hostname of the microservice: eventsink.example.
const Hostname = "eventsink.example"

// Fully-qualified URLs of the microservice's endpoints.
var (
	URLOfRegistered = httpx.JoinHostAndPath(Hostname, `:443/registered`)
)

// Client is an interface to calling the endpoints of the eventsink.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  service.Publisher
	host string
}

// NewClient creates a new unicast client to the eventsink.example microservice.
func NewClient(caller service.Publisher) *Client {
	return &Client{
		svc:  caller,
		host: "eventsink.example",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the eventsink.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  service.Publisher
	host string
}

// NewMulticastClient creates a new multicast client to the eventsink.example microservice.
func NewMulticastClient(caller service.Publisher) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "eventsink.example",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// RegisteredIn are the input arguments of Registered.
type RegisteredIn struct {
}

// RegisteredOut are the return values of Registered.
type RegisteredOut struct {
	Emails []string `json:"emails"`
}

// RegisteredResponse is the response to Registered.
type RegisteredResponse struct {
	data RegisteredOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *RegisteredResponse) Get() (emails []string, err error) {
	emails = _out.data.Emails
	err = _out.err
	return
}

/*
Registered returns the list of registered users.
*/
func (_c *MulticastClient) Registered(ctx context.Context) <-chan *RegisteredResponse {
	_url := httpx.JoinHostAndPath(_c.host, `:443/registered`)
	_url = httpx.InjectPathArguments(_url, map[string]any{
	})
	_in := RegisteredIn{
	}
	var _query url.Values
	_body := _in
	_ch := _c.svc.Publish(
		ctx,
		pub.Method(`POST`),
		pub.URL(_url),
		pub.Query(_query),
		pub.Body(_body),
	)

	_res := make(chan *RegisteredResponse, cap(_ch))
	for _i := range _ch {
		var _r RegisteredResponse
		_httpRes, _err := _i.Get()
		_r.HTTPResponse = _httpRes
		if _err != nil {
			_r.err = _err // No trace
		} else {
			_err = json.NewDecoder(_httpRes.Body).Decode(&(_r.data))
			if _err != nil {
				_r.err = errors.Trace(_err)
			}
		}
		_res <- &_r
	}
	close(_res)
	return _res
}

/*
Registered returns the list of registered users.
*/
func (_c *Client) Registered(ctx context.Context) (emails []string, err error) {
	var _err error
	_url := httpx.JoinHostAndPath(_c.host, `:443/registered`)
	_url = httpx.InjectPathArguments(_url, map[string]any{
	})
	_in := RegisteredIn{
	}
	var _query url.Values
	_body := _in
	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method(`POST`),
		pub.URL(_url),
		pub.Query(_query),
		pub.Body(_body),
	)
	if _err != nil {
		err = _err // No trace
		return
	}
	var _out RegisteredOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	emails = _out.Emails
	return
}
