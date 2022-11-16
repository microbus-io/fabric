// Code generated by Microbus. DO NOT EDIT.

package eventsink

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/services/configurator"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
)

var (
	_ context.Context
	_ fmt.Stringer
	_ io.Reader
	_ http.Request
	_ os.File
	_ pub.Option
	_ utils.BodyReader
	_ eventsinkapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test.
	App *application.Application
	// Configurator is the configurator system microservice.
	Configurator *configurator.Service
	// Svc is the eventsink.example microservice being tested.
	Svc *Service
)

func TestMain(m *testing.M) {
	var code int

	// Initialize the application
	err := func() error {
		App = application.NewTesting()
		Configurator = configurator.NewService()
		Svc = NewService()
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

// QueryArg add the query argument to the request.
// The same argument may have multiple values.
func QueryArg(name string, value string) WebOption {
	return WebOption(pub.QueryArg(name, value))
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

type RegisteredTestCase struct {
	Expect  func(t *testing.T, emails []string)
	Error   func(t *testing.T, errContains string)
	NoError func(t *testing.T)
	Assert  func(t *testing.T, asserter func(t *testing.T, emails []string, err error))
}

func Registered(ctx context.Context) *RegisteredTestCase {
	tc := &RegisteredTestCase{
		Expect: func(t *testing.T, _emails []string) {
			var emails []string
			var err error
			err = utils.CatchPanic(func() error {
				emails, err = Svc.Registered(ctx)
				return err
			})
			if assert.NoError(t, err) {
				assert.Equal(t, _emails, emails)
			}
		},
		Error: func(t *testing.T, errContains string) {
			var err error
			err = utils.CatchPanic(func() error {
				_, err = Svc.Registered(ctx)
				return err
			})
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), errContains)
			}
		},
		NoError: func(t *testing.T) {
			var err error
			err = utils.CatchPanic(func() error {
				_, err = Svc.Registered(ctx)
				return err
			})
			assert.NoError(t, err)
		},
		Assert: func(t *testing.T, asserter func(t *testing.T, emails []string, err error)) {
			var emails []string
			var err error
			err = utils.CatchPanic(func() error {
				emails, err = Svc.Registered(ctx)
				return err
			})
			asserter(t, emails, err)
		},
	}
	return tc
}

type OnAllowRegisterTestCase struct {
	Expect  func(t *testing.T, allow bool)
	Error   func(t *testing.T, errContains string)
	NoError func(t *testing.T)
	Assert  func(t *testing.T, asserter func(t *testing.T, allow bool, err error))
}

func OnAllowRegister(ctx context.Context, email string) *OnAllowRegisterTestCase {
	tc := &OnAllowRegisterTestCase{
		Expect: func(t *testing.T, _allow bool) {
			var allow bool
			var err error
			err = utils.CatchPanic(func() error {
				allow, err = Svc.OnAllowRegister(ctx, email)
				return err
			})
			if assert.NoError(t, err) {
				assert.Equal(t, _allow, allow)
			}
		},
		Error: func(t *testing.T, errContains string) {
			var err error
			err = utils.CatchPanic(func() error {
				_, err = Svc.OnAllowRegister(ctx, email)
				return err
			})
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), errContains)
			}
		},
		NoError: func(t *testing.T) {
			var err error
			err = utils.CatchPanic(func() error {
				_, err = Svc.OnAllowRegister(ctx, email)
				return err
			})
			assert.NoError(t, err)
		},
		Assert: func(t *testing.T, asserter func(t *testing.T, allow bool, err error)) {
			var allow bool
			var err error
			err = utils.CatchPanic(func() error {
				allow, err = Svc.OnAllowRegister(ctx, email)
				return err
			})
			asserter(t, allow, err)
		},
	}
	return tc
}

type OnRegisteredTestCase struct {
	Expect  func(t *testing.T)
	Error   func(t *testing.T, errContains string)
	NoError func(t *testing.T)
	Assert  func(t *testing.T, asserter func(t *testing.T, err error))
}

func OnRegistered(ctx context.Context, email string) *OnRegisteredTestCase {
	tc := &OnRegisteredTestCase{
		Expect: func(t *testing.T) {
			var err error
			err = utils.CatchPanic(func() error {
				err = Svc.OnRegistered(ctx, email)
				return err
			})
			assert.NoError(t, err)
		},
		Error: func(t *testing.T, errContains string) {
			var err error
			err = utils.CatchPanic(func() error {
				err = Svc.OnRegistered(ctx, email)
				return err
			})
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), errContains)
			}
		},
		NoError: func(t *testing.T) {
			var err error
			err = utils.CatchPanic(func() error {
				err = Svc.OnRegistered(ctx, email)
				return err
			})
			assert.NoError(t, err)
		},
		Assert: func(t *testing.T, asserter func(t *testing.T, err error)) {
			var err error
			err = utils.CatchPanic(func() error {
				err = Svc.OnRegistered(ctx, email)
				return err
			})
			asserter(t, err)
		},
	}
	return tc
}
