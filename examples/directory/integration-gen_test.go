/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

package directory

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

	"github.com/andybalholm/cascadia"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
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
	_ cascadia.Sel
	_ *connector.Connector
	_ *errors.TracedError
	_ frame.Frame
	_ *httpx.BodyReader
	_ pub.Option
	_ rand.Void
	_ utils.SyncMap[string, string]
	_ assert.TestingT
	_ *html.Node
	_ *directoryapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the directory.example microservice being tested
	Svc *Service
)

func TestMain(m *testing.M) {
	var code int

	// Initialize the application
	err := func() error {
		var err error
		App = application.NewTesting()
		Svc = NewService()
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
	return frame.ContextWithFrame(context.Background())
}

// CreateTestCase assists in asserting against the results of executing Create.
type CreateTestCase struct {
	_t *testing.T
	_dur time.Duration
	key directoryapi.PersonKey
	err error
}

// Expect asserts no error and exact return values.
func (_tc *CreateTestCase) Expect(key directoryapi.PersonKey) *CreateTestCase {
	if assert.NoError(_tc._t, _tc.err) {
		assert.Equal(_tc._t, key, _tc.key)
	}
	return _tc
}

// Error asserts an error.
func (tc *CreateTestCase) Error(errContains string) *CreateTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *CreateTestCase) ErrorCode(statusCode int) *CreateTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *CreateTestCase) NoError() *CreateTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *CreateTestCase) CompletedIn(threshold time.Duration) *CreateTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *CreateTestCase) Assert(asserter func(t *testing.T, key directoryapi.PersonKey, err error)) *CreateTestCase {
	asserter(tc._t, tc.key, tc.err)
	return tc
}

// Get returns the result of executing Create.
func (tc *CreateTestCase) Get() (key directoryapi.PersonKey, err error) {
	return tc.key, tc.err
}

// Create executes the function and returns a corresponding test case.
func Create(t *testing.T, ctx context.Context, httpRequestBody *directoryapi.Person) *CreateTestCase {
	tc := &CreateTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.key, tc.err = Svc.Create(ctx, httpRequestBody)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// LoadTestCase assists in asserting against the results of executing Load.
type LoadTestCase struct {
	_t *testing.T
	_dur time.Duration
	httpResponseBody *directoryapi.Person
	err error
}

// Expect asserts no error and exact return values.
func (_tc *LoadTestCase) Expect(httpResponseBody *directoryapi.Person) *LoadTestCase {
	if assert.NoError(_tc._t, _tc.err) {
		assert.Equal(_tc._t, httpResponseBody, _tc.httpResponseBody)
	}
	return _tc
}

// Error asserts an error.
func (tc *LoadTestCase) Error(errContains string) *LoadTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *LoadTestCase) ErrorCode(statusCode int) *LoadTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *LoadTestCase) NoError() *LoadTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *LoadTestCase) CompletedIn(threshold time.Duration) *LoadTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *LoadTestCase) Assert(asserter func(t *testing.T, httpResponseBody *directoryapi.Person, err error)) *LoadTestCase {
	asserter(tc._t, tc.httpResponseBody, tc.err)
	return tc
}

// Get returns the result of executing Load.
func (tc *LoadTestCase) Get() (httpResponseBody *directoryapi.Person, err error) {
	return tc.httpResponseBody, tc.err
}

// Load executes the function and returns a corresponding test case.
func Load(t *testing.T, ctx context.Context, key directoryapi.PersonKey) *LoadTestCase {
	tc := &LoadTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.httpResponseBody, tc.err = Svc.Load(ctx, key)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// DeleteTestCase assists in asserting against the results of executing Delete.
type DeleteTestCase struct {
	_t *testing.T
	_dur time.Duration
	err error
}

// Expect asserts no error and exact return values.
func (_tc *DeleteTestCase) Expect() *DeleteTestCase {
	assert.NoError(_tc._t, _tc.err)
	return _tc
}

// Error asserts an error.
func (tc *DeleteTestCase) Error(errContains string) *DeleteTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *DeleteTestCase) ErrorCode(statusCode int) *DeleteTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *DeleteTestCase) NoError() *DeleteTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *DeleteTestCase) CompletedIn(threshold time.Duration) *DeleteTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *DeleteTestCase) Assert(asserter func(t *testing.T, err error)) *DeleteTestCase {
	asserter(tc._t, tc.err)
	return tc
}

// Get returns the result of executing Delete.
func (tc *DeleteTestCase) Get() (err error) {
	return tc.err
}

