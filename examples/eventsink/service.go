package eventsink

import (
	"context"
	"net/http"
	"strings"

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

The Event Sink microservice handles an event that is fired by the Event source microservice.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

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
	if svc.registrations[email] {
		return false, nil
	}
	return true, nil
}

/*
OnRegistered keeps track of registrations.
*/
func (svc *Service) OnRegistered(ctx context.Context, email string) (err error) {
	email = strings.ToLower(email)
	svc.registrations[email] = true
	return nil
}

/*
Registered returns the list of registered users.
*/
func (svc *Service) Registered(ctx context.Context) (emails []string, err error) {
	emails = []string{}
	for k := range svc.registrations {
		emails = append(emails, k)
	}
	return emails, nil
}
