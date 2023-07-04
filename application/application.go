/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package application

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
)

// Application is a collection of microservices that run in a single process and share the same lifecycle.
type Application struct {
	services        []connector.Service
	sig             chan os.Signal
	plane           string
	deployment      string
	mux             sync.Mutex
	startupTimeout  time.Duration
	shutdownTimeout time.Duration
	withInits       []func(connector.Service) error
}

// New creates a new app with a collection of microservices.
// Microservices are included in the app's lifecycle management and will be
// started up or shut down with the app.
// Inclusion by itself does not startup or shutdown the microservices.
// Explicit action is required.
func New(services ...connector.Service) *Application {
	app := &Application{
		sig:             make(chan os.Signal, 1),
		startupTimeout:  time.Second * 20,
		shutdownTimeout: time.Second * 20,
	}
	app.Include(services...)
	return app
}

// NewTesting creates a new app for running in a unit test environment.
// Microservices are included in the app's lifecycle management and will be
// started up or shut down with the app.
// Inclusion by itself does not startup or shutdown the microservices.
// Explicit action is required.
// A random plane of communication is used to isolate the app from other apps.
// Tickers of microservices do not run in the TESTINGAPP deployment environment.
func NewTesting(services ...connector.Service) *Application {
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
// Added microservices are included in the app's lifecycle management and will be
// started up or shut down with the app.
// Inclusion by itself does not startup or shutdown the microservices.
// Explicit action is required.
func (app *Application) Include(services ...connector.Service) {
	app.mux.Lock()
	for _, s := range services {
		s.SetPlane(app.plane)
		s.SetDeployment(app.deployment)
	}
	app.services = append(app.services, services...)
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
	res := make([]connector.Service, len(app.services))
	copy(res, app.services)
	app.mux.Unlock()
	return res
}

// ServicesByHost returns the microservices included in this app that match the host name.
// If no microservices match the host name, an empty array is returned.
func (app *Application) ServicesByHost(host string) []connector.Service {
	app.mux.Lock()
	res := []connector.Service{}
	for _, s := range app.services {
		if s.HostName() == host {
			res = append(res, s)
		}
	}
	app.mux.Unlock()
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
// Configurators are started first, followed by other microservices in parallel.
// If an error is returned, there is no guarantee as to the state of the microservices:
// some microservices may have been started while others not.
func (app *Application) Startup() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	// Init the services
	for _, s := range app.services {
		s.With(app.withInits...)
	}

	// Sort the microservices in their startup sequence
	services := make([]connector.Service, len(app.services))
	copy(services, app.services)
	sort.Slice(services, func(i, j int) bool {
		return services[i].StartupSequence() < services[j].StartupSequence()
	})

	// Start the microservices in each sequence in parallel with an overall timeout of 20 seconds
	ctx, cancel := context.WithTimeout(context.Background(), app.startupTimeout)
	defer cancel()
	at := 0
	for i := at; i < len(services); i++ {
		if services[i].StartupSequence() != services[at].StartupSequence() {
			err := app.startupBatch(ctx, services[at:i])
			if err != nil {
				return err
			}
			at = i
		}
		if i == len(services)-1 {
			err := app.startupBatch(ctx, services[at:])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// startupBatch starts up a group of microservices in parallel.
// A context deadline is used to limit the time allotted to the operation.
func (app *Application) startupBatch(ctx context.Context, services []connector.Service) error {
	// Start the microservices in parallel
	startErrs := make(chan error, len(services))
	var wg sync.WaitGroup
	var offsettingDelay time.Duration
	for _, s := range services {
		if s.IsStarted() {
			continue
		}
		s := s
		wg.Add(1)
		offsettingDelay += 2 * time.Millisecond
		go func() {
			time.Sleep(offsettingDelay)
			defer wg.Done()
			err := errors.Newf("'%s' failed to start", s.HostName())
			delay := time.Millisecond
			for {
				select {
				case <-ctx.Done():
					// Failed to start in allotted time, return the last error
					startErrs <- err
					return
				case <-time.After(delay):
					err = s.Startup()
					if err == nil {
						return
					}
					delay = time.Second // Try again a second later
				}
			}
		}()
	}
	wg.Wait()
	close(startErrs)
	var lastErr error
	for e := range startErrs {
		if e != nil {
			lastErr = e
		}
	}
	return lastErr
}

// Shutdown shuts down all started microservices included in this app.
// The configurator is shut down last, after other microservices in parallel.
// If an error is returned, there is no guarantee as to the state of the microservices:
// some microservices may have been shut down while others not.
func (app *Application) Shutdown() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	// Sort the microservices in their reverse startup sequence
	services := make([]connector.Service, len(app.services))
	copy(services, app.services)
	sort.Slice(services, func(j, i int) bool {
		return services[i].StartupSequence() < services[j].StartupSequence()
	})

	// Shutdown the microservices in each sequence in parallel with an overall timeout of 20 seconds
	ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
	defer cancel()
	var lastErr error
	at := 0
	for i := at; i < len(services); i++ {
		if services[i].StartupSequence() != services[at].StartupSequence() {
			err := app.shutdownBatch(ctx, services[at:i])
			if err != nil {
				lastErr = err
			}
			at = i
		}
		if i == len(services)-1 {
			err := app.shutdownBatch(ctx, services[at:])
			if err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// shutdownBatch shuts down a group of microservices in parallel.
// A context deadline is used to limit the time allotted to the operation.
func (app *Application) shutdownBatch(ctx context.Context, services []connector.Service) error {
	// Shutdown the microservices in parallel
	shutdownErrs := make(chan error, len(services))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range services {
		if !s.IsStarted() {
			continue
		}
		s := s
		wg.Add(1)
		delay += 2 * time.Millisecond
		go func() {
			time.Sleep(delay)
			shutdownErrs <- s.Shutdown()
			wg.Done()
		}()
	}
	wg.Wait()
	close(shutdownErrs)
	var lastErr error
	for e := range shutdownErrs {
		if e != nil {
			lastErr = e
		}
	}
	return lastErr
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
