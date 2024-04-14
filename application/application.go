/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package application

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

// Application is a collection of microservices that run in a single process and share the same lifecycle.
type Application struct {
	groups          []Group
	sig             chan os.Signal
	plane           string
	deployment      string
	mux             sync.Mutex
	startupTimeout  time.Duration
	shutdownTimeout time.Duration
	withInits       []func(connector.Service) error
}

/*
New creates a new app with a collection of microservices.
Microservices can be added individually or in a group.
Microservices and groups of microservices are started sequentially in order of inclusion.
Microservices included in a group are started in parallel together.

In the following example, A is started first, then B1 and B2 in parallel, and finally C1 and C2 in parallel.

	app := application.New(
		a,
		application.Group{b1, b2},
		application.Group{c1, c2},
	)
	app.Startup()
*/
func New(services ...any) *Application {
	app := &Application{
		sig:             make(chan os.Signal, 1),
		startupTimeout:  time.Second * 20,
		shutdownTimeout: time.Second * 20,
	}
	app.Include(services...)
	return app
}

// NewTesting creates a new app for running in a unit test environment.
// Microservices can be added individually or in a group.
// A random plane of communication is used to isolate the app from other apps.
// Tickers of microservices do not run in the TESTINGAPP deployment environment.
func NewTesting(services ...any) *Application {
	app := &Application{
		sig:            make(chan os.Signal, 1),
		plane:          rand.AlphaNum64(12),
		deployment:     "TESTINGAPP",
		startupTimeout: time.Second * 8,
	}
	app.Include(services...)
	return app
}

// Include adds a collection of microservices to the app.
// Microservices can be added individually or in a group.
// Microservices and groups of microservices are started sequentially in order of inclusion.
// Microservices included in a group are started in parallel together.
func (app *Application) Include(services ...any) {
	app.mux.Lock()
	for _, s := range services {
		switch v := s.(type) {
		case connector.Service:
			v.SetPlane(app.plane)
			v.SetDeployment(app.deployment)
			app.groups = append(app.groups, Group{v})
		case Group:
			for _, ss := range v {
				ss.SetPlane(app.plane)
				ss.SetDeployment(app.deployment)
			}
			app.groups = append(app.groups, v)
		case []connector.Service:
			for _, ss := range v {
				ss.SetPlane(app.plane)
				ss.SetDeployment(app.deployment)
			}
			app.groups = append(app.groups, v)
		}
	}
	app.mux.Unlock()
}

// Join sets the plane and deployment of the microservices to that of the app.
// This allows microservices to be temporarily joined to the app without being
// permanently included in its lifecycle management.
// Joined microservices are not included in the app and do not get started up
// or shut down by the app.
func (app *Application) Join(services ...connector.Service) {
	for _, s := range services {
		s.SetPlane(app.plane)
		s.SetDeployment(app.deployment)
	}
}

// Services returns the microservices included in this app.
// The result is a new array of a limited interface of the microservices
// that provides means to identify the host of the microservice and start and stop it.
// Casting is needed in order to access the full microservice functionality.
func (app *Application) Services() []connector.Service {
	app.mux.Lock()
	var res []connector.Service
	for _, g := range app.groups {
		res = append(res, g...)
	}
	app.mux.Unlock()
	return res
}

// ServicesByHost returns the microservices included in this app that match the host name.
// If no microservices match the host name, an empty array is returned.
func (app *Application) ServicesByHost(host string) []connector.Service {
	res := []connector.Service{}
	for _, s := range app.Services() {
		if s.HostName() == host {
			res = append(res, s)
		}
	}
	return res
}

// ServiceByHost returns one of the microservices included in this app that match the host name.
// If no microservices match the host name, nil is returned.
func (app *Application) ServiceByHost(host string) connector.Service {
	services := app.ServicesByHost(host)
	if len(services) > 0 {
		return services[rand.Intn(len(services))]
	}
	return nil
}

// Startup starts all unstarted microservices included in this app.
// Microservices and groups of microservices are started sequentially in order of inclusion.
// Microservices included in a group are started in parallel together.
// If an error is returned, there is no guarantee as to the state of the microservices:
// some microservices may have been started while others not.
func (app *Application) Startup() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), app.startupTimeout)
	defer cancel()

	// Start each of the groups sequentially
	for _, g := range app.groups {
		for _, s := range g {
			s.With(app.withInits...)
		}
		err := g.Startup(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// Shutdown shuts down all started microservices included in this app.
// If an error is returned, there is no guarantee as to the state of the microservices:
// some microservices may have been shut down while others not.
func (app *Application) Shutdown() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
	defer cancel()

	// Stop each of the groups sequentially in reverse order
	for i := len(app.groups) - 1; i >= 0; i-- {
		err := app.groups[i].Shutdown(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// WaitForInterrupt blocks until an interrupt is received through
// a SIGTERM, SIGINT or a call to interrupt.
func (app *Application) WaitForInterrupt() {
	signal.Notify(app.sig, syscall.SIGINT, syscall.SIGTERM)
	<-app.sig
}

// Interrupt the app.
func (app *Application) Interrupt() {
	app.sig <- syscall.SIGINT
}

// Run starts up all microservices included in this app, waits for interrupt,
// then shuts them down.
func (app *Application) Run() error {
	err := app.Startup()
	if err != nil {
		return errors.Trace(err)
	}
	app.WaitForInterrupt()
	err = app.Shutdown()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// With adds initializers that are called on each of the included services
// before they started up.
func (app *Application) With(inits ...func(connector.Service) error) {
	app.withInits = append(app.withInits, inits...)
}
