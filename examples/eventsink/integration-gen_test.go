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
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
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
	_ *httpx.BodyReader
	_ pub.Option
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *eventsinkapi.Client
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

// RegisteredTestCase assists in asserting against the results of executing Registered.
type RegisteredTestCase struct {
	emails []string
	err error
}

// Expect asserts no error and exact return values.
func (tc *RegisteredTestCase) Expect(t *testing.T, emails []string) *RegisteredTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, emails, tc.emails)
	}
	return tc
}

// Error asserts an error.
func (tc *RegisteredTestCase) Error(t *testing.T, errContains string) *RegisteredTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *RegisteredTestCase) NoError(t *testing.T) *RegisteredTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Get returns the result of executing Registered.
func (tc *RegisteredTestCase) Get() (emails []string, err error) {
	return tc.emails, tc.err
}

// Registered executes the function and returns a corresponding test case.
func Registered(ctx context.Context) *RegisteredTestCase {
	tc := &RegisteredTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.emails, tc.err = Svc.Registered(ctx)
		return tc.err
	})
	return tc
}

// OnAllowRegisterTestCase assists in asserting against the results of executing OnAllowRegister.
type OnAllowRegisterTestCase struct {
	allow bool
	err error
}

// Expect asserts no error and exact return values.
func (tc *OnAllowRegisterTestCase) Expect(t *testing.T, allow bool) *OnAllowRegisterTestCase {
	if assert.NoError(t, tc.err) {
		assert.Equal(t, allow, tc.allow)
	}
	return tc
}

// Error asserts an error.
func (tc *OnAllowRegisterTestCase) Error(t *testing.T, errContains string) *OnAllowRegisterTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *OnAllowRegisterTestCase) NoError(t *testing.T) *OnAllowRegisterTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Get returns the result of executing OnAllowRegister.
func (tc *OnAllowRegisterTestCase) Get() (allow bool, err error) {
	return tc.allow, tc.err
}

// OnAllowRegister executes the function and returns a corresponding test case.
func OnAllowRegister(ctx context.Context, email string) *OnAllowRegisterTestCase {
	tc := &OnAllowRegisterTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.allow, tc.err = Svc.OnAllowRegister(ctx, email)
		return tc.err
	})
	return tc
}

// OnRegisteredTestCase assists in asserting against the results of executing OnRegistered.
type OnRegisteredTestCase struct {
	err error
}

// Expect asserts no error and exact return values.
func (tc *OnRegisteredTestCase) Expect(t *testing.T) *OnRegisteredTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Error asserts an error.
func (tc *OnRegisteredTestCase) Error(t *testing.T, errContains string) *OnRegisteredTestCase {
	if assert.Error(t, tc.err) {
		assert.Contains(t, tc.err.Error(), errContains)
	}
	return tc
}

// NoError asserts no error.
func (tc *OnRegisteredTestCase) NoError(t *testing.T) *OnRegisteredTestCase {
	assert.NoError(t, tc.err)
	return tc
}

// Get returns the result of executing OnRegistered.
func (tc *OnRegisteredTestCase) Get() (err error) {
	return tc.err
}

// OnRegistered executes the function and returns a corresponding test case.
func OnRegistered(ctx context.Context, email string) *OnRegisteredTestCase {
	tc := &OnRegisteredTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.err = Svc.OnRegistered(ctx, email)
		return tc.err
	})
	return tc
}
