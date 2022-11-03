// Code generated by Microbus. DO NOT EDIT.

package messagingapi

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

const ServiceName = "messaging.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Client provides type-safe access to the endpoints of the messaging.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the messaging.example microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "messaging.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient provides type-safe access to the endpoints of the messaging.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the messaging.example microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "messaging.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

/*
Home demonstrates making requests using multicast and unicast request/response patterns.
*/
func (_c *Client) Home(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/home`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
Home demonstrates making requests using multicast and unicast request/response patterns.
*/
func (_c *MulticastClient) Home(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/home`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
NoQueue demonstrates how the NoQueue subscription option is used to create
a multicast request/response communication pattern.
All instances of this microservice will respond to each request.
*/
func (_c *Client) NoQueue(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/no-queue`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
NoQueue demonstrates how the NoQueue subscription option is used to create
a multicast request/response communication pattern.
All instances of this microservice will respond to each request.
*/
func (_c *MulticastClient) NoQueue(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/no-queue`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
DefaultQueue demonstrates how the DefaultQueue subscription option is used to create
a unicast request/response communication pattern.
Only one of the instances of this microservice will respond to each request.
*/
func (_c *Client) DefaultQueue(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/default-queue`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
DefaultQueue demonstrates how the DefaultQueue subscription option is used to create
a unicast request/response communication pattern.
Only one of the instances of this microservice will respond to each request.
*/
func (_c *MulticastClient) DefaultQueue(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/default-queue`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
CacheLoad looks up an element in the distributed cache of the microservice.
*/
func (_c *Client) CacheLoad(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/cache-load`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
CacheLoad looks up an element in the distributed cache of the microservice.
*/
func (_c *MulticastClient) CacheLoad(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/cache-load`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}

/*
CacheStore stores an element in the distributed cache of the microservice.
*/
func (_c *Client) CacheStore(ctx context.Context, options ...pub.Option) (res *http.Response, err error) {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/cache-store`)),
	}
	opts = append(opts, options...)
	res, err = _c.svc.Request(ctx, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, err
}

/*
CacheStore stores an element in the distributed cache of the microservice.
*/
func (_c *MulticastClient) CacheStore(ctx context.Context, options ...pub.Option) <-chan *pub.Response {
	opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `/cache-store`)),
	}
	opts = append(opts, options...)
	return _c.svc.Publish(ctx, opts...)
}
