/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package eventsink

import (
	"context"
	"strings"
	"sync"

	"github.com/microbus-io/fabric/examples/eventsink/intermediate"
)

var (
	// Emulated data store
	mux           sync.Mutex
	registrations []string
)

/*
Service implements the eventsink.example microservice.

The event sink microservice handles events that are fired by the event source microservice.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
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
	return true, nil
}

/*
OnRegistered keeps track of registrations.
*/
func (svc *Service) OnRegistered(ctx context.Context, email string) (err error) {
	mux.Lock()
	registrations = append(registrations, strings.ToLower(email))
	mux.Unlock()
	return nil
}

/*
Registered returns the list of registered users.
*/
func (svc *Service) Registered(ctx context.Context) (emails []string, err error) {
	emails = []string{}
	mux.Lock()
	emails = append(emails, registrations...)
	mux.Unlock()
	return emails, nil
}
