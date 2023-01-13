// Code generated by Microbus. DO NOT EDIT.

package hello

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
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/hello/helloapi"
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
	_ *shardedsql.DB
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *helloapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the hello.example microservice being tested
	Svc *Service
)

func TestMain(m *testing.M) {
	var code int

	// Initialize the application
	err := func() error {
		var err error
		App = application.NewTesting()
		Svc = NewService().(*Service)
		err = Initialize()
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
		var err error
		var lastErr error
		err = Terminate()
		if err != nil {
			lastErr = err
		}
		err = App.Shutdown()
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

// HelloTestCase assists in asserting against the results of executing Hello.
type HelloTestCase struct {
	testName string
	res *http.Response
	err error
}

// Name sets a name to the test case.
func (tc *HelloTestCase) Name(testName string) *HelloTestCase {
	tc.testName = testName
	return tc
}

// StatusOK asserts no error and a status code 200.
func (tc *HelloTestCase) StatusOK(t *testing.T) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, http.StatusOK)
		}
	})
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *HelloTestCase) StatusCode(t *testing.T, statusCode int) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, statusCode)
		}
	})
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *HelloTestCase) BodyContains(t *testing.T, bodyContains any) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *HelloTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *HelloTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *HelloTestCase) Error(t *testing.T, errContains string) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *HelloTestCase) NoError(t *testing.T) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *HelloTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) *HelloTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.res, tc.err)
	})
	return tc
}

// Get returns the result of executing Hello.
func (tc *HelloTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// Hello executes the web handler and returns a corresponding test case.
func Hello(ctx context.Context, options ...WebOption) *HelloTestCase {
	tc := &HelloTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("hello.example", `:443/hello`)),
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
		return Svc.Hello(w, r)
	})
	tc.res = w.Result()
	return tc
}

// EchoTestCase assists in asserting against the results of executing Echo.
type EchoTestCase struct {
	testName string
	res *http.Response
	err error
}

// Name sets a name to the test case.
func (tc *EchoTestCase) Name(testName string) *EchoTestCase {
	tc.testName = testName
	return tc
}

// StatusOK asserts no error and a status code 200.
func (tc *EchoTestCase) StatusOK(t *testing.T) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, http.StatusOK)
		}
	})
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *EchoTestCase) StatusCode(t *testing.T, statusCode int) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, statusCode)
		}
	})
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *EchoTestCase) BodyContains(t *testing.T, bodyContains any) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *EchoTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *EchoTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *EchoTestCase) Error(t *testing.T, errContains string) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *EchoTestCase) NoError(t *testing.T) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *EchoTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) *EchoTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.res, tc.err)
	})
	return tc
}

// Get returns the result of executing Echo.
func (tc *EchoTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// Echo executes the web handler and returns a corresponding test case.
func Echo(ctx context.Context, options ...WebOption) *EchoTestCase {
	tc := &EchoTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("hello.example", `:443/echo`)),
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
		return Svc.Echo(w, r)
	})
	tc.res = w.Result()
	return tc
}

// PingTestCase assists in asserting against the results of executing Ping.
type PingTestCase struct {
	testName string
	res *http.Response
	err error
}

// Name sets a name to the test case.
func (tc *PingTestCase) Name(testName string) *PingTestCase {
	tc.testName = testName
	return tc
}

// StatusOK asserts no error and a status code 200.
func (tc *PingTestCase) StatusOK(t *testing.T) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, http.StatusOK)
		}
	})
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *PingTestCase) StatusCode(t *testing.T, statusCode int) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, statusCode)
		}
	})
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *PingTestCase) BodyContains(t *testing.T, bodyContains any) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *PingTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *PingTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *PingTestCase) Error(t *testing.T, errContains string) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *PingTestCase) NoError(t *testing.T) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *PingTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) *PingTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.res, tc.err)
	})
	return tc
}

// Get returns the result of executing Ping.
func (tc *PingTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// Ping executes the web handler and returns a corresponding test case.
func Ping(ctx context.Context, options ...WebOption) *PingTestCase {
	tc := &PingTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("hello.example", `:443/ping`)),
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
		return Svc.Ping(w, r)
	})
	tc.res = w.Result()
	return tc
}

// CalculatorTestCase assists in asserting against the results of executing Calculator.
type CalculatorTestCase struct {
	testName string
	res *http.Response
	err error
}

