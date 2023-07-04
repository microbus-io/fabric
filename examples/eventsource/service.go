/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package eventsource

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
	"github.com/microbus-io/fabric/examples/eventsource/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request

	_ eventsourceapi.Client
)

/*
Service implements the eventsource.example microservice.

The event source microservice fires events that are caught by the event sink microservice.
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
Register attempts to register a new user.
*/
func (svc *Service) Register(ctx context.Context, email string) (allowed bool, err error) {
	// Trigger an event to check if any event sink blocks the registration
	for r := range eventsourceapi.NewMulticastTrigger(svc).OnAllowRegister(ctx, email) {
		allowed, err := r.Get()
		if err != nil {
			return false, errors.Trace(err)
		}
		if !allowed {
			return false, nil
		}
	}

	// OK to register the user
	// ...

	// Trigger an event to inform all event sinks of the new registration
	for r := range eventsourceapi.NewMulticastTrigger(svc).OnRegistered(ctx, email) {
		err := r.Get()
		if err != nil {
			return true, errors.Trace(err)
		}
	}

	return true, nil
}
