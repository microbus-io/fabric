package application

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/microbus-io/fabric/clock"
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
	clock      clock.Clock
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

// SetClock sets an alternative clock for all microservices of this app,
// primarily to be used to inject a mock clock for testing.
func (a *Application) SetClock(clock clock.Clock) error {
	if a.started {
		for i := range a.services {
			err := a.services[i].SetClock(clock)
			if err != nil {
				for j := 0; j < i; j++ {
					a.services[j].SetClock(a.clock)
				}
				return errors.Trace(err)
			}
		}
	}
	a.clock = clock
	return nil
}

// Startup all microservices included in this app (in order)
func (a *Application) Startup() error {
	if a.started {
		return errors.New("already started")
	}

	for i := range a.services {
		var err error
		err = a.services[i].SetPlane(a.plane)
		if err == nil {
			err = a.services[i].SetDeployment(a.deployment)
		}
		if err == nil && a.clock != nil {
			err = a.services[i].SetClock(a.clock)
		}
		if err == nil {
			err = a.services[i].Startup()
		}
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
func (a *Application) WaitForInterrupt() error {
	if !a.started {
		return errors.New("not started")
	}
	signal.Notify(a.sig, syscall.SIGINT, syscall.SIGTERM)
	<-a.sig
	return nil
}

// Interrupt the app
func (a *Application) Interrupt() error {
	if !a.started {
		return errors.New("not started")
	}
	a.sig <- syscall.SIGINT
	return nil
}

// Run all microservices included in this app until an interrupt is received
func (a *Application) Run() error {
	err := a.Startup()
	if err != nil {
		return errors.Trace(err)
	}
	err = a.WaitForInterrupt()
	if err != nil {
		return errors.Trace(err)
	}
	err = a.Shutdown()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
