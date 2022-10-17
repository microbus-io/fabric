package connector

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/utils"
)

// SetOnStartup sets a function to be called during the starting up of the microservice.
// The default one minute timeout can be overridden by the appropriate option.
func (c *Connector) SetOnStartup(f cb.CallbackHandler, options ...cb.Option) error {
	callback, err := cb.NewCallback("onstartup", f, options...)
	if err != nil {
		return errors.Trace(err)
	}
	c.onStartup = callback
	return nil
}

// SetOnShutdown sets a function to be called during the shutting down of the microservice.
// The default one minute timeout can be overridden by the appropriate option.
func (c *Connector) SetOnShutdown(f cb.CallbackHandler, options ...cb.Option) error {
	callback, err := cb.NewCallback("onshutdown", f, options...)
	if err != nil {
		return errors.Trace(err)
	}
	c.onShutdown = callback
	return nil
}

// Startup the microservice by connecting to the NATS bus and activating the subscriptions.
func (c *Connector) Startup() (err error) {
	if c.started {
		return errors.New("already started")
	}
	if c.hostName == "" {
		return errors.New("no hostname")
	}

	// Look for configs in the environment or file system
	err = c.loadConfigs()
	if err != nil {
		return errors.Trace(err)
	}

	// Determine the communication plane
	if c.plane == "" {
		if plane, ok := c.Config("Plane"); ok {
			err := c.SetPlane(plane)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.plane == "" {
			c.plane = "microbus"
		}
	}

	// Identify the environment deployment
	if c.deployment == "" {
		if deployment, ok := c.Config("Deployment"); ok {
			err := c.SetDeployment(deployment)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.deployment == "" {
			c.deployment = LOCAL
			if nats, ok := c.Config("NATS"); ok {
				if !strings.Contains(nats, "/127.0.0.1:") &&
					!strings.Contains(nats, "/0.0.0.0:") &&
					!strings.Contains(nats, "/localhost:") {
					c.deployment = PROD
				}
			}
		}
	}

	// Call shutdown to clean up, if there's an error
	defer func() {
		if err != nil {
			c.Shutdown()
		}
	}()
	c.onStartupCalled = false

	// Initialize logger
	err = c.initLogger()
	if err != nil {
		return errors.Trace(err)
	}
	c.LogInfo(c.lifetimeCtx, "Startup")

	// Validate clock
	if _, ok := c.clock.Get().(*clock.Mock); ok && c.Deployment() == PROD {
		err = errors.New("mock clock not allowed in PROD deployment environment")
		return err
	}

	// Log configs
	c.logConfigs(c.lifetimeCtx)

	// Subscribe to :888 control messages
	err = c.subscribeControl()
	if err != nil {
		return errors.Trace(err)
	}

	// Connect to NATS
	err = c.connectToNATS()
	if err != nil {
		return errors.Trace(err)
	}
	c.started = true
	c.maxFragmentSize = c.natsConn.MaxPayload() - 64*1024 // Up to 64K for headers
	if c.maxFragmentSize < 64*1024 {
		err = errors.New("message size limit is too restrictive")
		return err
	}

	// Subscribe to the response subject
	c.natsResponseSub, err = c.natsConn.QueueSubscribe(subjectOfResponses(c.plane, c.hostName, c.id), c.id, c.onResponse)
	if err != nil {
		return errors.Trace(err)
	}

	// Call the callback function, if provided
	c.onStartupCalled = true
	if c.onStartup != nil {
		callbackCtx := c.lifetimeCtx
		cancel := func() {}
		if c.onStartup.TimeBudget > 0 {
			callbackCtx, cancel = c.clock.WithTimeout(c.lifetimeCtx, c.onStartup.TimeBudget)
		}
		err = utils.CatchPanic(func() error {
			return c.onStartup.Handler(callbackCtx)
		})
		cancel()
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Prepare the connector's root context
	c.lifetimeCtx, c.ctxCancel = context.WithCancel(context.Background())

	// Activate subscriptions
	for _, sub := range c.subs {
		err = c.activateSub(sub)
		if err != nil {
			return errors.Trace(err)
		}
	}
	time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS

	// Run all tickers
	c.runAllTickers()

	return nil
}

// Shutdown the microservice by deactivating subscriptions and disconnecting from the NATS bus.
func (c *Connector) Shutdown() error {
	if !c.started {
		return errors.New("not started")
	}
	c.started = false

	var lastErr error

	// Stop all tickers
	err := c.StopAllTickers()
	if err != nil {
		lastErr = errors.Trace(err)
		c.LogError(c.lifetimeCtx, "Stopping tickers", log.Error(err))
	}

	// Unsubscribe all handlers
	err = c.UnsubscribeAll()
	if err != nil {
		lastErr = errors.Trace(err)
		c.LogError(c.lifetimeCtx, "Deactivating subscriptions", log.Error(err))
	}

	// Drain pending operations (incoming requests and running tickers)
	totalDrainTime := time.Duration(0)
	for atomic.LoadInt32(&c.pendingOps) > 0 && totalDrainTime < 4*time.Second {
		time.Sleep(10 * time.Millisecond)
		totalDrainTime += 20 * time.Millisecond
	}
	undrained := atomic.LoadInt32(&c.pendingOps)
	if undrained > 0 {
		c.LogInfo(c.lifetimeCtx, "Stubborn pending operations", log.Int32("ops", undrained))
	}

	// Cancel the root context
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
		c.lifetimeCtx = context.Background()
	}

	// Drain pending operations again after cancelling the context
	totalDrainTime = time.Duration(0)
	for atomic.LoadInt32(&c.pendingOps) > 0 && totalDrainTime < 4*time.Second {
		time.Sleep(10 * time.Millisecond)
		totalDrainTime += 20 * time.Millisecond
	}
	undrained = atomic.LoadInt32(&c.pendingOps)
	if undrained > 0 {
		c.LogWarn(c.lifetimeCtx, "Unable to drain pending operations", log.Int32("ops", undrained))
	}

	// Call the callback function, if provided
	if c.onShutdown != nil && c.onStartupCalled {
		callbackCtx := c.lifetimeCtx
		cancel := func() {}
		if c.onShutdown.TimeBudget > 0 {
			callbackCtx, cancel = c.clock.WithTimeout(c.lifetimeCtx, c.onShutdown.TimeBudget)
		}
		err = utils.CatchPanic(func() error {
			return c.onShutdown.Handler(callbackCtx)
		})
		cancel()
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(c.lifetimeCtx, "Shutdown callback", log.Error(err))
		}
	}

	// Unsubscribe from the response subject
	if c.natsResponseSub != nil {
		err := c.natsResponseSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
			c.LogError(c.lifetimeCtx, "Unsubscribing response sub", log.Error(err))
		}
		c.natsResponseSub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	// Terminate logger
	c.LogInfo(c.lifetimeCtx, "Shutdown")
	_ = c.terminateLogger()
	// No point trying to log the error at this point

	return lastErr
}

// IsStarted indicates if the microservice has been successfully started.
func (c *Connector) IsStarted() bool {
	return c.started
}

// Lifetime returns a context that gets cancelled when the microservice is shutdown.
// The Done() channel can be used to detect when the microservice is shutting down.
// In most cases the lifetime context should be used instead of the background context.
func (c *Connector) Lifetime() context.Context {
	return c.lifetimeCtx
}
