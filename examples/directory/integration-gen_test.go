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

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/utils"

	"github.com/stretchr/testify/assert"

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
	_ *errors.TracedError
	_ *httpx.BodyReader
	_ pub.Option
	_ *shardedsql.DB
	_ utils.InfiniteChan[int]
	_ assert.TestingT
	_ *directoryapi.Client
)

var (
	// App manages the lifecycle of the microservices used in the test
	App *application.Application
	// Svc is the directory.example microservice being tested
	Svc *Service
	// DBMySQL is a temporary sharded MySQL database
	DBMySQL shardedsql.TestingDB
)

func TestMain(m *testing.M) {
	var code int

	// Initialize the application
	err := func() error {
		var err error
		App = application.NewTesting()
		Svc = NewService().(*Service)
		err = DBMySQL.Open("mysql")
		if err != nil {
			return err
		}
		App.With(MySQL(DBMySQL.DataSource()))
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
		err = DBMySQL.Close()
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

// CreateTestCase assists in asserting against the results of executing Create.
type CreateTestCase struct {
	_testName string
	created *directoryapi.Person
	err error
}

// Name sets a name to the test case.
func (tc *CreateTestCase) Name(testName string) *CreateTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *CreateTestCase) Expect(t *testing.T, created *directoryapi.Person) *CreateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, created, tc.created)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *CreateTestCase) Error(t *testing.T, errContains string) *CreateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *CreateTestCase) ErrorCode(t *testing.T, statusCode int) *CreateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *CreateTestCase) NoError(t *testing.T) *CreateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *CreateTestCase) Assert(t *testing.T, asserter func(t *testing.T, created *directoryapi.Person, err error)) *CreateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.created, tc.err)
	})
	return tc
}

// Get returns the result of executing Create.
func (tc *CreateTestCase) Get() (created *directoryapi.Person, err error) {
	return tc.created, tc.err
}

// Create executes the function and returns a corresponding test case.
func Create(ctx context.Context, person *directoryapi.Person) *CreateTestCase {
	tc := &CreateTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.created, tc.err = Svc.Create(ctx, person)
		return tc.err
	})
	return tc
}

// LoadTestCase assists in asserting against the results of executing Load.
type LoadTestCase struct {
	_testName string
	person *directoryapi.Person
	ok bool
	err error
}

// Name sets a name to the test case.
func (tc *LoadTestCase) Name(testName string) *LoadTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *LoadTestCase) Expect(t *testing.T, person *directoryapi.Person, ok bool) *LoadTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, person, tc.person)
			assert.Equal(t, ok, tc.ok)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *LoadTestCase) Error(t *testing.T, errContains string) *LoadTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *LoadTestCase) ErrorCode(t *testing.T, statusCode int) *LoadTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *LoadTestCase) NoError(t *testing.T) *LoadTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *LoadTestCase) Assert(t *testing.T, asserter func(t *testing.T, person *directoryapi.Person, ok bool, err error)) *LoadTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.person, tc.ok, tc.err)
	})
	return tc
}

// Get returns the result of executing Load.
func (tc *LoadTestCase) Get() (person *directoryapi.Person, ok bool, err error) {
	return tc.person, tc.ok, tc.err
}

// Load executes the function and returns a corresponding test case.
func Load(ctx context.Context, key directoryapi.PersonKey) *LoadTestCase {
	tc := &LoadTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.person, tc.ok, tc.err = Svc.Load(ctx, key)
		return tc.err
	})
	return tc
}

// DeleteTestCase assists in asserting against the results of executing Delete.
type DeleteTestCase struct {
	_testName string
	ok bool
	err error
}

// Name sets a name to the test case.
func (tc *DeleteTestCase) Name(testName string) *DeleteTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *DeleteTestCase) Expect(t *testing.T, ok bool) *DeleteTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, ok, tc.ok)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *DeleteTestCase) Error(t *testing.T, errContains string) *DeleteTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *DeleteTestCase) ErrorCode(t *testing.T, statusCode int) *DeleteTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *DeleteTestCase) NoError(t *testing.T) *DeleteTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *DeleteTestCase) Assert(t *testing.T, asserter func(t *testing.T, ok bool, err error)) *DeleteTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.ok, tc.err)
	})
	return tc
}

// Get returns the result of executing Delete.
func (tc *DeleteTestCase) Get() (ok bool, err error) {
	return tc.ok, tc.err
}

// Delete executes the function and returns a corresponding test case.
func Delete(ctx context.Context, key directoryapi.PersonKey) *DeleteTestCase {
	tc := &DeleteTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.ok, tc.err = Svc.Delete(ctx, key)
		return tc.err
	})
	return tc
}

// UpdateTestCase assists in asserting against the results of executing Update.
type UpdateTestCase struct {
	_testName string
	updated *directoryapi.Person
	ok bool
	err error
}

