// Code generated by Microbus. DO NOT EDIT.

/*
Package directoryapi implements the public API of the directory.example microservice,
including clients and data structures.

The directory microservice stores personal records in a MySQL database.
*/
package directoryapi

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

// The default host name addressed by the clients is directory.example.
const HostName = "directory.example"

// Service is an interface abstraction of a microservice used by the client.
// The connector implements this interface.
type Service interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
	Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
}

// Client is an interface to calling the endpoints of the directory.example microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the directory.example microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "directory.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient is an interface to calling the endpoints of the directory.example microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the directory.example microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "directory.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}

// CreateIn are the input arguments of Create.
type CreateIn struct {
	Person *Person `json:"person"`
}

// CreateOut are the return values of Create.
type CreateOut struct {
	Created *Person `json:"created"`
}

// CreateResponse is the response to Create.
type CreateResponse struct {
	data CreateOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *CreateResponse) Get() (created *Person, err error) {
	created = _out.data.Created
	err = _out.err
	return
}

/*
Create registers the person in the directory.
*/
func (_c *MulticastClient) Create(ctx context.Context, person *Person, _options ...pub.Option) <-chan *CreateResponse {
	_in := CreateIn{
		person,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *CreateResponse, 1)
		_res <- &CreateResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/create`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *CreateResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r CreateResponse
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

// LoadIn are the input arguments of Load.
type LoadIn struct {
	Key PersonKey `json:"key"`
}

// LoadOut are the return values of Load.
type LoadOut struct {
	Person *Person `json:"person"`
	Ok bool `json:"ok"`
}

// LoadResponse is the response to Load.
type LoadResponse struct {
	data LoadOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *LoadResponse) Get() (person *Person, ok bool, err error) {
	person = _out.data.Person
	ok = _out.data.Ok
	err = _out.err
	return
}

/*
Load looks up a person in the directory.
*/
func (_c *MulticastClient) Load(ctx context.Context, key PersonKey, _options ...pub.Option) <-chan *LoadResponse {
	_in := LoadIn{
		key,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *LoadResponse, 1)
		_res <- &LoadResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/load`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *LoadResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r LoadResponse
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

// DeleteIn are the input arguments of Delete.
type DeleteIn struct {
	Key PersonKey `json:"key"`
}

// DeleteOut are the return values of Delete.
type DeleteOut struct {
	Ok bool `json:"ok"`
}

// DeleteResponse is the response to Delete.
type DeleteResponse struct {
	data DeleteOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *DeleteResponse) Get() (ok bool, err error) {
	ok = _out.data.Ok
	err = _out.err
	return
}

/*
Delete removes a person from the directory.
*/
func (_c *MulticastClient) Delete(ctx context.Context, key PersonKey, _options ...pub.Option) <-chan *DeleteResponse {
	_in := DeleteIn{
		key,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *DeleteResponse, 1)
		_res <- &DeleteResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/delete`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *DeleteResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r DeleteResponse
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

// UpdateIn are the input arguments of Update.
type UpdateIn struct {
	Person *Person `json:"person"`
}

// UpdateOut are the return values of Update.
type UpdateOut struct {
	Updated *Person `json:"updated"`
	Ok bool `json:"ok"`
}

// UpdateResponse is the response to Update.
type UpdateResponse struct {
	data UpdateOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *UpdateResponse) Get() (updated *Person, ok bool, err error) {
	updated = _out.data.Updated
	ok = _out.data.Ok
	err = _out.err
	return
}

/*
Update updates the person's data in the directory.
*/
func (_c *MulticastClient) Update(ctx context.Context, person *Person, _options ...pub.Option) <-chan *UpdateResponse {
	_in := UpdateIn{
		person,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *UpdateResponse, 1)
		_res <- &UpdateResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/update`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *UpdateResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r UpdateResponse
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

// LoadByEmailIn are the input arguments of LoadByEmail.
type LoadByEmailIn struct {
	Email string `json:"email"`
}

// LoadByEmailOut are the return values of LoadByEmail.
type LoadByEmailOut struct {
	Person *Person `json:"person"`
	Ok bool `json:"ok"`
}

// LoadByEmailResponse is the response to LoadByEmail.
type LoadByEmailResponse struct {
	data LoadByEmailOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *LoadByEmailResponse) Get() (person *Person, ok bool, err error) {
	person = _out.data.Person
	ok = _out.data.Ok
	err = _out.err
	return
}

/*
LoadByEmail looks up a person in the directory by their email.
*/
func (_c *MulticastClient) LoadByEmail(ctx context.Context, email string, _options ...pub.Option) <-chan *LoadByEmailResponse {
	_in := LoadByEmailIn{
		email,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *LoadByEmailResponse, 1)
		_res <- &LoadByEmailResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/load-by-email`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *LoadByEmailResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r LoadByEmailResponse
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

// ListIn are the input arguments of List.
type ListIn struct {
}

// ListOut are the return values of List.
type ListOut struct {
	Keys []PersonKey `json:"keys"`
}

// ListResponse is the response to List.
type ListResponse struct {
	data ListOut
	HTTPResponse *http.Response
	err error
}

// Get retrieves the return values.
func (_out *ListResponse) Get() (keys []PersonKey, err error) {
	keys = _out.data.Keys
	err = _out.err
	return
}

/*
List returns the keys of all the persons in the directory.
*/
func (_c *MulticastClient) List(ctx context.Context, _options ...pub.Option) <-chan *ListResponse {
	_in := ListIn{
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		_res := make(chan *ListResponse, 1)
		_res <- &ListResponse{err: errors.Trace(_err)}
		close(_res)
		return _res
	}

	_opts := []pub.Option{
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/list`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	}
	_opts = append(_opts, _options...)
	_ch := _c.svc.Publish(ctx, _opts...)

	_res := make(chan *ListResponse, cap(_ch))
	go func() {
		for _i := range _ch {
			var _r ListResponse
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
Create registers the person in the directory.
*/
func (_c *Client) Create(ctx context.Context, person *Person) (created *Person, err error) {
	_in := CreateIn{
		person,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/create`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out CreateOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	created = _out.Created
	return
}

/*
Load looks up a person in the directory.
*/
func (_c *Client) Load(ctx context.Context, key PersonKey) (person *Person, ok bool, err error) {
	_in := LoadIn{
		key,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/load`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out LoadOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	person = _out.Person
	ok = _out.Ok
	return
}

/*
Delete removes a person from the directory.
*/
func (_c *Client) Delete(ctx context.Context, key PersonKey) (ok bool, err error) {
	_in := DeleteIn{
		key,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/delete`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out DeleteOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	ok = _out.Ok
	return
}

/*
Update updates the person's data in the directory.
*/
func (_c *Client) Update(ctx context.Context, person *Person) (updated *Person, ok bool, err error) {
	_in := UpdateIn{
		person,
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/update`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out UpdateOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	updated = _out.Updated
	ok = _out.Ok
	return
}

/*
LoadByEmail looks up a person in the directory by their email.
*/
func (_c *Client) LoadByEmail(ctx context.Context, email string) (person *Person, ok bool, err error) {
	_in := LoadByEmailIn{
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
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/load-by-email`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out LoadByEmailOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	person = _out.Person
	ok = _out.Ok
	return
}

/*
List returns the keys of all the persons in the directory.
*/
func (_c *Client) List(ctx context.Context) (keys []PersonKey, err error) {
	_in := ListIn{
	}
	_body, _err := json.Marshal(_in)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}

	_httpRes, _err := _c.svc.Request(
		ctx,
		pub.Method("POST"),
		pub.URL(httpx.JoinHostAndPath(_c.host, `:443/list`)),
		pub.Body(_body),
		pub.Header("Content-Type", "application/json"),
	)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	var _out ListOut
	_err = json.NewDecoder(_httpRes.Body).Decode(&_out)
	if _err != nil {
		err = errors.Trace(_err)
		return
	}
	keys = _out.Keys
	return
}
