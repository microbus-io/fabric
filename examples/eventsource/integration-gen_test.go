// Code generated by Microbus. DO NOT EDIT.

package eventsource

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
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
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
	_ *errors.TracedError
	_ *httpx.BodyReader
	_ pub.Option
	_ *shardedsql.DB
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *eventsourceapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the eventsource.example microservice being tested
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

// RegisterTestCase assists in asserting against the results of executing Register.
type RegisterTestCase struct {
	_testName string
	allowed bool
	err error
}

// Name sets a name to the test case.
func (tc *RegisterTestCase) Name(testName string) *RegisterTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *RegisterTestCase) Expect(t *testing.T, allowed bool) *RegisterTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, allowed, tc.allowed)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *RegisterTestCase) Error(t *testing.T, errContains string) *RegisterTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *RegisterTestCase) ErrorCode(t *testing.T, statusCode int) *RegisterTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *RegisterTestCase) NoError(t *testing.T) *RegisterTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *RegisterTestCase) Assert(t *testing.T, asserter func(t *testing.T, allowed bool, err error)) *RegisterTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.allowed, tc.err)
	})
	return tc
}

// Get returns the result of executing Register.
func (tc *RegisterTestCase) Get() (allowed bool, err error) {
	return tc.allowed, tc.err
}

// Register executes the function and returns a corresponding test case.
func Register(ctx context.Context, email string) *RegisterTestCase {
	tc := &RegisterTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.allowed, tc.err = Svc.Register(ctx, email)
		return tc.err
	})
	return tc
}
