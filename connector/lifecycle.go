/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/trc"
	"go.opentelemetry.io/otel/trace"
)

const (
	shutDown = iota
	startingUp
	startedUp
	shuttingDown
)

// SetOnStartup sets the function to be called during the starting up of the microservice.
func (c *Connector) SetOnStartup(handler service.StartupHandler) error {
	if !c.isPhase(shutDown) {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onStartup = handler
	return nil
}

// SetOnShutdown sets the function to be called during the shutting down of the microservice.
func (c *Connector) SetOnShutdown(handler service.ShutdownHandler) error {
	if !c.isPhase(shutDown) {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onShutdown = handler
	return nil
}

// Startup the microservice by connecting to the transport and activating the subscriptions.
func (c *Connector) Startup(ctx context.Context) (err error) {
	if !c.phase.CompareAndSwap(shutDown, startingUp) {
		return errors.New("not shut down")
	}
	defer func() { c.phase.CompareAndSwap(startingUp, shutDown) }()
	if c.hostname == "" {
		return errors.New("hostname is not set")
	}
	defer func() { c.initErr = nil }()
	if c.initErr != nil {
		return c.initErr
	}

	// Determine the communication plane
	if c.plane == "" {
		if plane := env.Get("MICROBUS_PLANE"); plane != "" {
			err := c.SetPlane(plane)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if c.plane == "" && c.underTest {
			// Generate a unique plane from the test name
			h := sha256.New()
			h.Write([]byte(c.underTestName))
			c.plane = strings.ToLower(hex.EncodeToString(h.Sum(nil)[:8]))
		}
		if c.plane == "" {
			c.plane = "microbus"
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
		if c.deployment == "" && c.underTest {
			c.deployment = TESTING
		}
		if c.deployment == "" {
			c.deployment = LOCAL
			if natsURL := env.Get("MICROBUS_NATS"); natsURL != "" {
				if !strings.Contains(natsURL, "/127.0.0.1:") &&
					!strings.Contains(natsURL, "/0.0.0.0:") &&
					!strings.Contains(natsURL, "/localhost:") {
					c.deployment = PROD
				}
			}
		}
	}

	// Call shutdown to clean up, if there's an error.
	// All errors must be assigned to err.
	span := trc.NewSpan(nil)
	startTime := time.Now()
	defer func() {
		// err is the named return value
		if err != nil {
			c.LogError(ctx, "Starting up", "error", err)
			// OpenTelemetry: record the error
			span.SetError(err)
			c.ForceTrace(ctx)
		} else {
			span.SetOK(http.StatusOK)
		}
		span.End()
		_ = c.RecordHistogram(
			ctx,
			"microbus_callback_duration_seconds",
			time.Since(startTime).Seconds(),
			"name", "OnStartup",
			"type", "lifecycle",
			"error", func() string {
				if err != nil {
					return "ERROR"
				}
				return "OK"
			}(),
		)
		if err != nil {
			c.Shutdown(ctx)
		}
	}()
	c.onStartupCalled = false

	// OpenTelemetry: init
	err = c.initMeter(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.initTracer(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	// OpenTelemetry: create a span for the callback
	ctx, span = c.StartSpan(ctx, "startup", trc.Internal())

	// Initialize logger
	err = c.initLogger(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	c.LogInfo(ctx, "Startup")

	// Connect to the transport.
	err = c.transportConn.Open(ctx, c.hostname, c)
	if err != nil {
		return errors.Trace(err)
	}
	c.maxFragmentSize = c.transportConn.MaxPayload() - 64<<10 // Up to 64K for headers
	if c.maxFragmentSize < 64<<10 {
		return errors.New("message size limit is too restrictive")
	}
	c.networkRoundtrip = c.transportConn.Latency()
	c.ackTimeout = c.networkRoundtrip
	c.LogInfo(ctx, "Transport latency", "latency", c.networkRoundtrip)

	// Prepare the connector's lifetime context before any user-visible callback runs.
	// OnStartup and downstream code can rely on svc.Lifetime() being a real cancellable
	// context. It is cancelled in Shutdown after OnShutdown returns.
	c.lifetimeCtx, c.ctxCancel = context.WithCancel(context.Background())

	// Subscribe to the response subject
	c.responseSub, err = c.transportConn.QueueSubscribe(subjectOfResponseSub(c.plane, c.hostname, c.id), c.id, c.onResponse)
	if err != nil {
		return errors.Trace(err)
	}

	// Fetch configs
	err = c.refreshConfig(ctx, false)
	if err != nil {
		return errors.Trace(err)
	}
	c.logConfigs(ctx)

	// Set up the distributed cache (before the callbacks)
	if c.distribCache == nil {
		c.distribCache, err = dlru.NewCache(ctx, c, ":888/dcache")
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Call the callback function
	c.onStartupCalled = true
	if c.onStartup != nil {
		err = errors.CatchPanic(func() error {
			return c.onStartup(ctx)
		})
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Subscribe to :888 control messages
	err = c.subscribeControl()
	if err != nil {
		return errors.Trace(err)
	}

	// Activate subscriptions
	err = c.activateSubs()
	if err != nil {
		return errors.Trace(err)
	}

	c.startupTime = time.Now().UTC()
	c.phase.Store(startedUp)

	// Run all tickers after startup is complete
	c.runTickers()

	return nil
}

// Shutdown the microservice by deactivating subscriptions and disconnecting from the transport.
// The deadline on ctx, if any, bounds the time allotted to the operation: OnShutdown receives
// the remaining budget, then the drains and teardown share what's left.
func (c *Connector) Shutdown(ctx context.Context) (err error) {
	if !c.phase.CompareAndSwap(startedUp, shuttingDown) && !c.phase.CompareAndSwap(startingUp, shuttingDown) {
		return errors.New("not started")
	}

	// OpenTelemetry: create a span using the caller's ctx so its deadline propagates.
	ctx, span := c.StartSpan(ctx, "shutdown", trc.Internal())

	startTime := time.Now()
	var lastErr error

	// Stop all tickers
	err = c.stopTickers()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Deactivate the auto subscriptions. Manual subscriptions (the distributed cache plus
	// anything the user marked sub.Manual) stay active so OnShutdown code can still use
	// them. The connector tears down its own dlru-tagged group after OnShutdown returns;
	// user-owned manual groups (Python venv handlers, etc.) are the caller's responsibility.
	err = c.deactivateAutoSubs()
	if err != nil {
		lastErr = errors.Trace(err)
	}

	// Call OnShutdown while everything is still alive: lifetime context is valid, dlru is up,
	// transport is up, outbound calls work, svc.Go can still launch goroutines. User code that
	// needs to drain its own workers or flush state should do it here, bounded by ctx.
	if c.onStartupCalled && c.onShutdown != nil {
		err = errors.CatchPanic(func() error {
			return c.onShutdown(ctx)
		})
		if err != nil {
			lastErr = errors.Trace(err)
		}
	}

	// Two-phase drain: soft (cooperative) then hard (coerced by lifetime ctx cancel).
	// Partition the remaining ctx budget; fall back to fixed defaults when no deadline was set.
	softDrain, hardDrain := c.shutdownDrainBudgets(ctx)
	undrained := c.drainPendingOps(time.Now().Add(softDrain))
	if undrained > 0 {
		c.LogInfo(ctx, "Stubborn pending operations",
			"ops", int(undrained),
		)
	}

	// Cancel the lifetime context. This is the escalation step: goroutines still running
	// in svc.Go / svc.Parallel observe ctx.Done() and should exit promptly.
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
	}

	// Hard drain: short tail for cancellation-aware goroutines to exit after observing it.
	undrained = c.drainPendingOps(time.Now().Add(hardDrain))
	if undrained > 0 {
		c.LogWarn(ctx, "Unable to drain pending operations",
			"ops", int(undrained),
		)
	}

	// From here on, mandatory framework teardown runs on a context that ignores the caller's
	// cancellation: closing dlru, unsubscribing, disconnecting the transport, and flushing
	// OTel must not be skipped because the caller's shutdown budget happened to expire.
	// teardownBudget is enough for these to make their last attempts; OTel exporters have
	// their own per-export timeouts (see connector/CLAUDE.md OTLP exporter resilience).
	mctx, mcancel := context.WithTimeout(context.WithoutCancel(ctx), teardownBudget)
	defer mcancel()

	// Close the distributed cache; see dlru/CLAUDE.md for the offload window.
	// Reset to nil so the next Startup creates a fresh cache - otherwise the
	// "if c.distribCache == nil" guard skips dlru.NewCache and the cache's
	// manual subs stay off the bus across the restart.
	if c.distribCache != nil {
		err = c.distribCache.Close(mctx)
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.distribCache = nil
	}

	// Unsubscribe from the response subject
	if c.responseSub != nil {
		err = c.responseSub.Unsubscribe()
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.responseSub = nil
	}

	// Disconnect the transport
	c.transportConn.Close()

	// Last chance to log an error
	if lastErr != nil {
		c.LogError(mctx, "Shutting down", "error", lastErr)
		// OpenTelemetry: record the error
		span.SetError(lastErr)
		c.ForceTrace(mctx)
	}
	_ = c.RecordHistogram(
		mctx,
		"microbus_callback_duration_seconds",
		time.Since(startTime).Seconds(),
		"name", "OnShutdown",
		"type", "lifecycle",
		"error", func() string {
			if lastErr != nil {
				return "ERROR"
			}
			return "OK"
		}(),
	)

	// Final log entry, recorded while the meter and tracer are still live so its log counter and span event are
	// captured before their providers are flushed.
	c.LogInfo(mctx, "Shutdown")
	span.End()

	// OpenTelemetry: terminate in reverse order of initialization - logs, then traces, then metrics. The final log
	// above feeds all three, so they are torn down only after it is recorded.
	_ = c.termLogger(mctx)
	_ = c.termTracer(mctx)
	_ = c.termMeter(mctx)
	c.phase.Store(shutDown)

	return lastErr
}

// teardownBudget bounds the post-cancellation, mandatory cleanup steps in Shutdown
// (dlru close, OTel flush) so they don't run unbounded if a downstream is unreachable.
// The per-export OTLP timeout (see OTel docs) governs each network attempt within that.
const teardownBudget = 2 * time.Second

// drainPendingOps polls pendingOps until it reaches zero or deadline passes. Returns
// the number of operations still pending when it returns.
func (c *Connector) drainPendingOps(deadline time.Time) int32 {
	const pollInterval = 20 * time.Millisecond
	for c.pendingOps.Load() > 0 {
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(pollInterval)
	}
	return c.pendingOps.Load()
}

// shutdownDrainBudgets partitions the remaining ctx budget into soft and hard drains.
// Soft drain is cooperative (lifetime ctx still valid); hard drain runs after the cancel.
// When ctx has no deadline, fixed defaults are used. teardownBudget is reserved out of
// the deadline for mandatory cleanup steps that run on context.WithoutCancel.
func (c *Connector) shutdownDrainBudgets(ctx context.Context) (soft, hard time.Duration) {
	const (
		defaultSoftDrain = 8 * time.Second
		defaultHardDrain = 2 * time.Second
		maxHardDrain     = 2 * time.Second
		minHardDrain     = 100 * time.Millisecond
	)
	deadline, ok := ctx.Deadline()
	if !ok {
		return defaultSoftDrain, defaultHardDrain
	}
	remaining := time.Until(deadline) - teardownBudget
	if remaining <= 0 {
		return 0, minHardDrain
	}
	hard = remaining / 4
	if hard < minHardDrain {
		hard = minHardDrain
	}
	if hard > maxHardDrain {
		hard = maxHardDrain
	}
	soft = remaining - hard
	if soft < 0 {
		soft = 0
	}
	return soft, hard
}

// IsStarted indicates if the microservice has been successfully started up.
func (c *Connector) IsStarted() bool {
	return c.isPhase(startedUp)
}

// isPhase checks if the microservice is in any of the indicates lifecycle phases.
func (c *Connector) isPhase(phase ...int) bool {
	actual := int(c.phase.Load())
	for _, p := range phase {
		if p == actual {
			return true
		}
	}
	return false
}

// Lifetime returns a context that becomes valid before OnStartup runs and is cancelled
// only after OnShutdown returns and the connector's soft drain elapses. It is the
// canonical root context for long-lived goroutines launched from OnStartup (worker
// pools, refillers, background reconcilers) and for outbound calls that should outlive
// the request that initiated them. Its Done channel signals "the microservice is
// shutting down, finish up." Use it instead of context.Background for any work whose
// lifecycle should track the microservice's.
func (c *Connector) Lifetime() context.Context {
	return c.lifetimeCtx
}

// captureInitErr captures errors during the pre-start phase of the connector.
// If such an error occurs, the connector fails to start.
// This is useful since errors can be ignored during initialization.
func (c *Connector) captureInitErr(err error) error {
	if err != nil && c.initErr == nil && c.isPhase(shutDown) {
		c.initErr = err
	}
	return err
}

// Go launches a goroutine in the lifetime context of the microservice.
// Errors and panics are automatically captured and logged.
// On shutdown, the microservice will attempt to gracefully end a pending goroutine before termination.
func (c *Connector) Go(ctx context.Context, f func(ctx context.Context) (err error)) error {
	if !c.isPhase(startedUp, startingUp) {
		return errors.New("not started")
	}
	c.pendingOps.Add(1)
	subCtx := frame.ContextWithClonedFrameOf(c.Lifetime(), ctx)        // Copy the original frame headers
	subCtx = trace.ContextWithSpan(subCtx, trace.SpanFromContext(ctx)) // Copy the tracing context
	subCtx, span := c.StartSpan(subCtx, "Goroutine", trc.Consumer())

	go func() {
		err := errors.CatchPanic(func() error {
			return errors.Trace(f(subCtx))
		})
		if err != nil {
			c.LogError(subCtx, "Goroutine", "error", err)
		}
		c.pendingOps.Add(-1)
		span.End()
	}()
	return nil
}

// Parallel executes multiple jobs in parallel, waits for all of them to complete,
// and returns the first error it encounters.
// Jobs are passed a cancelable subcontext of the parent context that is canceled when any job errors,
// allowing the remaining jobs to abandon their work early.
// It is a convenient pattern for calling multiple other microservices and thus amortize the network latency.
// There is no mechanism to identify the failed jobs so this pattern isn't suited for jobs that
// update data and require to be rolled back on failure.
func (c *Connector) Parallel(ctx context.Context, jobs ...func(ctx context.Context) (err error)) error {
	n := len(jobs)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errChan := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	c.pendingOps.Add(int32(n))
	for _, j := range jobs {
		go func() {
			defer c.pendingOps.Add(-1)
			defer wg.Done()
			err := errors.CatchPanic(func() error {
				return j(subCtx)
			})
			if err != nil {
				errChan <- err
				cancel()
			}
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
		return "", errors.New("determining %s AZ", strings.ToUpper(cloudProvider))
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
		for i := range az {
			if az[i] >= '0' && az[i] <= '9' {
				az = az[:i] + "-" + az[i:] // us-east-1-a
				break
			}
		}
	}

	return az, nil
}