// Delete executes the function and returns a corresponding test case.
func Delete(t *testing.T, ctx context.Context, key directoryapi.PersonKey) *DeleteTestCase {
	tc := &DeleteTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.err = Svc.Delete(ctx, key)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// UpdateTestCase assists in asserting against the results of executing Update.
type UpdateTestCase struct {
	_t *testing.T
	_dur time.Duration
	err error
}

// Expect asserts no error and exact return values.
func (_tc *UpdateTestCase) Expect() *UpdateTestCase {
	assert.NoError(_tc._t, _tc.err)
	return _tc
}

// Error asserts an error.
func (tc *UpdateTestCase) Error(errContains string) *UpdateTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *UpdateTestCase) ErrorCode(statusCode int) *UpdateTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *UpdateTestCase) NoError() *UpdateTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *UpdateTestCase) CompletedIn(threshold time.Duration) *UpdateTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *UpdateTestCase) Assert(asserter func(t *testing.T, err error)) *UpdateTestCase {
	asserter(tc._t, tc.err)
	return tc
}

// Get returns the result of executing Update.
func (tc *UpdateTestCase) Get() (err error) {
	return tc.err
}

// Update executes the function and returns a corresponding test case.
func Update(t *testing.T, ctx context.Context, key directoryapi.PersonKey, httpRequestBody *directoryapi.Person) *UpdateTestCase {
	tc := &UpdateTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.err = Svc.Update(ctx, key, httpRequestBody)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// LoadByEmailTestCase assists in asserting against the results of executing LoadByEmail.
type LoadByEmailTestCase struct {
	_t *testing.T
	_dur time.Duration
	httpResponseBody *directoryapi.Person
	err error
}

// Expect asserts no error and exact return values.
func (_tc *LoadByEmailTestCase) Expect(httpResponseBody *directoryapi.Person) *LoadByEmailTestCase {
	if assert.NoError(_tc._t, _tc.err) {
		assert.Equal(_tc._t, httpResponseBody, _tc.httpResponseBody)
	}
	return _tc
}

// Error asserts an error.
func (tc *LoadByEmailTestCase) Error(errContains string) *LoadByEmailTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *LoadByEmailTestCase) ErrorCode(statusCode int) *LoadByEmailTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *LoadByEmailTestCase) NoError() *LoadByEmailTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *LoadByEmailTestCase) CompletedIn(threshold time.Duration) *LoadByEmailTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *LoadByEmailTestCase) Assert(asserter func(t *testing.T, httpResponseBody *directoryapi.Person, err error)) *LoadByEmailTestCase {
	asserter(tc._t, tc.httpResponseBody, tc.err)
	return tc
}

// Get returns the result of executing LoadByEmail.
func (tc *LoadByEmailTestCase) Get() (httpResponseBody *directoryapi.Person, err error) {
	return tc.httpResponseBody, tc.err
}

// LoadByEmail executes the function and returns a corresponding test case.
func LoadByEmail(t *testing.T, ctx context.Context, email string) *LoadByEmailTestCase {
	tc := &LoadByEmailTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.httpResponseBody, tc.err = Svc.LoadByEmail(ctx, email)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// ListTestCase assists in asserting against the results of executing List.
type ListTestCase struct {
	_t *testing.T
	_dur time.Duration
	httpResponseBody []directoryapi.PersonKey
	err error
}

// Expect asserts no error and exact return values.
func (_tc *ListTestCase) Expect(httpResponseBody []directoryapi.PersonKey) *ListTestCase {
	if assert.NoError(_tc._t, _tc.err) {
		assert.Equal(_tc._t, httpResponseBody, _tc.httpResponseBody)
	}
	return _tc
}

// Error asserts an error.
func (tc *ListTestCase) Error(errContains string) *ListTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Contains(tc._t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *ListTestCase) ErrorCode(statusCode int) *ListTestCase {
	if assert.Error(tc._t, tc.err) {
		assert.Equal(tc._t, statusCode, errors.StatusCode(tc.err))
	}
	return tc
}

// NoError asserts no error.
func (tc *ListTestCase) NoError() *ListTestCase {
	assert.NoError(tc._t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *ListTestCase) CompletedIn(threshold time.Duration) *ListTestCase {
	assert.LessOrEqual(tc._t, tc._dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *ListTestCase) Assert(asserter func(t *testing.T, httpResponseBody []directoryapi.PersonKey, err error)) *ListTestCase {
	asserter(tc._t, tc.httpResponseBody, tc.err)
	return tc
}

// Get returns the result of executing List.
func (tc *ListTestCase) Get() (httpResponseBody []directoryapi.PersonKey, err error) {
	return tc.httpResponseBody, tc.err
}

// List executes the function and returns a corresponding test case.
func List(t *testing.T, ctx context.Context) *ListTestCase {
	tc := &ListTestCase{_t: t}
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		tc.httpResponseBody, tc.err = Svc.List(ctx)
		return tc.err
	})
	tc._dur = time.Since(t0)
	return tc
}

// WebUITestCase assists in asserting against the results of executing WebUI.
type WebUITestCase struct {
	t *testing.T
	dur time.Duration
	res *http.Response
	err error
}

// StatusOK asserts no error and a status code 200.
func (tc *WebUITestCase) StatusOK() *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Equal(tc.t, tc.res.StatusCode, http.StatusOK)
	}
	return tc
}

