/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

/*
Package controlapi implements the public API of the control.sys microservice,
including clients and data structures.

This microservice is created for the sake of generating the client API for the :888 control subscriptions.
The microservice itself does nothing and should not be included in applications.
*/
package controlapi

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

// Hostname is the default hostname of the microservice: control.sys.
const Hostname = "control.sys"

// Fully-qualified URLs of the microservice's endpoints.
var (
	URLOfPing = httpx.JoinHostAndPath(Hostname, `:888/ping`)
	URLOfConfigRefresh = httpx.JoinHostAndPath(Hostname, `:888/config-refresh`)
	URLOfTrace = httpx.JoinHostAndPath(Hostname, `:888/trace`)
)

// Client is an interface to calling the endpoints of the control.sys microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  service.Publisher
	host string
}

// NewClient creates a new unicast client to the control.sys microservice.
func NewClient(caller service.Publisher) *Client {
	return &Client{
		svc:  caller,
		host: "control.sys",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the control.sys microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  service.Publisher
	host string
}

// NewMulticastClient creates a new multicast client to the control.sys microservice.
func NewMulticastClient(caller service.Publisher) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "control.sys",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// PingIn are the input arguments of Ping.
type PingIn struct {
}

// PingOut are the return values of Ping.
type PingOut struct {
	Pong int `json:"pong"`
}

// PingResponse is the response to Ping.
type PingResponse struct {
	data PingOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *PingResponse) Get() (pong int, err error) {
	pong = _out.data.Pong
	err = _out.err
	return
}

/*
Ping responds to the message with a pong.
*/
func (_c *MulticastClient) Ping(ctx context.Context) <-chan *PingResponse {
	_url := httpx.JoinHostAndPath(_c.host, `:888/ping`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
	})
	_in := PingIn{
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

	_res := make(chan *PingResponse, cap(_ch))
	for _i := range _ch {
		var _r PingResponse
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
Ping responds to the message with a pong.
*/
func (_c *Client) Ping(ctx context.Context) (pong int, err error) {
	var _err error
	_url := httpx.JoinHostAndPath(_c.host, `:888/ping`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
	})
	_in := PingIn{
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
	var _out PingOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	pong = _out.Pong
	return
}

// ConfigRefreshIn are the input arguments of ConfigRefresh.
type ConfigRefreshIn struct {
}

// ConfigRefreshOut are the return values of ConfigRefresh.
type ConfigRefreshOut struct {
}

// ConfigRefreshResponse is the response to ConfigRefresh.
type ConfigRefreshResponse struct {
	data ConfigRefreshOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *ConfigRefreshResponse) Get() (err error) {
	err = _out.err
	return
}

/*
ConfigRefresh pulls the latest config values from the configurator service.
*/
func (_c *MulticastClient) ConfigRefresh(ctx context.Context) <-chan *ConfigRefreshResponse {
	_url := httpx.JoinHostAndPath(_c.host, `:888/config-refresh`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
	})
	_in := ConfigRefreshIn{
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

	_res := make(chan *ConfigRefreshResponse, cap(_ch))
	for _i := range _ch {
		var _r ConfigRefreshResponse
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
ConfigRefresh pulls the latest config values from the configurator service.
*/
func (_c *Client) ConfigRefresh(ctx context.Context) (err error) {
	var _err error
	_url := httpx.JoinHostAndPath(_c.host, `:888/config-refresh`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
	})
	_in := ConfigRefreshIn{
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
	var _out ConfigRefreshOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	return
}

// TraceIn are the input arguments of Trace.
type TraceIn struct {
	ID string `json:"id"`
}

// TraceOut are the return values of Trace.
type TraceOut struct {
}

// TraceResponse is the response to Trace.
type TraceResponse struct {
	data TraceOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *TraceResponse) Get() (err error) {
	err = _out.err
	return
}

/*
Trace forces exporting the indicated tracing span.
*/
func (_c *MulticastClient) Trace(ctx context.Context, id string) <-chan *TraceResponse {
	_url := httpx.JoinHostAndPath(_c.host, `:888/trace`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
		`id`: id,
	})
	_in := TraceIn{
		id,
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

	_res := make(chan *TraceResponse, cap(_ch))
	for _i := range _ch {
		var _r TraceResponse
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
Trace forces exporting the indicated tracing span.
*/
func (_c *Client) Trace(ctx context.Context, id string) (err error) {
	var _err error
	_url := httpx.JoinHostAndPath(_c.host, `:888/trace`)
	_url = httpx.InsertPathArguments(_url, httpx.QArgs{
		`id`: id,
	})
	_in := TraceIn{
		id,
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
	var _out TraceOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	return
}
