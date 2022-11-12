package application

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/services/configurator/configuratorapi"
)

// Application is a collection of microservices that run in a single process and share the same lifecycle.
type Application struct {
	services   []Service
	sig        chan os.Signal
	plane      string
	deployment string
}

// New creates a new app with a collection of microservices.
// Included microservices must be explicitly started up.
func New(services ...Service) *Application {
	return &Application{
		services: services,
		sig:      make(chan os.Signal, 1),
	}
}

// NewTesting creates a new app for running in a unit test environment.
// A random plane of communication is used to isolate the app from other apps.
// Included microservices must be explicitly started up.
func NewTesting(services ...Service) *Application {
	return &Application{
		services:   services,
		sig:        make(chan os.Signal, 1),
		plane:      rand.AlphaNum64(8),
		deployment: "LOCAL",
	}
}

// Include adds a collection of microservices to the app.
// The microservices must be explicitly started up even if other microservices added
// earlier had been started already.
func (a *Application) Include(services ...Service) {
	a.services = append(a.services, services...)
}

// RemoveByHost shuts down and removes all microservices with the indicated host name.
func (a *Application) RemoveByHost(host string) error {
	var lastErr error
	for j := len(a.services) - 1; j >= 0; j-- {
		if a.services[j].HostName() != host {
			continue
		}
		if a.services[j].IsStarted() {
			err := a.services[j].Shutdown()
			if err != nil {
				lastErr = errors.Trace(err)
			}
		}
		if !a.services[j].IsStarted() {
			copy(a.services[j:], a.services[j+1:])
		}
	}
	return lastErr
}

// Startup all unstarted microservices included in this app.
// If included, the configurator is started first, then other microservices in order of inclusion.
func (a *Application) Startup() error {
	// Move configurator to the front of the list
	for i, s := range a.services {
		if s.HostName() != configuratorapi.HostName || i == 0 {
			continue
		}
		copy(a.services[i+1:], a.services[i:])
		a.services[i] = s
	}
	// Start other services in order
	for _, s := range a.services {
		if s.IsStarted() {
			continue
		}
		s.SetPlane(a.plane)
		s.SetDeployment(a.deployment)
		err := s.Startup()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Shutdown all started microservices included in this app.
// If included, the configurator is shut down last, after other microservices in reverse order of inclusion.
func (a *Application) Shutdown() error {
	var lastErr error
	for j := len(a.services) - 1; j >= 0; j-- {
		if !a.services[j].IsStarted() {
			continue
		}
		err := a.services[j].Shutdown()
		if err != nil {
			lastErr = errors.Trace(err)
		}
	}
	return lastErr
}

// WaitForInterrupt blocks until an interrupt is received through
// a SIGTERM, SIGINT or a call to interrupt.
func (a *Application) WaitForInterrupt() {
	signal.Notify(a.sig, syscall.SIGINT, syscall.SIGTERM)
	<-a.sig
}

// Interrupt the app.
func (a *Application) Interrupt() {
	a.sig <- syscall.SIGINT
}

// Run all microservices included in this app until an interrupt is received.
func (a *Application) Run() error {
	err := a.Startup()
	if err != nil {
		return errors.Trace(err)
	}
	a.WaitForInterrupt()
	err = a.Shutdown()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