// StatusCode asserts no error and a status code.
func (tc *WebUITestCase) StatusCode(statusCode int) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Equal(tc.t, tc.res.StatusCode, statusCode)
	}
	return tc
}

// BodyContains asserts no error and that the response body contains the string or byte array value.
func (tc *WebUITestCase) BodyContains(value any) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := value.(type) {
		case []byte:
			assert.True(tc.t, bytes.Contains(body, v), "%v does not contain %v", body, v)
		case string:
			assert.Contains(tc.t, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.Contains(tc.t, string(body), vv)
		}
	}
	return tc
}

// BodyNotContains asserts no error and that the response body does not contain the string or byte array value.
func (tc *WebUITestCase) BodyNotContains(value any) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		switch v := value.(type) {
		case []byte:
			assert.False(tc.t, bytes.Contains(body, v), "%v contains %v", body, v)
		case string:
			assert.NotContains(tc.t, string(body), v)
		default:
			vv := fmt.Sprintf("%v", v)
			assert.NotContains(tc.t, string(body), vv)
		}
	}
	return tc
}

// HeaderContains asserts no error and that the named header contains the value.
func (tc *WebUITestCase) HeaderContains(headerName string, value string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Contains(tc.t, tc.res.Header.Get(headerName), value)
	}
	return tc
}

// HeaderNotContains asserts no error and that the named header does not contain a string.
func (tc *WebUITestCase) HeaderNotContains(headerName string, value string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.NotContains(tc.t, tc.res.Header.Get(headerName), value)
	}
	return tc
}

// HeaderEqual asserts no error and that the named header matches the value.
func (tc *WebUITestCase) HeaderEqual(headerName string, value string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Equal(tc.t, value, tc.res.Header.Get(headerName))
	}
	return tc
}

// HeaderNotEqual asserts no error and that the named header does not matche the value.
func (tc *WebUITestCase) HeaderNotEqual(headerName string, value string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.NotEqual(tc.t, value, tc.res.Header.Get(headerName))
	}
	return tc
}

// HeaderExists asserts no error and that the named header exists.
func (tc *WebUITestCase) HeaderExists(headerName string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.NotEmpty(tc.t, tc.res.Header.Values(headerName), "Header %s does not exist", headerName)
	}
	return tc
}

// HeaderNotExists asserts no error and that the named header does not exists.
func (tc *WebUITestCase) HeaderNotExists(headerName string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Empty(tc.t, tc.res.Header.Values(headerName), "Header %s exists", headerName)
	}
	return tc
}

// ContentType asserts no error and that the Content-Type header matches the expected value.
func (tc *WebUITestCase) ContentType(expected string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		assert.Equal(tc.t, expected, tc.res.Header.Get("Content-Type"))
	}
	return tc
}

/*
TagExists asserts no error and that the at least one tag matches the CSS selector query.

Examples:

	TagExists(`TR > TD > A.expandable[href]`)
	TagExists(`DIV#main_panel`)
	TagExists(`TR TD INPUT[name="x"]`)
*/
func (tc *WebUITestCase) TagExists(cssSelectorQuery string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		assert.NotEmpty(tc.t, matches, "Found no tags matching %s", cssSelectorQuery)
	}
	return tc
}

/*
TagNotExists asserts no error and that the no tag matches the CSS selector query.

Example:

	TagNotExists(`TR > TD > A.expandable[href]`)
	TagNotExists(`DIV#main_panel`)
	TagNotExists(`TR TD INPUT[name="x"]`)
*/
func (tc *WebUITestCase) TagNotExists(cssSelectorQuery string) *WebUITestCase {
	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		assert.Empty(tc.t, matches, "Found %d tag(s) matching %s", len(matches), cssSelectorQuery)
	}
	return tc
}

