/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package eventsink

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/eventsink/eventsinkapi"
	"github.com/microbus-io/fabric/examples/eventsink/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request

	_ eventsinkapi.Client
)

/*
Service implements the eventsink.example microservice.

The event sink microservice handles events that are fired by the event source microservice.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	mux           sync.Mutex
	registrations map[string]bool
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.registrations = map[string]bool{}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
OnAllowRegister blocks registrations from certain email providers
as well as duplicate registrations.
*/
func (svc *Service) OnAllowRegister(ctx context.Context, email string) (allow bool, err error) {
	email = strings.ToLower(email)
	if strings.HasSuffix(email, "@gmail.com") || strings.HasSuffix(email, "@hotmail.com") {
		return false, nil
	}
	svc.mux.Lock()
	defer svc.mux.Unlock()
	return !svc.registrations[email], nil
}

/*
OnRegistered keeps track of registrations.
*/
func (svc *Service) OnRegistered(ctx context.Context, email string) (err error) {
	email = strings.ToLower(email)
	svc.mux.Lock()
	svc.registrations[email] = true
	svc.mux.Unlock()
	return nil
}

/*
Registered returns the list of registered users.
*/
func (svc *Service) Registered(ctx context.Context) (emails []string, err error) {
	emails = []string{}
	svc.mux.Lock()
	for k := range svc.registrations {
		emails = append(emails, k)
	}
	svc.mux.Unlock()
	return emails, nil
}
