/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
	"go.opentelemetry.io/otel/trace"
)

// SetOnStartup adds a function to be called during the starting up of the microservice.
// Startup callbacks are called in the order they were added.
func (c *Connector) SetOnStartup(handler service.StartupHandler) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onStartup = append(c.onStartup, handler)
	return nil
}

// SetOnShutdown adds a function to be called during the shutting down of the microservice.
// Shutdown callbacks are called in the reverse order they were added,
// whether of the status of a corresponding startup callback.
func (c *Connector) SetOnShutdown(handler service.ShutdownHandler) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onShutdown = append(c.onShutdown, handler)
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
	defaultPlane := false
	if c.plane == "" {
		if plane := env.Get("MICROBUS_PLANE"); plane != "" {
			err := c.SetPlane(plane)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.plane == "" {
			c.plane = "microbus"
			defaultPlane = true
		}
	}

	// Identify the environment deployment
	if c.deployment == "" {
		if deployment := env.Get("MICROBUS_DEPLOYMENT"); deployment != "" {
			err := c.SetDeployment(deployment)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.deployment == "" {
			testNameHex := ""
			for lvl := 0; true; lvl++ {
				pc, _, _, ok := runtime.Caller(lvl)
				if !ok {
					break
				}
				runtimeFunc := runtime.FuncForPC(pc)
				funcName := runtimeFunc.Name()
				// testing.tRunner is the test runner
				// testing.(*B).runN is the benchmark runner
				if strings.HasPrefix(funcName, "testing.") {
					c.deployment = TESTING
					if defaultPlane && testNameHex != "" {
						c.plane = testNameHex
					}
					break
				} else if strings.Contains(funcName, ".Test") || strings.Contains(funcName, ".Benchmark") {
					// Generate a unique name for the test to be used as plane if none is explicitly specified
					testNameHex = hex.EncodeToString([]byte(funcName))
				}
			}
		}
		if c.deployment == "" {
			c.deployment = LOCAL
			if nats := env.Get("MICROBUS_NATS"); nats != "" {
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
	span := trc.NewSpan(nil)
	ctx := context.Background()
	defer func() {
		if err != nil {
			// OpenTelemetry: record the error
			span.SetError(err)
			c.ForceTrace(ctx)
			c.LogError(ctx, "Starting up", log.Error(err))
			span.End()
			c.Shutdown()
		} else {
			span.End()
		}
	}()
	c.onStartupCalled = false

	// OpenTelemetry: init
	err = c.initTracer(ctx)
	if err != nil {
		err = errors.Trace(err)
		return err
	}

	// OpenTelemetry: create a span for the callback
	ctx, span = c.StartSpan(ctx, "startup")

	// Initialize logger
	err = c.initLogger()
	if err != nil {
		err = errors.Trace(err)
		return err
	}
	c.LogInfo(ctx, "Startup")

	// Connect to NATS
	err = c.connectToNATS(ctx)
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
	err = c.refreshConfig(ctx, false)
	if err != nil {
		err = errors.Trace(err)
		return err
	}
	c.logConfigs(ctx)

	// Start the distributed cache
	c.distribCache, err = dlru.NewCache(ctx, c, ":888/dcache")
	if err != nil {
		err = errors.Trace(err)
		return err
	}

	// Call the callback functions in order
	c.onStartupCalled = true
	for i := 0; i < len(c.onStartup); i++ {
		err = c.doCallback(
			ctx,
			"onstartup",
			func(ctx context.Context) error {
				return c.onStartup[i](ctx)
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

	// OpenTelemetry: create a span for the callback
	ctx, span := c.StartSpan(context.Background(), "shutdown")

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
		c.LogInfo(ctx, "Stubborn pending operations", log.Int("ops", int(undrained)))
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
		c.LogWarn(ctx, "Unable to drain pending operations", log.Int("ops", int(undrained)))
	}

	// Call the callback functions in reverse order
	if c.onStartupCalled {
		for i := len(c.onShutdown) - 1; i >= 0; i-- {
			err = c.doCallback(
				ctx,
				"onshutdown",
				func(ctx context.Context) error {
					return c.onShutdown[i](ctx)
				},
			)
			if err != nil {
				lastErr = errors.Trace(err)
			}
		}
	}

	// Close the distributed cache
	if c.distribCache != nil {
		err = c.distribCache.Close(ctx)
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
		// OpenTelemetry: record the error
		span.SetError(lastErr)
		c.ForceTrace(ctx)
		c.LogError(ctx, "Shutting down", log.Error(lastErr))
	}

	// Terminate logger
	c.LogInfo(ctx, "Shutdown")
	_ = c.terminateLogger()
	// No point trying to log the error at this point

	// OpenTelemetry: terminate
	span.End()
	_ = c.termTracer(ctx)
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

// Go launches a goroutine in the lifetime context of the microservice.
// Errors and panics are automatically captured and logged.
// On shutdown, the microservice will attempt to gracefully end a pending goroutine before termination.
func (c *Connector) Go(ctx context.Context, f func(ctx context.Context) (err error)) error {
	if !c.started {
		return errors.New("not started")
	}
	atomic.AddInt32(&c.pendingOps, 1)
	subCtx := frame.ContextWithFrameOf(c.lifetimeCtx, ctx)             // Copy the frame headers
	subCtx = trace.ContextWithSpan(subCtx, trace.SpanFromContext(ctx)) // Copy the tracing context
	go func() {
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

// Parallel executes multiple jobs in parallel and returns the first error it encounters.
// It is a convenient pattern for calling multiple other microservices and thus amortize the network latency.
// There is no mechanism to identify the failed jobs so this pattern isn't suited for jobs that
// update data and require to be rolled back on failure.
func (c *Connector) Parallel(jobs ...func() (err error)) error {
	n := len(jobs)
	errChan := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	atomic.AddInt32(&c.pendingOps, int32(n))
	for _, j := range jobs {
		j := j
		go func() {
			defer atomic.AddInt32(&c.pendingOps, -1)
			defer wg.Done()
			errChan <- utils.CatchPanic(j)
		}()
	}
	wg.Wait()
	close(errChan)
	for e := range errChan {
		if e != nil {
			return e // NoTrace
		}
	}
	return nil
}
