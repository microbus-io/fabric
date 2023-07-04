/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

package eventsink

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
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
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
	_ *connector.Connector
	_ *errors.TracedError
	_ frame.Frame
	_ *httpx.BodyReader
	_ pub.Option
	_ *shardedsql.DB
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *eventsinkapi.Client
)

var (
	sequence int
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the eventsink.example microservice being tested
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
func Context(t *testing.T) context.Context {
	return context.Background()
}

// RegisteredTestCase assists in asserting against the results of executing Registered.
type RegisteredTestCase struct {
	_t *testing.T
	_testName string
	emails []string
	err error
}

// Name sets a name to the test case.
func (tc *RegisteredTestCase) Name(testName string) *RegisteredTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *RegisteredTestCase) Expect(emails []string) *RegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, emails, tc.emails)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *RegisteredTestCase) Error(errContains string) *RegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *RegisteredTestCase) ErrorCode(statusCode int) *RegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *RegisteredTestCase) NoError() *RegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *RegisteredTestCase) Assert(asserter func(t *testing.T, emails []string, err error)) *RegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.emails, tc.err)
	})
	return tc
}

// Get returns the result of executing Registered.
func (tc *RegisteredTestCase) Get() (emails []string, err error) {
	return tc.emails, tc.err
}

// Registered executes the function and returns a corresponding test case.
func Registered(t *testing.T, ctx context.Context) *RegisteredTestCase {
	tc := &RegisteredTestCase{_t: t}
	tc.err = utils.CatchPanic(func() error {
		tc.emails, tc.err = Svc.Registered(ctx)
		return tc.err
	})
	return tc
}

// OnAllowRegisterTestCase assists in asserting against the results of executing OnAllowRegister.
type OnAllowRegisterTestCase struct {
	_t *testing.T
	_testName string
	allow bool
	err error
}

// Name sets a name to the test case.
func (tc *OnAllowRegisterTestCase) Name(testName string) *OnAllowRegisterTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *OnAllowRegisterTestCase) Expect(allow bool) *OnAllowRegisterTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, allow, tc.allow)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *OnAllowRegisterTestCase) Error(errContains string) *OnAllowRegisterTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *OnAllowRegisterTestCase) ErrorCode(statusCode int) *OnAllowRegisterTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *OnAllowRegisterTestCase) NoError() *OnAllowRegisterTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *OnAllowRegisterTestCase) Assert(asserter func(t *testing.T, allow bool, err error)) *OnAllowRegisterTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.allow, tc.err)
	})
	return tc
}

// Get returns the result of executing OnAllowRegister.
func (tc *OnAllowRegisterTestCase) Get() (allow bool, err error) {
	return tc.allow, tc.err
}

// OnAllowRegister executes the function and returns a corresponding test case.
func OnAllowRegister(t *testing.T, ctx context.Context, email string) *OnAllowRegisterTestCase {
	tc := &OnAllowRegisterTestCase{_t: t}
	tc.err = utils.CatchPanic(func() error {
		tc.allow, tc.err = Svc.OnAllowRegister(ctx, email)
		return tc.err
	})
	return tc
}

// OnRegisteredTestCase assists in asserting against the results of executing OnRegistered.
type OnRegisteredTestCase struct {
	_t *testing.T
	_testName string
	err error
}

// Name sets a name to the test case.
func (tc *OnRegisteredTestCase) Name(testName string) *OnRegisteredTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *OnRegisteredTestCase) Expect() *OnRegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Error asserts an error.
func (tc *OnRegisteredTestCase) Error(errContains string) *OnRegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *OnRegisteredTestCase) ErrorCode(statusCode int) *OnRegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *OnRegisteredTestCase) NoError() *OnRegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *OnRegisteredTestCase) Assert(asserter func(t *testing.T, err error)) *OnRegisteredTestCase {
	tc._t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.err)
	})
	return tc
}

// Get returns the result of executing OnRegistered.
func (tc *OnRegisteredTestCase) Get() (err error) {
	return tc.err
}

// OnRegistered executes the function and returns a corresponding test case.
func OnRegistered(t *testing.T, ctx context.Context, email string) *OnRegisteredTestCase {
	tc := &OnRegisteredTestCase{_t: t}
	tc.err = utils.CatchPanic(func() error {
		tc.err = Svc.OnRegistered(ctx, email)
		return tc.err
	})
	return tc
}