// Name sets a name to the test case.
func (tc *CalculatorTestCase) Name(testName string) *CalculatorTestCase {
	tc.testName = testName
	return tc
}

// StatusOK asserts no error and a status code 200.
func (tc *CalculatorTestCase) StatusOK(t *testing.T) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, http.StatusOK)
		}
	})
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *CalculatorTestCase) StatusCode(t *testing.T, statusCode int) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, statusCode)
		}
	})
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *CalculatorTestCase) BodyContains(t *testing.T, bodyContains any) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *CalculatorTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *CalculatorTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *CalculatorTestCase) Error(t *testing.T, errContains string) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *CalculatorTestCase) NoError(t *testing.T) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *CalculatorTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) *CalculatorTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.res, tc.err)
	})
	return tc
}

// Get returns the result of executing Calculator.
func (tc *CalculatorTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// Calculator executes the web handler and returns a corresponding test case.
func Calculator(ctx context.Context, options ...WebOption) *CalculatorTestCase {
	tc := &CalculatorTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("hello.example", `:443/calculator`)),
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
		return Svc.Calculator(w, r)
	})
	tc.res = w.Result()
	return tc
}

// BusJPEGTestCase assists in asserting against the results of executing BusJPEG.
type BusJPEGTestCase struct {
	testName string
	res *http.Response
	err error
}

// Name sets a name to the test case.
func (tc *BusJPEGTestCase) Name(testName string) *BusJPEGTestCase {
	tc.testName = testName
	return tc
}

// StatusOK asserts no error and a status code 200.
func (tc *BusJPEGTestCase) StatusOK(t *testing.T) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, http.StatusOK)
		}
	})
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *BusJPEGTestCase) StatusCode(t *testing.T, statusCode int) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, tc.res.StatusCode, statusCode)
		}
	})
	return tc
}

// BodyContains asserts no error and that the response contains a string or byte array.
func (tc *BusJPEGTestCase) BodyContains(t *testing.T, bodyContains any) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// BodyNotContains asserts no error and that the response does not contain a string or byte array.
func (tc *BusJPEGTestCase) BodyNotContains(t *testing.T, bodyNotContains any) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
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
	})
	return tc
}

// HeaderContains asserts no error and that the named header contains a string.
func (tc *BusJPEGTestCase) HeaderContains(t *testing.T, headerName string, valueContains string) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.True(t, strings.Contains(tc.res.Header.Get(headerName), valueContains), `header "%s: %s" does not contain "%s"`, headerName, tc.res.Header.Get(headerName), valueContains)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *BusJPEGTestCase) Error(t *testing.T, errContains string) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *BusJPEGTestCase) NoError(t *testing.T) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *BusJPEGTestCase) Assert(t *testing.T, asserter func(t *testing.T, res *http.Response, err error)) *BusJPEGTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.res, tc.err)
	})
	return tc
}

// Get returns the result of executing BusJPEG.
func (tc *BusJPEGTestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

// BusJPEG executes the web handler and returns a corresponding test case.
func BusJPEG(ctx context.Context, options ...WebOption) *BusJPEGTestCase {
	tc := &BusJPEGTestCase{}
	pubOptions := []pub.Option{
		pub.URL(httpx.JoinHostAndPath("hello.example", `:443/bus.jpeg`)),
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
		return Svc.BusJPEG(w, r)
	})
	tc.res = w.Result()
	return tc
}

// TickTockTestCase assists in asserting against the results of executing TickTock.
type TickTockTestCase struct {
	testName string
	err error
}

// Name sets a name to the test case.
func (tc *TickTockTestCase) Name(testName string) *TickTockTestCase {
	tc.testName = testName
	return tc
}

// Error asserts an error.
func (tc *TickTockTestCase) Error(t *testing.T, errContains string) *TickTockTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *TickTockTestCase) NoError(t *testing.T) *TickTockTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *TickTockTestCase) Assert(t *testing.T, asserter func(t *testing.T, err error)) *TickTockTestCase {
	t.Run(tc.testName, func(t *testing.T) {
		asserter(t, tc.err)
	})
	return tc
}

// Get returns the result of executing TickTock.
func (tc *TickTockTestCase) Get() (err error) {
	return tc.err
}

// TickTock executes the ticker and returns a corresponding test case.
func TickTock(ctx context.Context) *TickTockTestCase {
	tc := &TickTockTestCase{}
	tc.err = utils.CatchPanic(func () error {
		return Svc.TickTock(ctx)
	})
	return tc
}
