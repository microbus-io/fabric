// Code generated by Microbus. DO NOT EDIT.

package messaging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/messaging/messagingapi"
)

var (
	_ bytes.Buffer
	_ context.Context
	_ fmt.Stringer
	_ io.Reader
	_ *http.Request
	_ os.File
	_ time.Time
	_ strings.Builder
	_ *httpx.BodyReader
	_ pub.Option
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *messagingapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the messaging.example microservice being tested
	Svc *Service
)

func TestMain(m *testing.M) {
	var code int

	// Initialize the application
	err := func() error {
		App = application.NewTesting()
		Svc = NewService().(*Service)
		err := Initialize()
		if err != nil {
			return err
		}
		err = App.Startup()
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "--- FAIL: %+v\n", err)
		code = 19
	}

	// Run the tests
	if err == nil {
		code = m.Run()
	}

	// Terminate the app
	err = func() error {
		var lastErr error
		err = Terminate()
		if err != nil {
			lastErr = err
		}
		err := App.Shutdown()
		if err != nil {
			lastErr = err
		}
		return lastErr
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "--- FAIL: %+v\n", err)
	}

	os.Exit(code)
}

// Context creates a new context for a test.
func Context() context.Context {
	return context.Background()
}

type WebOption func(req *pub.Request) error

// GET sets the method of the request.
func GET() WebOption {
	return WebOption(pub.Method("GET"))
}

// DELETE sets the method of the request.
func DELETE() WebOption {
	return WebOption(pub.Method("DELETE"))
}

// HEAD sets the method of the request.
func HEAD() WebOption {
	return WebOption(pub.Method("HEAD"))
}

// POST sets the method and body of the request.
func POST(body any) WebOption {
	return func(req *pub.Request) error {
		pub.Method("POST")(req)
		return pub.Body(body)(req)
	}
}

// PUT sets the method and body of the request.
func PUT(body any) WebOption {
	return func(req *pub.Request) error {
		pub.Method("PUT")(req)
		return pub.Body(body)(req)
	}
}

// PATCH sets the method and body of the request.
func PATCH(body any) WebOption {
	return func(req *pub.Request) error {
		pub.Method("PATCH")(req)
		return pub.Body(body)(req)
	}
}

// Method sets the method of the request.
func Method(method string) WebOption {
	return WebOption(pub.Method(method))
}

// Header add the header to the request.
// The same header may have multiple values.
func Header(name string, value string) WebOption {
	return WebOption(pub.Header(name, value))
}

// QueryArg adds the query argument to the request.
// The same argument may have multiple values.
func QueryArg(name string, value any) WebOption {
	return WebOption(pub.QueryArg(name, value))
}

// Query adds the escaped query arguments to the request.
// The same argument may have multiple values.
func Query(escapedQueryArgs string) WebOption {
	return WebOption(pub.Query(escapedQueryArgs))
}

// Body sets the body of the request.
// Arguments of type io.Reader, []byte and string are serialized in binary form.
// url.Values is serialized as form data.
// All other types are serialized as JSON.
func Body(body any) WebOption {
	return WebOption(pub.Body(body))
}

// ContentType sets the Content-Type header.
func ContentType(contentType string) WebOption {
	return WebOption(pub.ContentType(contentType))
}

// HomeTestCase assists in asserting against the results of executing Home.
type HomeTestCase struct {
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *HomeTestCase) StatusOK(t *testing.T) *HomeTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *HomeTestCase) StatusCode(t *testing.T, statusCode int) *HomeTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *HomeTestCase) BodyContains(t *testing.T, bodyContains any) *HomeTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyContains.(type) {
		case []byte:
			assert.True(t, bytes.Contains(body, v), `"%v" does not contain "%v"`, body, v)
		case string:
			assert.True(t, bytes.Contains(body, []byte(v)), `"%s" does not contain "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.True(t, bytes.Contains(body, []byte(vv)), `"%s" does not contain "%s"`, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *HomeTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *HomeTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyNotContains.(type) {
		case []byte:
			assert.False(t, bytes.Contains(body, v), `"%v" contains "%v"`, body, v)
		case string:
			assert.False(t, bytes.Contains(body, []byte(v)), `"%s" contains "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.False(t, bytes.Contains(body, []byte(vv)), `"%s" contains "%s"`, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *HomeTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *HomeTestCase {
	if assert.NoError(t, tc.err) {
		assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
	}
	return tc
}

// Error asserts an error.
func (tc *HomeTestCase) Error(t *testing.T, errContains string) *HomeTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *HomeTestCase) NoError(t *testing.T) *HomeTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Assert asserts using a provided function.
func (tc *HomeTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) {
	asserter(t, tc.res, tc.err)
}

