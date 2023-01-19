/*
Copyright 2023 Microbus LLC and various contributors

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

package application

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/services/configurator/configuratorapi"
)

// Application is a collection of microservices that run in a single process and share the same lifecycle.
type Application struct {
	services       []connector.Service
	sig            chan os.Signal
	plane          string
	deployment     string
	mux            sync.Mutex
	startupTimeout time.Duration
	withInits      []func(connector.Service) error
}

// New creates a new app with a collection of microservices.
// Microservices are included in the app's lifecycle management and will be
// started up or shut down with the app.
// Inclusion by itself does not startup or shutdown the microservices.
// Explicit action is required.
func New(services ...connector.Service) *Application {
	app := &Application{
		sig:            make(chan os.Signal, 1),
		startupTimeout: time.Second * 20,
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

	// Start configurators first
	for _, s := range app.services {
		if s.IsStarted() {
			continue
		}
		if s.HostName() == configuratorapi.HostName {
			err := s.Startup()
			if err != nil {
				return err
			}
		}
	}

	// Give services 20 seconds to retry starting up, if needed
	exit := make(chan bool)
	timer := time.NewTimer(app.startupTimeout)
	defer timer.Stop()
	go func() {
		<-timer.C
		close(exit)
	}()

	// Start services in parallel
	startErrs := make(chan error, len(app.services))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range app.services {
		if s.IsStarted() {
			continue
		}
		s := s
		wg.Add(1)
		delay += 2 * time.Millisecond
		go func() {
			time.Sleep(delay)
			defer wg.Done()
			err := errors.Newf("'%s' failed to start", s.HostName())
			for {
				select {
				case <-exit:
					// Failed to start in 20 seconds, return the last error
					startErrs <- err
					return
				case <-time.After(delay):
					err = s.Startup()
					if err == nil {
						return
					}
					delay = app.startupTimeout / 10 // Try again up to 10 times
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

	// Shutdown services in parallel, except for configurators
	shutdownErrs := make(chan error, len(app.services))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range app.services {
		if !s.IsStarted() || s.HostName() == configuratorapi.HostName {
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

	// Shutdown configurators last
	for _, s := range app.services {
		if !s.IsStarted() {
			continue
		}
		if s.HostName() == configuratorapi.HostName {
			err := s.Shutdown()
			if err != nil {
				lastErr = err
			}
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
