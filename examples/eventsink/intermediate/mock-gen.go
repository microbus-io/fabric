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

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
	
	eventsourceapi1 "github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
	eventsourceapi2 "github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
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
	_ eventsinkapi.Client
)

// Mock is a mockable version of the eventsink.example microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockRegistered func(ctx context.Context) (emails []string, err error)
	mockOnAllowRegister func(ctx context.Context, email string) (allow bool, err error)
	mockOnRegistered func(ctx context.Context, email string) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("eventsink.example"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The event sink microservice handles events that are fired by the event source microservice.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Functions
	svc.Subscribe(`ANY`, `:443/registered`, svc.doRegistered)

	// Sinks
	eventsourceapi1.NewHook(svc).OnAllowRegister(svc.doOnAllowRegister)
	eventsourceapi2.NewHook(svc).OnRegistered(svc.doOnRegistered)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doRegistered handles marshaling for the Registered function.
func (svc *Mock) doRegistered(w http.ResponseWriter, r *http.Request) error {
	if svc.mockRegistered == nil {
		return errors.New("mocked endpoint 'Registered' not implemented")
	}
	var i eventsinkapi.RegisteredIn
	var o eventsinkapi.RegisteredOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/registered`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/registered`)
		_, spec, _ = strings.Cut(spec, "://")
		_, spec, _ = strings.Cut(spec, "/")
		spec = "/" + spec
		pathArgs, err := httpx.ExtractPathArguments(spec, r.URL.Path)
		if err != nil {
			return errors.Trace(err)
		}
		err = httpx.DecodeDeepObject(pathArgs, &i)
		if err != nil {
			return errors.Trace(err)
		}
	}
	o.Emails, err = svc.mockRegistered(
		r.Context(),
	)
	if err != nil {
		return err // No trace
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MockRegistered sets up a mock handler for the Registered function.
func (svc *Mock) MockRegistered(handler func(ctx context.Context) (emails []string, err error)) *Mock {
	svc.mockRegistered = handler
	return svc
}

// doOnAllowRegister handles marshaling for the OnAllowRegister event sink.
func (svc *Mock) doOnAllowRegister(ctx context.Context, email string) (allow bool, err error) {
	if svc.mockOnAllowRegister == nil {
		err = errors.New("mocked endpoint 'OnAllowRegister' not implemented")
		return
	}
	allow, err = svc.mockOnAllowRegister(ctx, email)
	err = errors.Trace(err)
	return
}

// MockOnAllowRegister sets up a mock handler for the OnAllowRegister event sink.
func (svc *Mock) MockOnAllowRegister(handler func(ctx context.Context, email string) (allow bool, err error)) *Mock {
	svc.mockOnAllowRegister = handler
	return svc
}

// doOnRegistered handles marshaling for the OnRegistered event sink.
func (svc *Mock) doOnRegistered(ctx context.Context, email string) (err error) {
	if svc.mockOnRegistered == nil {
		err = errors.New("mocked endpoint 'OnRegistered' not implemented")
		return
	}
	err = svc.mockOnRegistered(ctx, email)
	err = errors.Trace(err)
	return
}

// MockOnRegistered sets up a mock handler for the OnRegistered event sink.
func (svc *Mock) MockOnRegistered(handler func(ctx context.Context, email string) (err error)) *Mock {
	svc.mockOnRegistered = handler
	return svc
}
