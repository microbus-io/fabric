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

	"github.com/microbus-io/fabric/examples/messaging/messagingapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ messagingapi.Client
)

// Mock is a mockable version of the messaging.example microservice,
// allowing functions, sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	MockHome func(w http.ResponseWriter, r *http.Request) (err error)
	MockNoQueue func(w http.ResponseWriter, r *http.Request) (err error)
	MockDefaultQueue func(w http.ResponseWriter, r *http.Request) (err error)
	MockCacheLoad func(w http.ResponseWriter, r *http.Request) (err error)
	MockCacheStore func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock(version int) *Mock {
	svc := &Mock{
		Connector: connector.New("messaging.example"),
	}
	svc.SetVersion(version)
	svc.SetDescription(`The Messaging microservice demonstrates service-to-service communication patterns.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Webs
	svc.Subscribe(`:443/home`, svc.doHome)
	svc.Subscribe(`:443/no-queue`, svc.doNoQueue, sub.NoQueue())
	svc.Subscribe(`:443/default-queue`, svc.doDefaultQueue)
	svc.Subscribe(`:443/cache-load`, svc.doCacheLoad)
	svc.Subscribe(`:443/cache-store`, svc.doCacheStore)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTINGAPP {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
    return nil
}

// doHome handles the Home web handler.
func (svc *Mock) doHome(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockHome == nil {
		return errors.New("mocked endpoint 'Home' not implemented")
	}
	err = svc.MockHome(w, r)
    return errors.Trace(err)
}

// doNoQueue handles the NoQueue web handler.
func (svc *Mock) doNoQueue(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockNoQueue == nil {
		return errors.New("mocked endpoint 'NoQueue' not implemented")
	}
	err = svc.MockNoQueue(w, r)
    return errors.Trace(err)
}

// doDefaultQueue handles the DefaultQueue web handler.
func (svc *Mock) doDefaultQueue(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockDefaultQueue == nil {
		return errors.New("mocked endpoint 'DefaultQueue' not implemented")
	}
	err = svc.MockDefaultQueue(w, r)
    return errors.Trace(err)
}

// doCacheLoad handles the CacheLoad web handler.
func (svc *Mock) doCacheLoad(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockCacheLoad == nil {
		return errors.New("mocked endpoint 'CacheLoad' not implemented")
	}
	err = svc.MockCacheLoad(w, r)
    return errors.Trace(err)
}

// doCacheStore handles the CacheStore web handler.
func (svc *Mock) doCacheStore(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.MockCacheStore == nil {
		return errors.New("mocked endpoint 'CacheStore' not implemented")
	}
	err = svc.MockCacheStore(w, r)
    return errors.Trace(err)
}
