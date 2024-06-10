/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/codegen/tester/testerapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ strings.Builder
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ testerapi.Client
)

// Mock is a mockable version of the codegen.test microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockStringCut func(ctx context.Context, s string, sep string) (before string, after string, found bool, err error)
	mockPointDistance func(ctx context.Context, p1 testerapi.XYCoord, p2 *testerapi.XYCoord) (d float64, err error)
	mockShiftPoint func(ctx context.Context, p *testerapi.XYCoord, x float64, y float64) (shifted *testerapi.XYCoord, err error)
	mockSubArrayRange func(ctx context.Context, httpRequestBody []int, min int, max int) (httpResponseBody []int, httpStatusCode int, err error)
	mockSumTwoIntegers func(ctx context.Context, x int, y int) (sum int, httpStatusCode int, err error)
	mockFunctionPathArguments func(ctx context.Context, named string, path2 string, suffix string) (joined string, err error)
	mockNonStringPathArguments func(ctx context.Context, named int, path2 bool, suffix float64) (joined string, err error)
	mockUnnamedFunctionPathArguments func(ctx context.Context, path1 string, path2 string, path3 string) (joined string, err error)
	mockPathArgumentsPriority func(ctx context.Context, foo string) (echo string, err error)
	mockEcho func(w http.ResponseWriter, r *http.Request) (err error)
	mockWebPathArguments func(w http.ResponseWriter, r *http.Request) (err error)
	mockUnnamedWebPathArguments func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("codegen.test"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The tester is used to test the code generator's functions.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Functions
	svc.Subscribe(`ANY`, `:443/string-cut`, svc.doStringCut)
	svc.Subscribe(`GET`, `:443/point-distance`, svc.doPointDistance)
	svc.Subscribe(`ANY`, `:443/shift-point`, svc.doShiftPoint)
	svc.Subscribe(`ANY`, `:443/sub-array-range/{max}`, svc.doSubArrayRange)
	svc.Subscribe(`ANY`, `:443/sum-two-integers`, svc.doSumTwoIntegers)
	svc.Subscribe(`GET`, `:443/function-path-arguments/fixed/{named}/{}/{suffix+}`, svc.doFunctionPathArguments)
	svc.Subscribe(`GET`, `:443/non-string-path-arguments/fixed/{named}/{}/{suffix+}`, svc.doNonStringPathArguments)
	svc.Subscribe(`GET`, `:443/unnamed-function-path-arguments/{}/foo/{}/bar/{+}`, svc.doUnnamedFunctionPathArguments)
	svc.Subscribe(`ANY`, `:443/path-arguments-priority/{foo}`, svc.doPathArgumentsPriority)

	// Webs
	svc.Subscribe(`ANY`, `:443/echo`, svc.doEcho)
	svc.Subscribe(`ANY`, `:443/web-path-arguments/fixed/{named}/{}/{suffix+}`, svc.doWebPathArguments)
	svc.Subscribe(`GET`, `:443/unnamed-web-path-arguments/{}/foo/{}/bar/{+}`, svc.doUnnamedWebPathArguments)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doStringCut handles marshaling for the StringCut function.
func (svc *Mock) doStringCut(w http.ResponseWriter, r *http.Request) error {
	if svc.mockStringCut == nil {
		return errors.New("mocked endpoint 'StringCut' not implemented")
	}
	var i testerapi.StringCutIn
	var o testerapi.StringCutOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/string-cut`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/string-cut`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Before, o.After, o.Found, err = svc.mockStringCut(
		r.Context(),
		i.S,
		i.Sep,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockStringCut sets up a mock handler for the StringCut function.
func (svc *Mock) MockStringCut(handler func(ctx context.Context, s string, sep string) (before string, after string, found bool, err error)) *Mock {
	svc.mockStringCut = handler
	return svc
}

// doPointDistance handles marshaling for the PointDistance function.
func (svc *Mock) doPointDistance(w http.ResponseWriter, r *http.Request) error {
	if svc.mockPointDistance == nil {
		return errors.New("mocked endpoint 'PointDistance' not implemented")
	}
	var i testerapi.PointDistanceIn
	var o testerapi.PointDistanceOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/point-distance`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/point-distance`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.D, err = svc.mockPointDistance(
		r.Context(),
		i.P1,
		i.P2,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockPointDistance sets up a mock handler for the PointDistance function.
func (svc *Mock) MockPointDistance(handler func(ctx context.Context, p1 testerapi.XYCoord, p2 *testerapi.XYCoord) (d float64, err error)) *Mock {
	svc.mockPointDistance = handler
	return svc
}

// doShiftPoint handles marshaling for the ShiftPoint function.
func (svc *Mock) doShiftPoint(w http.ResponseWriter, r *http.Request) error {
	if svc.mockShiftPoint == nil {
		return errors.New("mocked endpoint 'ShiftPoint' not implemented")
	}
	var i testerapi.ShiftPointIn
	var o testerapi.ShiftPointOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/shift-point`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/shift-point`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Shifted, err = svc.mockShiftPoint(
		r.Context(),
		i.P,
		i.X,
		i.Y,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockShiftPoint sets up a mock handler for the ShiftPoint function.
func (svc *Mock) MockShiftPoint(handler func(ctx context.Context, p *testerapi.XYCoord, x float64, y float64) (shifted *testerapi.XYCoord, err error)) *Mock {
	svc.mockShiftPoint = handler
	return svc
}

// doSubArrayRange handles marshaling for the SubArrayRange function.
func (svc *Mock) doSubArrayRange(w http.ResponseWriter, r *http.Request) error {
	if svc.mockSubArrayRange == nil {
		return errors.New("mocked endpoint 'SubArrayRange' not implemented")
	}
	var i testerapi.SubArrayRangeIn
	var o testerapi.SubArrayRangeOut
	err := httpx.ParseRequestBody(r, &i.HTTPRequestBody)
	if err != nil {
		return errors.Trace(err)
	}
	err = httpx.DecodeDeepObject(r.URL.Query(), &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/sub-array-range/{max}`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/sub-array-range/{max}`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.HTTPResponseBody, o.HTTPStatusCode, err = svc.mockSubArrayRange(
		r.Context(),
		i.HTTPRequestBody,
		i.Min,
		i.Max,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(o.HTTPStatusCode)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o.HTTPResponseBody)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockSubArrayRange sets up a mock handler for the SubArrayRange function.
func (svc *Mock) MockSubArrayRange(handler func(ctx context.Context, httpRequestBody []int, min int, max int) (httpResponseBody []int, httpStatusCode int, err error)) *Mock {
	svc.mockSubArrayRange = handler
	return svc
}

// doSumTwoIntegers handles marshaling for the SumTwoIntegers function.
func (svc *Mock) doSumTwoIntegers(w http.ResponseWriter, r *http.Request) error {
	if svc.mockSumTwoIntegers == nil {
		return errors.New("mocked endpoint 'SumTwoIntegers' not implemented")
	}
	var i testerapi.SumTwoIntegersIn
	var o testerapi.SumTwoIntegersOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/sum-two-integers`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/sum-two-integers`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Sum, o.HTTPStatusCode, err = svc.mockSumTwoIntegers(
		r.Context(),
		i.X,
		i.Y,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(o.HTTPStatusCode)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockSumTwoIntegers sets up a mock handler for the SumTwoIntegers function.
func (svc *Mock) MockSumTwoIntegers(handler func(ctx context.Context, x int, y int) (sum int, httpStatusCode int, err error)) *Mock {
	svc.mockSumTwoIntegers = handler
	return svc
}

// doFunctionPathArguments handles marshaling for the FunctionPathArguments function.
func (svc *Mock) doFunctionPathArguments(w http.ResponseWriter, r *http.Request) error {
	if svc.mockFunctionPathArguments == nil {
		return errors.New("mocked endpoint 'FunctionPathArguments' not implemented")
	}
	var i testerapi.FunctionPathArgumentsIn
	var o testerapi.FunctionPathArgumentsOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/function-path-arguments/fixed/{named}/{}/{suffix+}`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/function-path-arguments/fixed/{named}/{}/{suffix+}`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Joined, err = svc.mockFunctionPathArguments(
		r.Context(),
		i.Named,
		i.Path2,
		i.Suffix,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockFunctionPathArguments sets up a mock handler for the FunctionPathArguments function.
func (svc *Mock) MockFunctionPathArguments(handler func(ctx context.Context, named string, path2 string, suffix string) (joined string, err error)) *Mock {
	svc.mockFunctionPathArguments = handler
	return svc
}

// doNonStringPathArguments handles marshaling for the NonStringPathArguments function.
func (svc *Mock) doNonStringPathArguments(w http.ResponseWriter, r *http.Request) error {
	if svc.mockNonStringPathArguments == nil {
		return errors.New("mocked endpoint 'NonStringPathArguments' not implemented")
	}
	var i testerapi.NonStringPathArgumentsIn
	var o testerapi.NonStringPathArgumentsOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/non-string-path-arguments/fixed/{named}/{}/{suffix+}`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/non-string-path-arguments/fixed/{named}/{}/{suffix+}`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Joined, err = svc.mockNonStringPathArguments(
		r.Context(),
		i.Named,
		i.Path2,
		i.Suffix,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockNonStringPathArguments sets up a mock handler for the NonStringPathArguments function.
func (svc *Mock) MockNonStringPathArguments(handler func(ctx context.Context, named int, path2 bool, suffix float64) (joined string, err error)) *Mock {
	svc.mockNonStringPathArguments = handler
	return svc
}

// doUnnamedFunctionPathArguments handles marshaling for the UnnamedFunctionPathArguments function.
func (svc *Mock) doUnnamedFunctionPathArguments(w http.ResponseWriter, r *http.Request) error {
	if svc.mockUnnamedFunctionPathArguments == nil {
		return errors.New("mocked endpoint 'UnnamedFunctionPathArguments' not implemented")
	}
	var i testerapi.UnnamedFunctionPathArgumentsIn
	var o testerapi.UnnamedFunctionPathArgumentsOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/unnamed-function-path-arguments/{}/foo/{}/bar/{+}`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/unnamed-function-path-arguments/{}/foo/{}/bar/{+}`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Joined, err = svc.mockUnnamedFunctionPathArguments(
		r.Context(),
		i.Path1,
		i.Path2,
		i.Path3,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockUnnamedFunctionPathArguments sets up a mock handler for the UnnamedFunctionPathArguments function.
func (svc *Mock) MockUnnamedFunctionPathArguments(handler func(ctx context.Context, path1 string, path2 string, path3 string) (joined string, err error)) *Mock {
	svc.mockUnnamedFunctionPathArguments = handler
	return svc
}

// doPathArgumentsPriority handles marshaling for the PathArgumentsPriority function.
func (svc *Mock) doPathArgumentsPriority(w http.ResponseWriter, r *http.Request) error {
	if svc.mockPathArgumentsPriority == nil {
		return errors.New("mocked endpoint 'PathArgumentsPriority' not implemented")
	}
	var i testerapi.PathArgumentsPriorityIn
	var o testerapi.PathArgumentsPriorityOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/path-arguments-priority/{foo}`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/path-arguments-priority/{foo}`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Echo, err = svc.mockPathArgumentsPriority(
		r.Context(),
		i.Foo,
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockPathArgumentsPriority sets up a mock handler for the PathArgumentsPriority function.
func (svc *Mock) MockPathArgumentsPriority(handler func(ctx context.Context, foo string) (echo string, err error)) *Mock {
	svc.mockPathArgumentsPriority = handler
	return svc
}

// doEcho handles the Echo web handler.
func (svc *Mock) doEcho(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockEcho == nil {
		return errors.New("mocked endpoint 'Echo' not implemented")
	}
	err = svc.mockEcho(w, r)
	return errors.Trace(err)
}

// MockEcho sets up a mock handler for the Echo web handler.
func (svc *Mock) MockEcho(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockEcho = handler
	return svc
}

// doWebPathArguments handles the WebPathArguments web handler.
func (svc *Mock) doWebPathArguments(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockWebPathArguments == nil {
		return errors.New("mocked endpoint 'WebPathArguments' not implemented")
	}
	err = svc.mockWebPathArguments(w, r)
	return errors.Trace(err)
}

// MockWebPathArguments sets up a mock handler for the WebPathArguments web handler.
func (svc *Mock) MockWebPathArguments(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockWebPathArguments = handler
	return svc
}

// doUnnamedWebPathArguments handles the UnnamedWebPathArguments web handler.
func (svc *Mock) doUnnamedWebPathArguments(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockUnnamedWebPathArguments == nil {
		return errors.New("mocked endpoint 'UnnamedWebPathArguments' not implemented")
	}
	err = svc.mockUnnamedWebPathArguments(w, r)
	return errors.Trace(err)
}

// MockUnnamedWebPathArguments sets up a mock handler for the UnnamedWebPathArguments web handler.
func (svc *Mock) MockUnnamedWebPathArguments(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockUnnamedWebPathArguments = handler
	return svc
}
