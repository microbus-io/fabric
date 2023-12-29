/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/utils"
)

// StartupHandler handles the OnStartup callback.
type StartupHandler func(ctx context.Context) error

// StartupHandler handles the OnShutdown callback.
type ShutdownHandler func(ctx context.Context) error

// SetOnStartup adds a function to be called during the starting up of the microservice.
// Startup callbacks are called in the order they were added.
// The default one minute timeout can be overridden by the appropriate option.
func (c *Connector) SetOnStartup(handler StartupHandler, options ...cb.Option) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	callback, err := cb.NewCallback("onstartup", handler, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	c.onStartup = append(c.onStartup, callback)
	return nil
}

// SetOnShutdown adds a function to be called during the shutting down of the microservice.
// Shutdown callbacks are called in the reverse order they were added,
// whether of the status of a corresponding startup callback.
// The default one minute timeout can be overridden by the appropriate option.
func (c *Connector) SetOnShutdown(handler ShutdownHandler, options ...cb.Option) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	callback, err := cb.NewCallback("onshutdown", handler, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	c.onShutdown = append(c.onShutdown, callback)
	return nil
}

// Startup the microservice by connecting to the NATS bus and activating the subscriptions.
func (c *Connector) Startup() (err error) {
	if c.started {
		return errors.New("already started")
	}
	if c.hostName == "" {
		return errors.New("hostname is not set")
	}
	defer func() { c.initErr = nil }()
	if c.initErr != nil {
		return c.initErr
	}

	// Determine the communication plane
	if c.plane == "" {
		if plane := os.Getenv("MICROBUS_PLANE"); plane != "" {
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
		if deployment := os.Getenv("MICROBUS_DEPLOYMENT"); deployment != "" {
			err := c.SetDeployment(deployment)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.deployment == "" {
			c.deployment = LOCAL
			if nats := os.Getenv("MICROBUS_NATS"); nats != "" {
				if !strings.Contains(nats, "/127.0.0.1:") &&
					!strings.Contains(nats, "/0.0.0.0:") &&
					!strings.Contains(nats, "/localhost:") {
					c.deployment = PROD
				}
			}
		}
	}

	// Call shutdown to clean up, if there's an error.
	// All errors must be assigned to err.
	defer func() {
		if err != nil {
			c.LogError(c.lifetimeCtx, "Starting up", log.Error(err))
			c.Shutdown()
		}
	}()
	c.onStartupCalled = false

	// Initialize logger
	err = c.initLogger()
	if err != nil {
		err = errors.Trace(err)
		return err
	}
	c.LogInfo(c.lifetimeCtx, "Startup")

	// Validate that clock is not changed except for development purposes
	if c.Deployment() != LOCAL && c.Deployment() != TESTINGAPP && c.clockSet {
		err = errors.Newf("clock can't be changed in %s deployment", c.Deployment())
		return err
	}

	// Connect to NATS
	err = c.connectToNATS()
	if err != nil {
		err = errors.Trace(err)
		return err
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
		err = errors.Trace(err)
		return err
	}

	// Fetch configs
	err = c.refreshConfig(c.lifetimeCtx, false)
	if err != nil {
		err = errors.Trace(err)
		return err
	}
	c.logConfigs()

	// Start the distributed cache
	c.distribCache, err = dlru.NewCache(c.lifetimeCtx, c, ":888/dcache")
	if err != nil {
		err = errors.Trace(err)
		return err
	}

	// Call the callback functions in order
	c.onStartupCalled = true
	for i := 0; i < len(c.onStartup); i++ {
		err = c.doCallback(
			c.lifetimeCtx,
			c.onStartup[i].TimeBudget,
			c.onStartup[i].Name,
			func(ctx context.Context) error {
				return c.onStartup[i].Handler.(StartupHandler)(ctx)
			},
		)
		if err != nil {
			err = errors.Trace(err)
			return err
		}
	}

	// Prepare the connector's root context
	c.lifetimeCtx, c.ctxCancel = context.WithCancel(context.Background())

	// Subscribe to :888 control messages
	err = c.subscribeControl()
	if err != nil {
		err = errors.Trace(err)
		return err
	}

	// Activate subscriptions
	for _, sub := range c.subs {
		err = c.activateSub(sub)
		if err != nil {
			err = errors.Trace(err)
			return err
		}
	}
	time.Sleep(20 * time.Millisecond) // Give time for subscription activation by NATS

	// Run all tickers
	c.runTickers()

	c.startupTime = time.Now().UTC()

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
	err := c.stopTickers()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Unsubscribe all handlers
	err = c.deactivateSubs()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Drain pending operations (incoming requests and running tickers)
	totalDrainTime := time.Duration(0)
	for atomic.LoadInt32(&c.pendingOps) > 0 && totalDrainTime < 4*time.Second {
		time.Sleep(20 * time.Millisecond)
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
		time.Sleep(20 * time.Millisecond)
		totalDrainTime += 20 * time.Millisecond
	}
	undrained = atomic.LoadInt32(&c.pendingOps)
	if undrained > 0 {
		c.LogWarn(c.lifetimeCtx, "Unable to drain pending operations", log.Int32("ops", undrained))
	}

	// Call the callback functions in reverse order
	if c.onStartupCalled {
		for i := len(c.onShutdown) - 1; i >= 0; i-- {
			err = c.doCallback(
				c.lifetimeCtx,
				c.onShutdown[i].TimeBudget,
				c.onShutdown[i].Name,
				func(ctx context.Context) error {
					return c.onShutdown[i].Handler.(ShutdownHandler)(ctx)
				},
			)
			if err != nil {
				lastErr = errors.Trace(err)
			}
		}
	}

	// Close the distributed cache
	if c.distribCache != nil {
		err = c.distribCache.Close(c.lifetimeCtx)
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.distribCache = nil
	}

	// Unsubscribe from the response subject
	if c.natsResponseSub != nil {
		err := c.natsResponseSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.natsResponseSub = nil
	}

	// Disconnect from NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	// Last chance to log an error
	if lastErr != nil {
		c.LogError(c.lifetimeCtx, "Shutting down", log.Error(lastErr))
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

// captureInitErr captures errors during the pre-start phase of the connector.
// If such an error occurs, the connector fails to start.
// This is useful since errors can be ignored during initialization.
func (c *Connector) captureInitErr(err error) error {
	if err != nil && c.initErr == nil && !c.started {
		c.initErr = err
	}
	return err
}

// StartupSequence returns the order in which this microservice should be started
// when included inside an application.
// The startup sequence is only relevant when an application contains more than
// one microservice, such as during integration testing.
func (c *Connector) StartupSequence() int {
	return c.startupSequence
}

// SetStartupSequence sets the order in which this microservice should be started
// when included inside an application.
// The startup sequence is only relevant when an application contains more than
// one microservice, such as during integration testing.
func (c *Connector) SetStartupSequence(seq int) {
	c.startupSequence = seq
}

// Go launches a goroutine in the lifetime context of the microservice.
// Errors and panics are automatically captured and logged.
// On shutdown, the microservice will attempt to gracefully end a pending goroutine
// before termination.
func (c *Connector) Go(ctx context.Context, f func(ctx context.Context) (err error)) error {
	if !c.started {
		return errors.New("not started")
	}
	subCtx := frame.Copy(c.lifetimeCtx, ctx)
	go func() {
		atomic.AddInt32(&c.pendingOps, 1)
		defer atomic.AddInt32(&c.pendingOps, -1)
		err := utils.CatchPanic(func() error {
			return errors.Trace(f(subCtx))
		})
		if err != nil {
			c.LogError(subCtx, "Goroutine", log.Error(err))
		}
	}()
	return nil
}
