package application

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

// Application is a collection of microservices that run in a single process and share the same lifecycle
type Application struct {
	services   []connector.Service
	started    bool
	sig        chan os.Signal
	plane      string
	deployment string
}

// New creates a new app with a collection of microservices
func New(services ...connector.Service) *Application {
	return &Application{
		services: services,
		sig:      make(chan os.Signal, 1),
	}
}

// NewTesting creates a new app for running in a unit test environment.
// A random plane of communication is used to isolate the app from other apps.
func NewTesting(services ...connector.Service) *Application {
	return &Application{
		services:   services,
		sig:        make(chan os.Signal, 1),
		plane:      rand.AlphaNum64(8),
		deployment: "LOCAL",
	}
}

// Startup all microservices included in this app (in order)
func (a *Application) Startup() error {
	if a.started {
		return errors.New("already started")
	}

	for i := range a.services {
		a.services[i].SetPlane(a.plane)
		a.services[i].SetDeployment(a.deployment)
		err := a.services[i].Startup()
		if err != nil {
			// Shutdown all services in reverse order
			for j := i - 1; j >= 0; j-- {
				a.services[j].Shutdown()
			}
			return errors.Trace(err)
		}
	}

	a.started = true
	return nil
}

// Shutdown all microservices included in this app (in reverse order)
func (a *Application) Shutdown() error {
	if !a.started {
		return errors.New("not started")
	}

	var returnErr error
	for j := len(a.services) - 1; j >= 0; j-- {
		err := a.services[j].Shutdown()
		if err != nil {
			returnErr = errors.Trace(err)
		}
	}

	a.started = false
	return returnErr
}

// IsStarted indicates if the app has been successfully started
func (a *Application) IsStarted() bool {
	return a.started
}

// WaitForInterrupt blocks until an interrupt is received through
// a SIGTERM, SIGINT or a call to interrupt
func (a *Application) WaitForInterrupt() {
	signal.Notify(a.sig, syscall.SIGINT, syscall.SIGTERM)
	<-a.sig
}

// Interrupt the app
func (a *Application) Interrupt() {
	a.sig <- syscall.SIGINT
}

// Run all microservices included in this app until an interrupt is received
func (a *Application) Run() error {
	err := a.Startup()
	if err != nil {
		return errors.Trace(err)
	}
	a.WaitForInterrupt()
	err = a.Shutdown()
	return errors.Trace(err)
}
