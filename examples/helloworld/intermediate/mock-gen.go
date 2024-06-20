// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/helloworld/helloworldapi"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ helloworldapi.Client
)

// Mock is a mockable version of the helloworld.example microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*Intermediate
	mockHelloWorld func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	m := &Mock{}
	m.Intermediate = NewService(m, 7357) // Stands for TEST
	return m
}

// OnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) OnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// OnShutdown is a no op.
func (svc *Mock) OnShutdown(ctx context.Context) (err error) {
	return nil
}

// MockHelloWorld sets up a mock handler for the HelloWorld endpoint.
func (svc *Mock) MockHelloWorld(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockHelloWorld = handler
	return svc
}

// HelloWorld runs the mock handler set by MockHelloWorld.
func (svc *Mock) HelloWorld(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockHelloWorld == nil {
		return errors.New("mocked endpoint 'HelloWorld' not implemented")
	}
	err = svc.mockHelloWorld(w, r)
	return errors.Trace(err)
}