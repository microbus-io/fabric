/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

/*
Package helloapi implements the public API of the hello.example microservice,
including clients and data structures.

The Hello microservice demonstrates the various capabilities of a microservice.
*/
package helloapi

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

// Hostname is the default hostname of the microservice: hello.example.
const Hostname = "hello.example"

// Fully-qualified URLs of the microservice's endpoints.
var (
	URLOfHello = httpx.JoinHostAndPath(Hostname, `:443/hello`)
	URLOfEcho = httpx.JoinHostAndPath(Hostname, `:443/echo`)
	URLOfPing = httpx.JoinHostAndPath(Hostname, `:443/ping`)
	URLOfCalculator = httpx.JoinHostAndPath(Hostname, `:443/calculator`)
	URLOfBusJPEG = httpx.JoinHostAndPath(Hostname, `:443/bus.jpeg`)
	URLOfLocalization = httpx.JoinHostAndPath(Hostname, `:443/localization`)
	URLOfRoot = httpx.JoinHostAndPath(Hostname, `//root`)
)

// Client is an interface to calling the endpoints of the hello.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  service.Publisher
	host string
}

// NewClient creates a new unicast client to the hello.example microservice.
func NewClient(caller service.Publisher) *Client {
	return &Client{
		svc:  caller,
		host: "hello.example",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the hello.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  service.Publisher
	host string
}

// NewMulticastClient creates a new multicast client to the hello.example microservice.
func NewMulticastClient(caller service.Publisher) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "hello.example",
	}
}

// ForHost replaces the default hostname of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// errChan returns a response channel with a single error response.
func (_c *MulticastClient) errChan(err error) <-chan *pub.Response {
	ch := make(chan *pub.Response, 1)
	ch <- pub.NewErrorResponse(err)
	close(ch)
	return ch
}

/*
Hello_Get performs a GET request to the Hello endpoint.

Hello prints a greeting.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Hello_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfHello, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Hello_Get performs a GET request to the Hello endpoint.

Hello prints a greeting.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Hello_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfHello, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Hello_Post performs a POST request to the Hello endpoint.

Hello prints a greeting.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Hello_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfHello, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Hello_Post performs a POST request to the Hello endpoint.

Hello prints a greeting.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Hello_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfHello, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Hello prints a greeting.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Hello(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfHello, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Hello prints a greeting.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Hello(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfHello, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
Echo_Get performs a GET request to the Echo endpoint.

Echo back the incoming request in wire format.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Echo_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfEcho, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Echo_Get performs a GET request to the Echo endpoint.

Echo back the incoming request in wire format.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Echo_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfEcho, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Echo_Post performs a POST request to the Echo endpoint.

Echo back the incoming request in wire format.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Echo_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfEcho, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Echo_Post performs a POST request to the Echo endpoint.

Echo back the incoming request in wire format.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Echo_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfEcho, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Echo back the incoming request in wire format.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Echo(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfEcho, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Echo back the incoming request in wire format.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Echo(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfEcho, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
Ping_Get performs a GET request to the Ping endpoint.

Ping all microservices and list them.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Ping_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfPing, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Ping_Get performs a GET request to the Ping endpoint.

Ping all microservices and list them.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Ping_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfPing, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Ping_Post performs a POST request to the Ping endpoint.

Ping all microservices and list them.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Ping_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfPing, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Ping_Post performs a POST request to the Ping endpoint.

Ping all microservices and list them.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Ping_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfPing, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Ping all microservices and list them.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Ping(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfPing, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Ping all microservices and list them.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Ping(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfPing, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
Calculator_Get performs a GET request to the Calculator endpoint.

Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Calculator_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfCalculator, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Calculator_Get performs a GET request to the Calculator endpoint.

Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Calculator_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfCalculator, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Calculator_Post performs a POST request to the Calculator endpoint.

Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Calculator_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfCalculator, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Calculator_Post performs a POST request to the Calculator endpoint.

Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Calculator_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfCalculator, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Calculator(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfCalculator, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Calculator(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfCalculator, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
BusJPEG serves an image from the embedded resources.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) BusJPEG(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfBusJPEG, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(`GET`), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
BusJPEG serves an image from the embedded resources.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) BusJPEG(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfBusJPEG, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(`GET`), pub.URL(url))
}

/*
BusJPEG_Do performs a customized request to the BusJPEG endpoint.

BusJPEG serves an image from the embedded resources.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) BusJPEG_Do(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfBusJPEG, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
BusJPEG_Do performs a customized request to the BusJPEG endpoint.

BusJPEG serves an image from the embedded resources.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) BusJPEG_Do(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfBusJPEG, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
Localization_Get performs a GET request to the Localization endpoint.

Localization prints hello in the language best matching the request's Accept-Language header.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Localization_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfLocalization, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Localization_Get performs a GET request to the Localization endpoint.

Localization prints hello in the language best matching the request's Accept-Language header.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Localization_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfLocalization, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Localization_Post performs a POST request to the Localization endpoint.

Localization prints hello in the language best matching the request's Accept-Language header.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Localization_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfLocalization, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Localization_Post performs a POST request to the Localization endpoint.

Localization prints hello in the language best matching the request's Accept-Language header.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Localization_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfLocalization, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Localization prints hello in the language best matching the request's Accept-Language header.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Localization(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfLocalization, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Localization prints hello in the language best matching the request's Accept-Language header.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Localization(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfLocalization, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}

/*
Root_Get performs a GET request to the Root endpoint.

Root is the top-most root page.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Root_Get(ctx context.Context, url string) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfRoot, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("GET"), pub.URL(url))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Root_Get performs a GET request to the Root endpoint.

Root is the top-most root page.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Root_Get(ctx context.Context, url string) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfRoot, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("GET"), pub.URL(url))
}

/*
Root_Post performs a POST request to the Root endpoint.

Root is the top-most root page.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *Client) Root_Post(ctx context.Context, url string, contentType string, body any) (res *http.Response, err error) {
	url, err = httpx.ResolveURL(URLOfRoot, url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Root_Post performs a POST request to the Root endpoint.

Root is the top-most root page.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func (_c *MulticastClient) Root_Post(ctx context.Context, url string, contentType string, body any) <-chan *pub.Response {
	var err error
	url, err = httpx.ResolveURL(URLOfRoot, url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method("POST"), pub.URL(url), pub.ContentType(contentType), pub.Body(body))
}

/*
Root is the top-most root page.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *Client) Root(ctx context.Context, r *http.Request) (res *http.Response, err error) {
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	url, err := httpx.ResolveURL(URLOfRoot, r.URL.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err = _c.svc.Request(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
	if err != nil {
		return nil, err // No trace
	}
	return res, err
}

/*
Root is the top-most root page.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func (_c *MulticastClient) Root(ctx context.Context, r *http.Request) <-chan *pub.Response {
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			return _c.errChan(errors.Trace(err))
		}
	}
	url, err := httpx.ResolveURL(URLOfRoot, r.URL.String())
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	url, err = httpx.FillPathArguments(url)
	if err != nil {
		return _c.errChan(errors.Trace(err))
	}
	return _c.svc.Publish(ctx, pub.Method(r.Method), pub.URL(url), pub.CopyHeaders(r.Header), pub.Body(r.Body))
}