/*
TagEqual asserts no error and that the at least one of the tags matching the CSS selector query
either contains the exact text itself or has a descendant that does.

Example:

	TagEqual("TR > TD > A.expandable[href]", "Expand")
	TagEqual("DIV#main_panel > SELECT > OPTION", "Red")
*/
func (tc *WebUITestCase) TagEqual(cssSelectorQuery string, value string) *WebUITestCase {
	var textMatches func(n *html.Node) bool
	textMatches = func(n *html.Node) bool {
		for x := n.FirstChild; x != nil; x = x.NextSibling {
			if x.Data == value || textMatches(x) {
				return true
			}
		}
		return false
	}

	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		if !assert.NotEmpty(tc.t, matches, "Selector %s does not match any tags", cssSelectorQuery) {
			return tc
		}
		if value == "" {
			return tc
		}
		found := false
		for _, match := range matches {
			if textMatches(match) {
				found = true
				break
			}
		}
		assert.True(tc.t, found, "No tag matching %s contains %s", cssSelectorQuery, value)
	}
	return tc
}

/*
TagContains asserts no error and that the at least one of the tags matching the CSS selector query
either contains the text itself or has a descendant that does.

Example:

	TagContains("TR > TD > A.expandable[href]", "Expand")
	TagContains("DIV#main_panel > SELECT > OPTION", "Red")
*/
func (tc *WebUITestCase) TagContains(cssSelectorQuery string, value string) *WebUITestCase {
	var textMatches func(n *html.Node) bool
	textMatches = func(n *html.Node) bool {
		for x := n.FirstChild; x != nil; x = x.NextSibling {
			if strings.Contains(x.Data, value) || textMatches(x) {
				return true
			}
		}
		return false
	}

	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		if !assert.NotEmpty(tc.t, matches, "Selector %s does not match any tags", cssSelectorQuery) {
			return tc
		}
		if value == "" {
			return tc
		}
		found := false
		for _, match := range matches {
			if textMatches(match) {
				found = true
				break
			}
		}
		assert.True(tc.t, found, "No tag matching %s contains %s", cssSelectorQuery, value)
	}
	return tc
}

/*
TagNotEqual asserts no error and that there is no tag matching the CSS selector that
either contains the exact text itself or has a descendant that does.

Example:

	TagNotEqual("TR > TD > A[href]", "Harry Potter")
	TagNotEqual("DIV#main_panel > SELECT > OPTION", "Red")
*/
func (tc *WebUITestCase) TagNotEqual(cssSelectorQuery string, value string) *WebUITestCase {
	var textMatches func(n *html.Node) bool
	textMatches = func(n *html.Node) bool {
		for x := n.FirstChild; x != nil; x = x.NextSibling {
			if x.Data == value || textMatches(x) {
				return true
			}
		}
		return false
	}

	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		if len(matches) == 0 {
			return tc
		}
		if !assert.NotEmpty(tc.t, value, "Found tag matching %s", cssSelectorQuery) {
			return tc
		}
		found := false
		for _, match := range matches {
			if textMatches(match) {
				found = true
				break
			}
		}
		assert.False(tc.t, found, "Found tag matching %s that contains %s", cssSelectorQuery, value)
	}
	return tc
}

/*
TagNotContains asserts no error and that there is no tag matching the CSS selector that
either contains the text itself or has a descendant that does.

Example:

	TagNotContains("TR > TD > A[href]", "Harry Potter")
	TagNotContains("DIV#main_panel > SELECT > OPTION", "Red")
*/
func (tc *WebUITestCase) TagNotContains(cssSelectorQuery string, value string) *WebUITestCase {
	var textMatches func(n *html.Node) bool
	textMatches = func(n *html.Node) bool {
		for x := n.FirstChild; x != nil; x = x.NextSibling {
			if strings.Contains(x.Data, value) || textMatches(x) {
				return true
			}
		}
		return false
	}

	if assert.NoError(tc.t, tc.err) {
		selector, err := cascadia.Compile(cssSelectorQuery)
		if !assert.NoError(tc.t, err, "Invalid selector %s", cssSelectorQuery) {
			return tc
		}
		body := tc.res.Body.(*httpx.BodyReader).Bytes()
		doc, err := html.Parse(bytes.NewReader(body))
		if !assert.NoError(tc.t, err, "Failed to parse HTML") {
			return tc
		}
		matches := selector.MatchAll(doc)
		if len(matches) == 0 {
			return tc
		}
		if !assert.NotEmpty(tc.t, value, "Found tag matching %s", cssSelectorQuery) {
			return tc
		}
		found := false
		for _, match := range matches {
			if textMatches(match) {
				found = true
				break
			}
		}
		assert.False(tc.t, found, "Found tag matching %s that contains %s", cssSelectorQuery, value)
	}
	return tc
}

