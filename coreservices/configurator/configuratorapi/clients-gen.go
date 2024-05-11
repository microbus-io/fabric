/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

/*
Package configuratorapi implements the public API of the configurator.sys microservice,
including clients and data structures.

The Configurator is a core microservice that centralizes the dissemination of configuration values to other microservices.
*/
package configuratorapi

import (
	"context"
	"encoding/json"
	"net/http"
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
	_ *http.Request
	_ strings.Reader
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.BodyReader
	_ pub.Option
	_ sub.Option
)

// HostName is the default host name of the microservice: configurator.sys.
const HostName = "configurator.sys"

// Fully-qualified URLs of the microservice's endpoints.
var (
	URLOfValues = httpx.JoinHostAndPath(HostName, ":443/values")
	URLOfRefresh = httpx.JoinHostAndPath(HostName, ":443/refresh")
	URLOfSync = httpx.JoinHostAndPath(HostName, ":443/sync")
)

// Client is an interface to calling the endpoints of the configurator.sys microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  service.Publisher
	host string
}

// NewClient creates a new unicast client to the configurator.sys microservice.
func NewClient(caller service.Publisher) *Client {
	return &Client{
		svc:  caller,
		host: "configurator.sys",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the configurator.sys microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  service.Publisher
	host string
}

// NewMulticastClient creates a new multicast client to the configurator.sys microservice.
func NewMulticastClient(caller service.Publisher) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "configurator.sys",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// ValuesIn are the input arguments of Values.
type ValuesIn struct {
	Names []string `json:"names"`
}

// ValuesOut are the return values of Values.
type ValuesOut struct {
	Values map[string]string `json:"values"`
}

// ValuesResponse is the response to Values.
type ValuesResponse struct {
	data ValuesOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *ValuesResponse) Get() (values map[string]string, err error) {
	values = _out.data.Values
	err = _out.err
	return
}

/*
Values returns the values associated with the specified config property names for the caller microservice.
*/
func (_c *MulticastClient) Values(ctx context.Context, names []string, _options ...pub.Option) <-chan *ValuesResponse {
	method := `*`
	if method == "*" {
		method = "POST"
	}
	_in := ValuesIn{
		names,
	}
	_opts := []pub.Option{
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/values`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *ValuesResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r ValuesResponse
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
	}()
	return _res
}

// RefreshIn are the input arguments of Refresh.
type RefreshIn struct {
}

// RefreshOut are the return values of Refresh.
type RefreshOut struct {
}

// RefreshResponse is the response to Refresh.
type RefreshResponse struct {
	data RefreshOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *RefreshResponse) Get() (err error) {
	err = _out.err
	return
}

/*
Refresh tells all microservices to contact the configurator and refresh their configs.
An error is returned if any of the values sent to the microservices fails validation.
*/
func (_c *MulticastClient) Refresh(ctx context.Context, _options ...pub.Option) <-chan *RefreshResponse {
	method := `*`
	if method == "*" {
		method = "POST"
	}
	_in := RefreshIn{
	}
	_opts := []pub.Option{
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/refresh`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *RefreshResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r RefreshResponse
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
	}()
	return _res
}

// SyncIn are the input arguments of Sync.
type SyncIn struct {
	Timestamp time.Time `json:"timestamp"`
	Values map[string]map[string]string `json:"values"`
}

// SyncOut are the return values of Sync.
type SyncOut struct {
}

// SyncResponse is the response to Sync.
type SyncResponse struct {
	data SyncOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *SyncResponse) Get() (err error) {
	err = _out.err
	return
}

/*
Sync is used to synchronize values among replica peers of the configurator.
*/
func (_c *MulticastClient) Sync(ctx context.Context, timestamp time.Time, values map[string]map[string]string, _options ...pub.Option) <-chan *SyncResponse {
	method := `*`
	if method == "*" {
		method = "POST"
	}
	_in := SyncIn{
		timestamp,
		values,
	}
	_opts := []pub.Option{
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/sync`)),
		pub.Body(_in),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *SyncResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r SyncResponse
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
	}()
	return _res
}

/*
Values returns the values associated with the specified config property names for the caller microservice.
*/
func (_c *Client) Values(ctx context.Context, names []string) (values map[string]string, err error) {
	method := `*`
	if method == "" || method == "*" {
		method = "POST"
	}
	_in := ValuesIn{
		names,
	}
	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/values`)),
		pub.Body(_in),
	)
	if _err != nil {
		err = _err // No trace
		return
	}
	var _out ValuesOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	values = _out.Values
	return
}

/*
Refresh tells all microservices to contact the configurator and refresh their configs.
An error is returned if any of the values sent to the microservices fails validation.
*/
func (_c *Client) Refresh(ctx context.Context) (err error) {
	method := `*`
	if method == "" || method == "*" {
		method = "POST"
	}
	_in := RefreshIn{
	}
	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/refresh`)),
		pub.Body(_in),
	)
	if _err != nil {
		err = _err // No trace
		return
	}
	var _out RefreshOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	return
}

/*
Sync is used to synchronize values among replica peers of the configurator.
*/
func (_c *Client) Sync(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error) {
	method := `*`
	if method == "" || method == "*" {
		method = "POST"
	}
	_in := SyncIn{
		timestamp,
		values,
	}
	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method(method),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/sync`)),
		pub.Body(_in),
	)
	if _err != nil {
		err = _err // No trace
		return
	}
	var _out SyncOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	return
}
