/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

package connector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
	"go.opentelemetry.io/otel/trace"
)

// SetOnStartup adds a function to be called during the starting up of the microservice.
// Startup callbacks are called in the order they were added.
func (c *Connector) SetOnStartup(handler service.StartupHandler) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onStartup = append(c.onStartup, handler)
	return nil
}

// SetOnShutdown adds a function to be called during the shutting down of the microservice.
// Shutdown callbacks are called in the reverse order they were added,
// whether of the status of a corresponding startup callback.
func (c *Connector) SetOnShutdown(handler service.ShutdownHandler) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onShutdown = append(c.onShutdown, handler)
	return nil
}

// Startup the microservice by connecting to the NATS bus and activating the subscriptions.
func (c *Connector) Startup() (err error) {
	if c.IsStarted() {
		return errors.New("already started")
	}
	if c.hostname == "" {
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

	// Determine the geographic locality
	if c.locality == "" {
		if locality := env.Get("MICROBUS_LOCALITY"); locality != "" {
			err := c.SetLocality(locality)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	c.locality, err = determineCloudLocality(c.locality)
	if err != nil {
		return errors.Trace(err)
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
			testNameHash := ""
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
					if defaultPlane && testNameHash != "" {
						c.plane = testNameHash
					}
					break
				} else if strings.Contains(funcName, ".Test") || strings.Contains(funcName, ".Benchmark") {
					// Generate a unique name for the test to be used as plane if none is explicitly specified
					h := sha256.New()
					h.Write([]byte(funcName))
					testNameHash = hex.EncodeToString(h.Sum(nil)[:8])
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
	startTime := time.Now()
	defer func() {
		if err != nil {
			c.LogError(ctx, "Starting up", "error", err)
			// OpenTelemetry: record the error
			span.SetError(err)
			c.ForceTrace(ctx)
		}
		span.End()
		_ = c.ObserveMetric(
			"microbus_callback_duration_seconds",
			time.Since(startTime).Seconds(),
			"startup",
			func() string {
				if err != nil {
					return "ERROR"
				}
				return "OK"
			}(),
		)
		if err != nil {
			c.Shutdown()
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
	ctx, span = c.StartSpan(ctx, "startup", trc.Internal())

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
	c.started.Store(true)

	c.maxFragmentSize = c.natsConn.MaxPayload() - 64*1024 // Up to 64K for headers
	if c.maxFragmentSize < 64*1024 {
		err = errors.New("message size limit is too restrictive")
		return err
	}

	// Subscribe to the response subject
	c.natsResponseSub, err = c.natsConn.QueueSubscribe(subjectOfResponses(c.plane, c.hostname, c.id), c.id, c.onResponse)
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
		err = utils.CatchPanic(func() error {
			return c.onStartup[i](ctx)
		})
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
func (c *Connector) Shutdown() (err error) {
	if !c.IsStarted() {
		return errors.New("not started")
	}
	c.started.Store(false)

	// OpenTelemetry: create a span for the callback
	ctx, span := c.StartSpan(context.Background(), "shutdown", trc.Internal())

	startTime := time.Now()
	var lastErr error

	// Stop all tickers
	err = c.stopTickers()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Unsubscribe all handlers
	err = c.deactivateSubs()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Drain pending operations (incoming requests, running tickers, goroutines)
	totalDrainTime := time.Duration(0)
	for atomic.LoadInt32(&c.pendingOps) > 0 && totalDrainTime < 8*time.Second { // 8 seconds
		time.Sleep(20 * time.Millisecond)
		totalDrainTime += 20 * time.Millisecond
	}
	undrained := atomic.LoadInt32(&c.pendingOps)
	if undrained > 0 {
		c.LogInfo(ctx, "Stubborn pending operations",
			"ops", int(undrained),
		)
	}

	// Cancel the root context
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
		c.lifetimeCtx = context.Background()
	}

	// Drain pending operations again after cancelling the context
	totalDrainTime = time.Duration(0)
	for atomic.LoadInt32(&c.pendingOps) > 0 && totalDrainTime < 4*time.Second { // 4 seconds
		time.Sleep(20 * time.Millisecond)
		totalDrainTime += 20 * time.Millisecond
	}
	undrained = atomic.LoadInt32(&c.pendingOps)
	if undrained > 0 {
		c.LogWarn(ctx, "Unable to drain pending operations",
			"ops", int(undrained),
		)
	}

	// Call the callback functions in reverse order
	if c.onStartupCalled {
		for i := len(c.onShutdown) - 1; i >= 0; i-- {
			err = utils.CatchPanic(func() error {
				return c.onShutdown[i](ctx)
			})
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
		err = c.natsResponseSub.Unsubscribe()
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
		c.LogError(ctx, "Shutting down", "error", lastErr)
		// OpenTelemetry: record the error
		span.SetError(lastErr)
		c.ForceTrace(ctx)
	}
	_ = c.ObserveMetric(
		"microbus_callback_duration_seconds",
		time.Since(startTime).Seconds(),
		"shutdown",
		func() string {
			if lastErr != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)

	// OpenTelemetry: terminate
	span.End()
	_ = c.termTracer(ctx)

	c.LogInfo(ctx, "Shutdown")

	return lastErr
}

// IsStarted indicates if the microservice has been successfully started.
func (c *Connector) IsStarted() bool {
	return c.started.Load()
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
	if err != nil && c.initErr == nil && !c.IsStarted() {
		c.initErr = err
	}
	return err
}

// Go launches a goroutine in the lifetime context of the microservice.
// Errors and panics are automatically captured and logged.
// On shutdown, the microservice will attempt to gracefully end a pending goroutine before termination.
func (c *Connector) Go(ctx context.Context, f func(ctx context.Context) (err error)) error {
	if !c.IsStarted() {
		return errors.New("not started")
	}
	atomic.AddInt32(&c.pendingOps, 1)
	subCtx := frame.ContextWithFrameOf(c.lifetimeCtx, ctx)             // Copy the frame headers
	subCtx = trace.ContextWithSpan(subCtx, trace.SpanFromContext(ctx)) // Copy the tracing context
	subCtx, span := c.StartSpan(subCtx, "Goroutine", trc.Consumer())

	go func() {
		defer span.End()
		defer atomic.AddInt32(&c.pendingOps, -1)
		err := utils.CatchPanic(func() error {
			return errors.Trace(f(subCtx))
		})
		if err != nil {
			c.LogError(subCtx, "Goroutine", "error", err)
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

// determineCloudLocality determines the locality from the instance meta-data when hosted on AWS or GCP.
func determineCloudLocality(cloudProvider string) (locality string, err error) {
	var httpReq *http.Request
	switch strings.ToUpper(cloudProvider) {
	case "AWS":
		// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
		// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-categories.html
		httpReq, _ = http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/placement/availability-zone", nil)
	case "GCP":
		// https://cloud.google.com/compute/docs/metadata/querying-metadata
		// https://cloud.google.com/compute/docs/metadata/predefined-metadata-keys
		httpReq, _ = http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
		httpReq.Header.Set("Metadata-Flavor", "Google")
	default:
		return cloudProvider, nil
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}
	res, err := client.Do(httpReq)
	if err != nil {
		return "", errors.Trace(err)
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.Newf("determining %s AZ", strings.ToUpper(cloudProvider))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Trace(err)
	}
	az := string(body)

	if cloudProvider == "AWS" {
		// az == us-east-1a
		az = az[:len(az)-1] + "-" + az[len(az)-1:] // us-east-1-a
	}

	if cloudProvider == "GCP" {
		// az == projects/415104041262/zones/us-east1-a
		_, az, _ = strings.Cut(az, "/zones/") // us-east1-a
		for i := 0; i < len(az); i++ {
			if az[i] >= '0' && az[i] <= '9' {
				az = az[:i] + "-" + az[i:] // us-east-1-a
				break
			}
		}
	}

	parts := strings.Split(az, "-") // [us, east, 1, a]
	for i := 0; i < len(parts)/2; i++ {
		parts[i], parts[len(parts)-1-i] = parts[len(parts)-1-i], parts[i]
	} // [a, 1, east, us]
	return strings.Join(parts, "."), nil // a.1.east.us
}