// Get returns the result of executing Home.
func (tc *HomeTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// Home executes the web handler and returns a corresponding test case.
func Home(ctx context.Context, options ...WebOption) *HomeTestCase {
	tc := &HomeTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("messaging.example", `:443/home`)),
	}
	for _, opt := range options {
		pubOptions = append(pubOptions, pub.Option(opt))
	}
	req, err := pub.NewRequest(pubOptions...)
	if err != nil {
		panic(err)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		panic(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	r := httpReq.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	tc.err = utils.CatchPanic(func () error {
		return Svc.Home(w, r)
	})
	tc.res = w.Result()
	return tc
}

// NoQueueTestCase assists in asserting against the results of executing NoQueue.
type NoQueueTestCase struct {
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *NoQueueTestCase) StatusOK(t *testing.T) *NoQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *NoQueueTestCase) StatusCode(t *testing.T, statusCode int) *NoQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *NoQueueTestCase) BodyContains(t *testing.T, bodyContains any) *NoQueueTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyContains.(type) {
		case []byte:
			assert.True(t, bytes.Contains(body, v), `"%v" does not contain "%v"`, body, v)
		case string:
			assert.True(t, bytes.Contains(body, []byte(v)), `"%s" does not contain "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.True(t, bytes.Contains(body, []byte(vv)), `"%s" does not contain "%s"`, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *NoQueueTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *NoQueueTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyNotContains.(type) {
		case []byte:
			assert.False(t, bytes.Contains(body, v), `"%v" contains "%v"`, body, v)
		case string:
			assert.False(t, bytes.Contains(body, []byte(v)), `"%s" contains "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.False(t, bytes.Contains(body, []byte(vv)), `"%s" contains "%s"`, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *NoQueueTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *NoQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
	}
	return tc
}

// Error asserts an error.
func (tc *NoQueueTestCase) Error(t *testing.T, errContains string) *NoQueueTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *NoQueueTestCase) NoError(t *testing.T) *NoQueueTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Assert asserts using a provided function.
func (tc *NoQueueTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) {
	asserter(t, tc.res, tc.err)
}

// Get returns the result of executing NoQueue.
func (tc *NoQueueTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// NoQueue executes the web handler and returns a corresponding test case.
func NoQueue(ctx context.Context, options ...WebOption) *NoQueueTestCase {
	tc := &NoQueueTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("messaging.example", `:443/no-queue`)),
	}
	for _, opt := range options {
		pubOptions = append(pubOptions, pub.Option(opt))
	}
	req, err := pub.NewRequest(pubOptions...)
	if err != nil {
		panic(err)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		panic(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	r := httpReq.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	tc.err = utils.CatchPanic(func () error {
		return Svc.NoQueue(w, r)
	})
	tc.res = w.Result()
	return tc
}

// DefaultQueueTestCase assists in asserting against the results of executing DefaultQueue.
type DefaultQueueTestCase struct {
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *DefaultQueueTestCase) StatusOK(t *testing.T) *DefaultQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *DefaultQueueTestCase) StatusCode(t *testing.T, statusCode int) *DefaultQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *DefaultQueueTestCase) BodyContains(t *testing.T, bodyContains any) *DefaultQueueTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyContains.(type) {
		case []byte:
			assert.True(t, bytes.Contains(body, v), `"%v" does not contain "%v"`, body, v)
		case string:
			assert.True(t, bytes.Contains(body, []byte(v)), `"%s" does not contain "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.True(t, bytes.Contains(body, []byte(vv)), `"%s" does not contain "%s"`, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *DefaultQueueTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *DefaultQueueTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyNotContains.(type) {
		case []byte:
			assert.False(t, bytes.Contains(body, v), `"%v" contains "%v"`, body, v)
		case string:
			assert.False(t, bytes.Contains(body, []byte(v)), `"%s" contains "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.False(t, bytes.Contains(body, []byte(vv)), `"%s" contains "%s"`, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *DefaultQueueTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *DefaultQueueTestCase {
	if assert.NoError(t, tc.err) {
		assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
	}
	return tc
}

// Error asserts an error.
func (tc *DefaultQueueTestCase) Error(t *testing.T, errContains string) *DefaultQueueTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *DefaultQueueTestCase) NoError(t *testing.T) *DefaultQueueTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Assert asserts using a provided function.
func (tc *DefaultQueueTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) {
	asserter(t, tc.res, tc.err)
}

