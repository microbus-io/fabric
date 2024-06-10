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

	"github.com/microbus-io/fabric/coreservices/configurator/configuratorapi"
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
	_ configuratorapi.Client
)

// Mock is a mockable version of the configurator.sys microservice, allowing functions, event sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	mockValues func(ctx context.Context, names []string) (values map[string]string, err error)
	mockRefresh func(ctx context.Context) (err error)
	mockSync func(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock() *Mock {
	svc := &Mock{
		Connector: connector.New("configurator.sys"),
	}
	svc.SetVersion(7357) // Stands for TEST
	svc.SetDescription(`The Configurator is a core microservice that centralizes the dissemination of configuration values to other microservices.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Functions
	svc.Subscribe(`ANY`, `:443/values`, svc.doValues)
	svc.Subscribe(`ANY`, `:443/refresh`, svc.doRefresh)
	svc.Subscribe(`ANY`, `:443/sync`, svc.doSync, sub.NoQueue())

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doValues handles marshaling for the Values function.
func (svc *Mock) doValues(w http.ResponseWriter, r *http.Request) error {
	if svc.mockValues == nil {
		return errors.New("mocked endpoint 'Values' not implemented")
	}
	var i configuratorapi.ValuesIn
	var o configuratorapi.ValuesOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/values`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/values`)
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
	o.Values, err = svc.mockValues(
		r.Context(),
		i.Names,
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

// MockValues sets up a mock handler for the Values function.
func (svc *Mock) MockValues(handler func(ctx context.Context, names []string) (values map[string]string, err error)) *Mock {
	svc.mockValues = handler
	return svc
}

// doRefresh handles marshaling for the Refresh function.
func (svc *Mock) doRefresh(w http.ResponseWriter, r *http.Request) error {
	if svc.mockRefresh == nil {
		return errors.New("mocked endpoint 'Refresh' not implemented")
	}
	var i configuratorapi.RefreshIn
	var o configuratorapi.RefreshOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/refresh`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/refresh`)
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
	err = svc.mockRefresh(
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

// MockRefresh sets up a mock handler for the Refresh function.
func (svc *Mock) MockRefresh(handler func(ctx context.Context) (err error)) *Mock {
	svc.mockRefresh = handler
	return svc
}

// doSync handles marshaling for the Sync function.
func (svc *Mock) doSync(w http.ResponseWriter, r *http.Request) error {
	if svc.mockSync == nil {
		return errors.New("mocked endpoint 'Sync' not implemented")
	}
	var i configuratorapi.SyncIn
	var o configuratorapi.SyncOut
	err := httpx.ParseRequestData(r, &i)
	if err != nil {
		return errors.Trace(err)
	}
	if strings.ContainsAny(`:443/sync`, "{}") {
		spec := httpx.JoinHostAndPath("host", `:443/sync`)
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
	err = svc.mockSync(
		r.Context(),
		i.Timestamp,
		i.Values,
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

// MockSync sets up a mock handler for the Sync function.
func (svc *Mock) MockSync(handler func(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error)) *Mock {
	svc.mockSync = handler
	return svc
}
