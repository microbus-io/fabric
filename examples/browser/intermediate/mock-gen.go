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

	"github.com/microbus-io/fabric/examples/browser/browserapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ browserapi.Client
)

// Mock is a mockable version of the browser.example microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockBrowse func(w http.ResponseWriter, r *http.Request) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("browser.example"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The browser microservice implements a simple web browser that utilizes the egress proxy.`)
	svc.SetOnStartup(svc.doOnStartup)

	// Webs
	svc.Subscribe(`*`, `:443/browse`, svc.doBrowse)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doBrowse handles the Browse web handler.
func (svc *Mock) doBrowse(w http.ResponseWriter, r *http.Request) (err error) {
	if svc.mockBrowse == nil {
		return errors.New("mocked endpoint 'Browse' not implemented")
	}
	err = svc.mockBrowse(w, r)
	return errors.Trace(err)
}

// MockBrowse sets up a mock handler for the Browse web handler.
func (svc *Mock) MockBrowse(handler func(w http.ResponseWriter, r *http.Request) (err error)) *Mock {
	svc.mockBrowse = handler
	return svc
}
