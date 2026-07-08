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
	"net/http"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
)

// tickerCallback holds settings for a user tickerCallback handler, such as the OnStartup and OnShutdown callbacks.
type tickerCallback struct {
	Name     string
	Handler  service.TickerHandler
	Interval time.Duration
	Ticker   *time.Ticker
	Done     chan bool
}

// StartTicker initiates a recurring job at a set interval.
// Tickers do not run when the connector is running in the TESTING deployment environment.
// Ticker names are case sensitive.
func (c *Connector) StartTicker(name string, interval time.Duration, handler service.TickerHandler) error {
	if name == "" {
		return c.captureInitErr(errors.New("ticker name is required"))
	}
	if handler == nil {
		return nil
	}
	if interval <= 0 {
		return c.captureInitErr(errors.New("non-positive interval '%v'", interval))
	}

	c.tickersLock.Lock()
	_, ok := c.tickers[name]
	if ok {
		c.tickersLock.Unlock()
		return c.captureInitErr(errors.New("ticker '%s' is already started", name))
	}
	c.tickers[name] = &tickerCallback{
		Name:     name,
		Handler:  handler,
		Interval: interval,
	}
	if c.isPhase(startedUp) {
		c.runTicker(c.tickers[name])
	}
	c.tickersLock.Unlock()

	return nil
}

// StopTicker stops a running ticker.
// Ticker names are case sensitive.
func (c *Connector) StopTicker(name string) error {
	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()
	job, ok := c.tickers[name]
	if !ok {
		err := errors.New("unknown ticker '%s'", name)
		return c.captureInitErr(err)
	}
	if job.Ticker != nil {
		job.Ticker.Stop()
		job.Ticker = nil
		close(job.Done)
		job.Done = nil
	}
	delete(c.tickers, name)
	return nil
}

// stopTickers terminates all recurring jobs.
func (c *Connector) stopTickers() error {
	c.tickersLock.Lock()
	for _, job := range c.tickers {
		if job.Ticker != nil {
			job.Ticker.Stop()
			job.Ticker = nil
			close(job.Done)
			job.Done = nil
		}
	}
	c.tickersLock.Unlock()
	return nil
}

// runTickers starts goroutines to run all tickers.
func (c *Connector) runTickers() {
	c.tickersLock.Lock()
	for _, job := range c.tickers {
		c.runTicker(job)
	}
	c.tickersLock.Unlock()
}

// runTicker starts a goroutine to run the ticker.
func (c *Connector) runTicker(job *tickerCallback) {
	if c.deployment == TESTING {
		c.LogDebug(c.Lifetime(), "Ticker disabled while testing",
			"name", job.Name,
		)
		return
	}
	if job.Handler == nil {
		return
	}
	if job.Ticker != nil {
		return // Already running
	}
	job.Ticker = time.NewTicker(job.Interval)
	job.Done = make(chan bool)
	ticker := job.Ticker
	done := job.Done
	lifetime := c.Lifetime()
	c.pendingOps.Add(1)
	go func() {
		defer c.pendingOps.Add(-1)
		c.LogDebug(lifetime, "Ticker started",
			"name", job.Name,
		)
		defer c.LogDebug(lifetime, "Ticker stopped",
			"name", job.Name,
		)
		for {
			select {
			case <-ticker.C:
			case <-done:
				return
			case <-lifetime.Done():
				return
			}
			if !c.isPhase(startedUp) {
				continue
			}

			// OpenTelemetry: create a span for the callback
			handlerName := utils.ToKebabCase(job.Name)
			ctx, span := c.StartSpan(lifetime, handlerName, trc.Internal())

			c.pendingOps.Add(1)
			startTime := time.Now()
			err := errors.CatchPanic(func() error {
				return job.Handler(ctx)
			})
			if err != nil {
				c.LogError(ctx, "Running ticker",
					"error", err,
					"name", job.Name,
				)
				// OpenTelemetry: record the error
				span.SetError(err)
				c.ForceTrace(ctx)
			} else {
				span.SetOK(http.StatusOK)
			}
			dur := time.Since(startTime)
			c.pendingOps.Add(-1)
			_ = c.RecordHistogram(
				ctx,
				"microbus_callback_duration_seconds",
				dur.Seconds(),
				"name", job.Name,
				"type", "ticker",
				"error", func() string {
					if err != nil {
						return "ERROR"
					}
					return "OK"
				}(),
			)
			span.End()

			// Drain ticker, in case of a long-running job that spans multiple intervals
			skipped := 0
			drained := false
			for !drained {
				select {
				case <-ticker.C:
					skipped++
				default:
					drained = true
				}
			}
			if skipped > 0 {
				c.LogWarn(lifetime, "Ticker skipped",
					"name", job.Name,
					"beats", skipped,
					"runtime", dur,
				)
			}
		}
	}()
}

// Sleep pauses the current goroutine for the specified duration,
// or until the provided context or the lifetime context of the microservice is canceled or its deadline is exceeded.
// It returns nil if the full duration elapsed, or the canceling context's error
// (context.Canceled or context.DeadlineExceeded), traced, if interrupted.
func (c *Connector) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		// The duration has elapsed
		return nil
	case <-ctx.Done():
		// The provided context was canceled or its deadline was exceeded
		return errors.Trace(ctx.Err())
	case <-c.Lifetime().Done():
		// The microservice is shutting down
		return errors.Trace(c.Lifetime().Err())
	}
}
