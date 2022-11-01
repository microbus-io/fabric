// Code generated by Microbus. DO NOT EDIT.

package helloapi

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

const ServiceName = "hello.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Client provides type-safe access to the endpoints of the "hello.example" microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the "hello.example" microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "hello.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient provides type-safe access to the endpoints of the "hello.example" microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the "hello.example" microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "hello.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

/*
Hello prints a greeting.
*/
func (_c *Client) Hello(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/hello`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
Hello prints a greeting.
*/
func (_c *MulticastClient) Hello(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/hello`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
Echo back the incoming request in wire format.
*/
func (_c *Client) Echo(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/echo`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
Echo back the incoming request in wire format.
*/
func (_c *MulticastClient) Echo(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/echo`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
Ping all microservices and list them.
*/
func (_c *Client) Ping(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/ping`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
Ping all microservices and list them.
*/
func (_c *MulticastClient) Ping(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/ping`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.
*/
func (_c *Client) Calculator(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/calculator`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
Calculator renders a UI for a calculator.
The calculation operation is delegated to another microservice in order to demonstrate
a call from one microservice to another.
*/
func (_c *MulticastClient) Calculator(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/calculator`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
BusJPEG serves an image from the embedded resources.
*/
func (_c *Client) BusJPEG(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/bus.jpeg`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
BusJPEG serves an image from the embedded resources.
*/
func (_c *MulticastClient) BusJPEG(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/bus.jpeg`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}
