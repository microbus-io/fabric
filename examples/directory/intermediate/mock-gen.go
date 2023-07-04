/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

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

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
)

var (
	_ context.Context
	_ *json.Decoder
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ sub.Option
	_ directoryapi.Client
)

// Mock is a mockable version of the directory.example microservice,
// allowing functions, sinks and web handlers to be mocked.
type Mock struct {
	*connector.Connector
	MockCreate func(ctx context.Context, person *directoryapi.Person) (created *directoryapi.Person, err error)
	MockLoad func(ctx context.Context, key directoryapi.PersonKey) (person *directoryapi.Person, ok bool, err error)
	MockDelete func(ctx context.Context, key directoryapi.PersonKey) (ok bool, err error)
	MockUpdate func(ctx context.Context, person *directoryapi.Person) (updated *directoryapi.Person, ok bool, err error)
	MockLoadByEmail func(ctx context.Context, email string) (person *directoryapi.Person, ok bool, err error)
	MockList func(ctx context.Context) (keys []directoryapi.PersonKey, err error)
}

// NewMock creates a new mockable version of the microservice.
func NewMock(version int) *Mock {
	svc := &Mock{
		Connector: connector.New("directory.example"),
	}
	svc.SetVersion(version)
	svc.SetDescription(`The directory microservice stores personal records in a SQL database.`)
	svc.SetOnStartup(svc.doOnStartup)
	
	// Functions
	svc.Subscribe(`:443/create`, svc.doCreate)
	svc.Subscribe(`:443/load`, svc.doLoad)
	svc.Subscribe(`:443/delete`, svc.doDelete)
	svc.Subscribe(`:443/update`, svc.doUpdate)
	svc.Subscribe(`:443/load-by-email`, svc.doLoadByEmail)
	svc.Subscribe(`:443/list`, svc.doList)

	return svc
}

// doOnStartup makes sure that the mock is not executed in a non-dev environment.
func (svc *Mock) doOnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTINGAPP {
		return errors.Newf("mocking disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// doCreate handles marshaling for the Create function.
func (svc *Mock) doCreate(w http.ResponseWriter, r *http.Request) error {
	if svc.MockCreate == nil {
		return errors.New("mocked endpoint 'Create' not implemented")
	}
	var i directoryapi.CreateIn
	var o directoryapi.CreateOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Created, err = svc.MockCreate(
		r.Context(),
		i.Person,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doLoad handles marshaling for the Load function.
func (svc *Mock) doLoad(w http.ResponseWriter, r *http.Request) error {
	if svc.MockLoad == nil {
		return errors.New("mocked endpoint 'Load' not implemented")
	}
	var i directoryapi.LoadIn
	var o directoryapi.LoadOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Person, o.Ok, err = svc.MockLoad(
		r.Context(),
		i.Key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doDelete handles marshaling for the Delete function.
func (svc *Mock) doDelete(w http.ResponseWriter, r *http.Request) error {
	if svc.MockDelete == nil {
		return errors.New("mocked endpoint 'Delete' not implemented")
	}
	var i directoryapi.DeleteIn
	var o directoryapi.DeleteOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Ok, err = svc.MockDelete(
		r.Context(),
		i.Key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doUpdate handles marshaling for the Update function.
func (svc *Mock) doUpdate(w http.ResponseWriter, r *http.Request) error {
	if svc.MockUpdate == nil {
		return errors.New("mocked endpoint 'Update' not implemented")
	}
	var i directoryapi.UpdateIn
	var o directoryapi.UpdateOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Updated, o.Ok, err = svc.MockUpdate(
		r.Context(),
		i.Person,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doLoadByEmail handles marshaling for the LoadByEmail function.
func (svc *Mock) doLoadByEmail(w http.ResponseWriter, r *http.Request) error {
	if svc.MockLoadByEmail == nil {
		return errors.New("mocked endpoint 'LoadByEmail' not implemented")
	}
	var i directoryapi.LoadByEmailIn
	var o directoryapi.LoadByEmailOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Person, o.Ok, err = svc.MockLoadByEmail(
		r.Context(),
		i.Email,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doList handles marshaling for the List function.
func (svc *Mock) doList(w http.ResponseWriter, r *http.Request) error {
	if svc.MockList == nil {
		return errors.New("mocked endpoint 'List' not implemented")
	}
	var i directoryapi.ListIn
	var o directoryapi.ListOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Keys, err = svc.MockList(
		r.Context(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
