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
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ metricsapi.Client
)

// Mock is a mockable version of the metrics.sys microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockCollect func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("metrics.sys"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The Metrics service is a core microservice that aggregates metrics from other microservices and makes them available for collection.`)
	svc.SetOnStartup(svc.doOnStartup)

	// Webs
	svc.Subscribe(`ANY`, `:443/collect`, svc.doCollect)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doCollect handles the Collect web handler.
func (svc *Mock) doCollect(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockCollect == nil {
		return errors.New("mocked endpoint 'Collect' not implemented")
	}
	err = svc.mockCollect(w, r)
	return errors.Trace(err)
}

// MockCollect sets up a mock handler for the Collect web handler.
func (svc *Mock) MockCollect(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockCollect = handler
	return svc
}
