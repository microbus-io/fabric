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

	"github.com/microbus-io/fabric/examples/messaging/messagingapi"
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
	_ messagingapi.Client
)

// Mock is a mockable version of the messaging.example microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockHome func(w http.ResponseWriter, r *http.Request) (err error)
	mockNoQueue func(w http.ResponseWriter, r *http.Request) (err error)
	mockDefaultQueue func(w http.ResponseWriter, r *http.Request) (err error)
	mockCacheLoad func(w http.ResponseWriter, r *http.Request) (err error)
	mockCacheStore func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("messaging.example"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The Messaging microservice demonstrates service-to-service communication patterns.`)
	svc.SetOnStartup(svc.doOnStartup)

	// Webs
	svc.Subscribe(`ANY`, `:443/home`, svc.doHome)
	svc.Subscribe(`ANY`, `:443/no-queue`, svc.doNoQueue, sub.NoQueue())
	svc.Subscribe(`ANY`, `:443/default-queue`, svc.doDefaultQueue)
	svc.Subscribe(`ANY`, `:443/cache-load`, svc.doCacheLoad)
	svc.Subscribe(`ANY`, `:443/cache-store`, svc.doCacheStore)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doHome handles the Home web handler.
func (svc *Mock) doHome(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockHome == nil {
		return errors.New("mocked endpoint 'Home' not implemented")
	}
	err = svc.mockHome(w, r)
	return errors.Trace(err)
}

// MockHome sets up a mock handler for the Home web handler.
func (svc *Mock) MockHome(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockHome = handler
	return svc
}

// doNoQueue handles the NoQueue web handler.
func (svc *Mock) doNoQueue(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockNoQueue == nil {
		return errors.New("mocked endpoint 'NoQueue' not implemented")
	}
	err = svc.mockNoQueue(w, r)
	return errors.Trace(err)
}

// MockNoQueue sets up a mock handler for the NoQueue web handler.
func (svc *Mock) MockNoQueue(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockNoQueue = handler
	return svc
}

// doDefaultQueue handles the DefaultQueue web handler.
func (svc *Mock) doDefaultQueue(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockDefaultQueue == nil {
		return errors.New("mocked endpoint 'DefaultQueue' not implemented")
	}
	err = svc.mockDefaultQueue(w, r)
	return errors.Trace(err)
}

// MockDefaultQueue sets up a mock handler for the DefaultQueue web handler.
func (svc *Mock) MockDefaultQueue(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockDefaultQueue = handler
	return svc
}

// doCacheLoad handles the CacheLoad web handler.
func (svc *Mock) doCacheLoad(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockCacheLoad == nil {
		return errors.New("mocked endpoint 'CacheLoad' not implemented")
	}
	err = svc.mockCacheLoad(w, r)
	return errors.Trace(err)
}

// MockCacheLoad sets up a mock handler for the CacheLoad web handler.
func (svc *Mock) MockCacheLoad(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockCacheLoad = handler
	return svc
}

// doCacheStore handles the CacheStore web handler.
func (svc *Mock) doCacheStore(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockCacheStore == nil {
		return errors.New("mocked endpoint 'CacheStore' not implemented")
	}
	err = svc.mockCacheStore(w, r)
	return errors.Trace(err)
}

// MockCacheStore sets up a mock handler for the CacheStore web handler.
func (svc *Mock) MockCacheStore(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockCacheStore = handler
	return svc
}