// Get returns the result of executing DefaultQueue.
func (tc *DefaultQueueTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// DefaultQueue executes the web handler and returns a corresponding test case.
func DefaultQueue(ctx context.Context, options ...WebOption) *DefaultQueueTestCase {
	tc := &DefaultQueueTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("messaging.example", `:443/default-queue`)),
	}
	for _, opt := range options {
		pubOptions = append(pubOptions, pub.Option(opt))
	}
	req, err := pub.NewRequest(pubOptions...)
	if err != nil {
		panic(err)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		panic(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	r := httpReq.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	tc.err = utils.CatchPanic(func () error {
		return Svc.DefaultQueue(w, r)
	})
	tc.res = w.Result()
	return tc
}

// CacheLoadTestCase assists in asserting against the results of executing CacheLoad.
type CacheLoadTestCase struct {
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *CacheLoadTestCase) StatusOK(t *testing.T) *CacheLoadTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *CacheLoadTestCase) StatusCode(t *testing.T, statusCode int) *CacheLoadTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *CacheLoadTestCase) BodyContains(t *testing.T, bodyContains any) *CacheLoadTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyContains.(type) {
		case []byte:
			assert.True(t, bytes.Contains(body, v), `"%v" does not contain "%v"`, body, v)
		case string:
			assert.True(t, bytes.Contains(body, []byte(v)), `"%s" does not contain "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.True(t, bytes.Contains(body, []byte(vv)), `"%s" does not contain "%s"`, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *CacheLoadTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *CacheLoadTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyNotContains.(type) {
		case []byte:
			assert.False(t, bytes.Contains(body, v), `"%v" contains "%v"`, body, v)
		case string:
			assert.False(t, bytes.Contains(body, []byte(v)), `"%s" contains "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.False(t, bytes.Contains(body, []byte(vv)), `"%s" contains "%s"`, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *CacheLoadTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *CacheLoadTestCase {
	if assert.NoError(t, tc.err) {
		assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
	}
	return tc
}

// Error asserts an error.
func (tc *CacheLoadTestCase) Error(t *testing.T, errContains string) *CacheLoadTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *CacheLoadTestCase) NoError(t *testing.T) *CacheLoadTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Assert asserts using a provided function.
func (tc *CacheLoadTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) {
	asserter(t, tc.res, tc.err)
}

// Get returns the result of executing CacheLoad.
func (tc *CacheLoadTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// CacheLoad executes the web handler and returns a corresponding test case.
func CacheLoad(ctx context.Context, options ...WebOption) *CacheLoadTestCase {
	tc := &CacheLoadTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("messaging.example", `:443/cache-load`)),
	}
	for _, opt := range options {
		pubOptions = append(pubOptions, pub.Option(opt))
	}
	req, err := pub.NewRequest(pubOptions...)
	if err != nil {
		panic(err)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		panic(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	r := httpReq.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	tc.err = utils.CatchPanic(func () error {
		return Svc.CacheLoad(w, r)
	})
	tc.res = w.Result()
	return tc
}

// CacheStoreTestCase assists in asserting against the results of executing CacheStore.
type CacheStoreTestCase struct {
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *CacheStoreTestCase) StatusOK(t *testing.T) *CacheStoreTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *CacheStoreTestCase) StatusCode(t *testing.T, statusCode int) *CacheStoreTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *CacheStoreTestCase) BodyContains(t *testing.T, bodyContains any) *CacheStoreTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyContains.(type) {
		case []byte:
			assert.True(t, bytes.Contains(body, v), `"%v" does not contain "%v"`, body, v)
		case string:
			assert.True(t, bytes.Contains(body, []byte(v)), `"%s" does not contain "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.True(t, bytes.Contains(body, []byte(vv)), `"%s" does not contain "%s"`, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *CacheStoreTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *CacheStoreTestCase {
	if assert.NoError(t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := bodyNotContains.(type) {
		case []byte:
			assert.False(t, bytes.Contains(body, v), `"%v" contains "%v"`, body, v)
		case string:
			assert.False(t, bytes.Contains(body, []byte(v)), `"%s" contains "%s"`, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.False(t, bytes.Contains(body, []byte(vv)), `"%s" contains "%s"`, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *CacheStoreTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *CacheStoreTestCase {
	if assert.NoError(t, tc.err) {
		assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
	}
	return tc
}

// Error asserts an error.
func (tc *CacheStoreTestCase) Error(t *testing.T, errContains string) *CacheStoreTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *CacheStoreTestCase) NoError(t *testing.T) *CacheStoreTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Assert asserts using a provided function.
func (tc *CacheStoreTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) {
	asserter(t, tc.res, tc.err)
}

// Get returns the result of executing CacheStore.
func (tc *CacheStoreTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// CacheStore executes the web handler and returns a corresponding test case.
func CacheStore(ctx context.Context, options ...WebOption) *CacheStoreTestCase {
	tc := &CacheStoreTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("messaging.example", `:443/cache-store`)),
	}
	for _, opt := range options {
		pubOptions = append(pubOptions, pub.Option(opt))
	}
	req, err := pub.NewRequest(pubOptions...)
	if err != nil {
		panic(err)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		panic(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	r := httpReq.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	tc.err = utils.CatchPanic(func () error {
		return Svc.CacheStore(w, r)
	})
	tc.res = w.Result()
	return tc
}