// Name sets a name to the test case.
func (tc *UpdateTestCase) Name(testName string) *UpdateTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *UpdateTestCase) Expect(t *testing.T, updated *directoryapi.Person, ok bool) *UpdateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, updated, tc.updated)
			assert.Equal(t, ok, tc.ok)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *UpdateTestCase) Error(t *testing.T, errContains string) *UpdateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *UpdateTestCase) ErrorCode(t *testing.T, statusCode int) *UpdateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *UpdateTestCase) NoError(t *testing.T) *UpdateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *UpdateTestCase) Assert(t *testing.T, asserter func(t *testing.T, updated *directoryapi.Person, ok bool, err error)) *UpdateTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.updated, tc.ok, tc.err)
	})
	return tc
}

// Get returns the result of executing Update.
func (tc *UpdateTestCase) Get() (updated *directoryapi.Person, ok bool, err error) {
	return tc.updated, tc.ok, tc.err
}

// Update executes the function and returns a corresponding test case.
func Update(ctx context.Context, person *directoryapi.Person) *UpdateTestCase {
	tc := &UpdateTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.updated, tc.ok, tc.err = Svc.Update(ctx, person)
		return tc.err
	})
	return tc
}

// LoadByEmailTestCase assists in asserting against the results of executing LoadByEmail.
type LoadByEmailTestCase struct {
	_testName string
	person *directoryapi.Person
	ok bool
	err error
}

// Name sets a name to the test case.
func (tc *LoadByEmailTestCase) Name(testName string) *LoadByEmailTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *LoadByEmailTestCase) Expect(t *testing.T, person *directoryapi.Person, ok bool) *LoadByEmailTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, person, tc.person)
			assert.Equal(t, ok, tc.ok)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *LoadByEmailTestCase) Error(t *testing.T, errContains string) *LoadByEmailTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *LoadByEmailTestCase) ErrorCode(t *testing.T, statusCode int) *LoadByEmailTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *LoadByEmailTestCase) NoError(t *testing.T) *LoadByEmailTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *LoadByEmailTestCase) Assert(t *testing.T, asserter func(t *testing.T, person *directoryapi.Person, ok bool, err error)) *LoadByEmailTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.person, tc.ok, tc.err)
	})
	return tc
}

// Get returns the result of executing LoadByEmail.
func (tc *LoadByEmailTestCase) Get() (person *directoryapi.Person, ok bool, err error) {
	return tc.person, tc.ok, tc.err
}

// LoadByEmail executes the function and returns a corresponding test case.
func LoadByEmail(ctx context.Context, email string) *LoadByEmailTestCase {
	tc := &LoadByEmailTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.person, tc.ok, tc.err = Svc.LoadByEmail(ctx, email)
		return tc.err
	})
	return tc
}

// ListTestCase assists in asserting against the results of executing List.
type ListTestCase struct {
	_testName string
	keys []directoryapi.PersonKey
	err error
}

// Name sets a name to the test case.
func (tc *ListTestCase) Name(testName string) *ListTestCase {
	tc._testName = testName
	return tc
}

// Expect asserts no error and exact return values.
func (tc *ListTestCase) Expect(t *testing.T, keys []directoryapi.PersonKey) *ListTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.NoError(t, tc.err) {
			assert.Equal(t, keys, tc.keys)
		}
	})
	return tc
}

// Error asserts an error.
func (tc *ListTestCase) Error(t *testing.T, errContains string) *ListTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Contains(t, tc.err.Error(), errContains)
		}
	})
	return tc
}

// ErrorCode asserts an error by its status code.
func (tc *ListTestCase) ErrorCode(t *testing.T, statusCode int) *ListTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		if assert.Error(t, tc.err) {
			assert.Equal(t, statusCode, errors.Convert(tc.err).StatusCode)
		}
	})
	return tc
}

// NoError asserts no error.
func (tc *ListTestCase) NoError(t *testing.T) *ListTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		assert.NoError(t, tc.err)
	})
	return tc
}

// Assert asserts using a provided function.
func (tc *ListTestCase) Assert(t *testing.T, asserter func(t *testing.T, keys []directoryapi.PersonKey, err error)) *ListTestCase {
	t.Run(tc._testName, func(t *testing.T) {
		asserter(t, tc.keys, tc.err)
	})
	return tc
}

// Get returns the result of executing List.
func (tc *ListTestCase) Get() (keys []directoryapi.PersonKey, err error) {
	return tc.keys, tc.err
}

// List executes the function and returns a corresponding test case.
func List(ctx context.Context) *ListTestCase {
	tc := &ListTestCase{}
	tc.err = utils.CatchPanic(func() error {
		tc.keys, tc.err = Svc.List(ctx)
		return tc.err
	})
	return tc
}
