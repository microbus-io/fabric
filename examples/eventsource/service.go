/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
	// Trigger a synchronous event to check if any event sink blocks the registration
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

	// Trigger an asynchronous fire-and-forget event to inform all event sinks of the new registration
	svc.Go(ctx, func(ctx context.Context) (err error) {
		for range eventsourceapi.NewMulticastTrigger(svc).OnRegistered(ctx, email) {
		}
		return nil
	})

	return true, nil
}