// Error asserts an error.
func (tc *WebUITestCase) Error(errContains string) *WebUITestCase {
	if assert.Error(tc.t, tc.err) {
		assert.Contains(tc.t, tc.err.Error(), errContains)
	}
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *WebUITestCase) ErrorCode(statusCode int) *WebUITestCase {
	if assert.Error(tc.t, tc.err) {
		assert.Equal(tc.t, statusCode, errors.Convert(tc.err).StatusCode)
	}
	return tc
}

// NoError asserts no error.
func (tc *WebUITestCase) NoError() *WebUITestCase {
	assert.NoError(tc.t, tc.err)
	return tc
}

// CompletedIn checks that the duration of the operation is less than or equal the threshold.
func (tc *WebUITestCase) CompletedIn(threshold time.Duration) *WebUITestCase {
	assert.LessOrEqual(tc.t, tc.dur, threshold)
	return tc
}

// Assert asserts using a provided function.
func (tc *WebUITestCase) Assert(asserter func(t *testing.T, res *http.Response, err error)) *WebUITestCase {
	asserter(tc.t, tc.res, tc.err)
	return tc
}

// Get returns the result of executing WebUI.
func (tc *WebUITestCase) Get() (res *http.Response, err error) {
	return tc.res, tc.err
}

/*
WebUI_Get performs a GET request to the WebUI endpoint.

WebUI provides a form for making web requests to the CRUD endpoints.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func WebUI_Get(t *testing.T, ctx context.Context, url string) *WebUITestCase {
	tc := &WebUITestCase{t: t}
	var err error
	url, err = httpx.ResolveURL(directoryapi.URLOfWebUI, url)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	ctx = frame.CloneContext(ctx)
	r = r.WithContext(ctx)
	r.Header = frame.Of(ctx).Header()
	w := httpx.NewResponseRecorder()
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		return Svc.WebUI(w, r)
	})
	tc.dur = time.Since(t0)
	tc.res = w.Result()
	return tc
}

/*
WebUI_Post performs a POST request to the WebUI endpoint.

WebUI provides a form for making web requests to the CRUD endpoints.

If a URL is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
If the body if of type io.Reader, []byte or string, it is serialized in binary form.
If it is of type url.Values, it is serialized as form data. All other types are serialized as JSON.
If a content type is not explicitly provided, an attempt will be made to derive it from the body.
*/
func WebUI_Post(t *testing.T, ctx context.Context, url string, contentType string, body any) *WebUITestCase {
	tc := &WebUITestCase{t: t}
	var err error
	url, err = httpx.ResolveURL(directoryapi.URLOfWebUI, url)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	r, err := httpx.NewRequest("POST", url, nil)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	ctx = frame.CloneContext(ctx)
	r = r.WithContext(ctx)
	r.Header = frame.Of(ctx).Header()
	err = httpx.SetRequestBody(r, body)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	w := httpx.NewResponseRecorder()
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		return Svc.WebUI(w, r)
	})
	tc.dur = time.Since(t0)
	tc.res = w.Result()
	return tc
}

/*
WebUI provides a form for making web requests to the CRUD endpoints.

If a request is not provided, it defaults to the URL of the endpoint. Otherwise, it is resolved relative to the URL of the endpoint.
*/
func WebUI(t *testing.T, r *http.Request) *WebUITestCase {
	tc := &WebUITestCase{t: t}
	var err error
	if r == nil {
		r, err = http.NewRequest(`GET`, "", nil)
		if err != nil {
			tc.err = errors.Trace(err)
			return tc
		}
	}
	u, err := httpx.ResolveURL(directoryapi.URLOfWebUI, r.URL.String())
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	r.URL, err = httpx.ParseURL(u)
	if err != nil {
		tc.err = errors.Trace(err)
		return tc
	}
	for k, vv := range frame.Of(r.Context()).Header() {
		r.Header[k] = vv
	}
	ctx := frame.ContextWithFrameOf(r.Context(), r.Header)
	r = r.WithContext(ctx)
	w := httpx.NewResponseRecorder()
	t0 := time.Now()
	tc.err = utils.CatchPanic(func() error {
		return Svc.WebUI(w, r)
	})
	tc.res = w.Result()
	tc.dur = time.Since(t0)
	return tc
}
