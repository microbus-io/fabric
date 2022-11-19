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
		startupTimeout: time.Second * 20,
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
func (app *Application) Join(services ...connector.Service) {
	for _, s := range services {
		s.SetPlane(app.plane)
		s.SetDeployment(app.deployment)
	}
}

// Replace first stops any microservice that match the host names of the alternatives,
// then joins the alternatives to the app and starts them up.
// Between shutting down and starting up, there is a period of time in which the host name is not served.
// An restore function shuts down the alternatives and restores the app to its previous state.
// If an error is returned during replacing or restoring, there is no guarantee as to the state of the microservices:
// some microservices may have been started while others not.
// This function is beneficial for replacing microservices with mocks during testing and should generally not
// be used in production settings.
func (app *Application) Replace(alternatives ...connector.Service) (restore func() error, err error) {
	// Identify the affected host names
	hosts := map[string]bool{}
	for _, s := range alternatives {
		hosts[s.HostName()] = true
	}

	// Shutdown the included services that match the host name
	restart := []connector.Service{}
	app.mux.Lock()
	for _, s := range app.services {
		if hosts[s.HostName()] && s.IsStarted() {
			err := s.Shutdown()
			if err != nil {
				app.mux.Unlock()
				return nil, errors.Trace(err)
			}
			restart = append(restart, s)
		}
	}
	app.mux.Unlock()

	// Join the alternatives to the app and start them up
	for _, s := range alternatives {
		s.SetPlane(app.plane)
		s.SetDeployment(app.deployment)
		if !s.IsStarted() {
			err := s.Startup()
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	restore = func() error {
		var lastErr error
		// Shutdown the alternative services
		for _, s := range alternatives {
			err := s.Shutdown()
			if err != nil {
				lastErr = errors.Trace(err)
			}
			s.SetPlane("")
			s.SetDeployment("")
		}
		// Restart the services that were shut down
		for _, s := range restart {
			err := s.Startup()
			if err != nil {
				lastErr = errors.Trace(err)
			}
		}
		return lastErr
	}

	return restore, nil
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

// Startup all unstarted microservices included in this app.
// Configurators are started first, followed by other microservices in parallel.
// If an error is returned, there is no guarantee as to the state of the microservices:
// some microservices may have been started while others not.
func (app *Application) Startup() error {
	app.mux.Lock()
	defer app.mux.Unlock()

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
					delay = app.startupTimeout / 10 // Try again every 2 seconds
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

// Shutdown all started microservices included in this app.
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
