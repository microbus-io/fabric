// Code generated by Microbus. DO NOT EDIT.

package intermediate

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/examples/hello/helloapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ helloapi.Client
)

// Mock is a mockable version of the hello.example microservice,
// allowing functions, sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	MockHello func(w http.ResponseWriter, r *http.Request) (err error)
	MockEcho func(w http.ResponseWriter, r *http.Request) (err error)
	MockPing func(w http.ResponseWriter, r *http.Request) (err error)
	MockCalculator func(w http.ResponseWriter, r *http.Request) (err error)
	MockBusJPEG func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock(version int) *Mock {
	svc := &Mock{
		Connector: connector.New("hello.example"),
	}
	svc.SetVersion(version)
	svc.SetDescription(`The Hello microservice demonstrates the various capabilities of a microservice.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Webs
	svc.Subscribe(`:443/hello`, svc.doHello)
	svc.Subscribe(`:443/echo`, svc.doEcho)
	svc.Subscribe(`:443/ping`, svc.doPing)
	svc.Subscribe(`:443/calculator`, svc.doCalculator)
	svc.Subscribe(`:443/bus.jpeg`, svc.doBusJPEG)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTINGAPP {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doHello handles the Hello web handler.
func (svc *Mock) doHello(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockHello == nil {
		return errors.New("mocked endpoint 'Hello' not implemented")
	}
	err = svc.MockHello(w, r)
	return errors.Trace(err)
}

// doEcho handles the Echo web handler.
func (svc *Mock) doEcho(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockEcho == nil {
		return errors.New("mocked endpoint 'Echo' not implemented")
	}
	err = svc.MockEcho(w, r)
	return errors.Trace(err)
}

// doPing handles the Ping web handler.
func (svc *Mock) doPing(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockPing == nil {
		return errors.New("mocked endpoint 'Ping' not implemented")
	}
	err = svc.MockPing(w, r)
	return errors.Trace(err)
}

// doCalculator handles the Calculator web handler.
func (svc *Mock) doCalculator(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockCalculator == nil {
		return errors.New("mocked endpoint 'Calculator' not implemented")
	}
	err = svc.MockCalculator(w, r)
	return errors.Trace(err)
}

// doBusJPEG handles the BusJPEG web handler.
func (svc *Mock) doBusJPEG(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockBusJPEG == nil {
		return errors.New("mocked endpoint 'BusJPEG' not implemented")
	}
	err = svc.MockBusJPEG(w, r)
	return errors.Trace(err)
}
