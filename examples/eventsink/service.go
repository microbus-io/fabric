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
