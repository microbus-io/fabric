// Code generated by Microbus. DO NOT EDIT.

/*
Package calculatorapi implements the public API of the calculator.example microservice,
including clients and data structures.

The Calculator microservice performs simple mathematical operations.
*/
package calculatorapi

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

// The default host name addressed by the clients is calculator.example.
const HostName = "calculator.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Client is an interface to calling the endpoints of the calculator.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the calculator.example microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "calculator.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the calculator.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the calculator.example microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "calculator.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// ArithmeticIn are the input arguments of Arithmetic.
type ArithmeticIn struct {
	X int `json:"x"`
	Op string `json:"op"`
	Y int `json:"y"`
}

// ArithmeticOut are the return values of Arithmetic.
type ArithmeticOut struct {
	XEcho int `json:"xEcho"`
	OpEcho string `json:"opEcho"`
	YEcho int `json:"yEcho"`
	Result int `json:"result"`
}

// ArithmeticResponse is the response to Arithmetic.
type ArithmeticResponse struct {
	data ArithmeticOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *ArithmeticResponse) Get() (xEcho int, opEcho string, yEcho int, result int, err error) {
	xEcho = _out.data.XEcho
	opEcho = _out.data.OpEcho
	yEcho = _out.data.YEcho
	result = _out.data.Result
	err = _out.err
	return
}

/*
Arithmetic perform an arithmetic operation between two integers x and y given an operator op.
*/
func (_c *MulticastClient) Arithmetic(ctx context.Context, x int, op string, y int, _options ...pub.Option) <-chan *ArithmeticResponse {
	_in := ArithmeticIn{
		x,
		op,
		y,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *ArithmeticResponse, 1)
		_res <- &ArithmeticResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/arithmetic`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *ArithmeticResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r ArithmeticResponse
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

// SquareIn are the input arguments of Square.
type SquareIn struct {
	X int `json:"x"`
}

// SquareOut are the return values of Square.
type SquareOut struct {
	XEcho int `json:"xEcho"`
	Result int `json:"result"`
}

// SquareResponse is the response to Square.
type SquareResponse struct {
	data SquareOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *SquareResponse) Get() (xEcho int, result int, err error) {
	xEcho = _out.data.XEcho
	result = _out.data.Result
	err = _out.err
	return
}

/*
Square prints the square of the integer x.
*/
func (_c *MulticastClient) Square(ctx context.Context, x int, _options ...pub.Option) <-chan *SquareResponse {
	_in := SquareIn{
		x,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *SquareResponse, 1)
		_res <- &SquareResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/square`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *SquareResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r SquareResponse
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

// DistanceIn are the input arguments of Distance.
type DistanceIn struct {
	P1 Point `json:"p1"`
	P2 Point `json:"p2"`
}

// DistanceOut are the return values of Distance.
type DistanceOut struct {
	D float64 `json:"d"`
}

// DistanceResponse is the response to Distance.
type DistanceResponse struct {
	data DistanceOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *DistanceResponse) Get() (d float64, err error) {
	d = _out.data.D
	err = _out.err
	return
}

/*
Distance calculates the distance between two points.
It demonstrates the use of the defined type Point.
*/
func (_c *MulticastClient) Distance(ctx context.Context, p1 Point, p2 Point, _options ...pub.Option) <-chan *DistanceResponse {
	_in := DistanceIn{
		p1,
		p2,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *DistanceResponse, 1)
		_res <- &DistanceResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/distance`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *DistanceResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r DistanceResponse
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
Arithmetic perform an arithmetic operation between two integers x and y given an operator op.
*/
func (_c *Client) Arithmetic(ctx context.Context, x int, op string, y int) (xEcho int, opEcho string, yEcho int, result int, err error) {
	_in := ArithmeticIn{
		x,
		op,
		y,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/arithmetic`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out ArithmeticOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	xEcho = _out.XEcho
	opEcho = _out.OpEcho
	yEcho = _out.YEcho
	result = _out.Result
	return
}

/*
Square prints the square of the integer x.
*/
func (_c *Client) Square(ctx context.Context, x int) (xEcho int, result int, err error) {
	_in := SquareIn{
		x,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/square`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out SquareOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	xEcho = _out.XEcho
	result = _out.Result
	return
}

/*
Distance calculates the distance between two points.
It demonstrates the use of the defined type Point.
*/
func (_c *Client) Distance(ctx context.Context, p1 Point, p2 Point) (d float64, err error) {
	_in := DistanceIn{
		p1,
		p2,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(sub.JoinHostAndPath(_c.host, `:443/distance`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out DistanceOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	d = _out.D
	return
}
