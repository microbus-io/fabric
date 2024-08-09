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
	"strings"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/timex"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
)

// tickerCallback holds settings for a user tickerCallback handler, such as the OnStartup and OnShutdown callbacks.
type tickerCallback struct {
	Name     string
	Handler  service.TickerHandler
	Interval time.Duration
	Ticker   *time.Ticker
}

// StartTicker initiates a recurring job at a set interval.
// Tickers do not run when the connector is running in the TESTING deployment environment.
func (c *Connector) StartTicker(name string, interval time.Duration, handler service.TickerHandler) error {
	if err := utils.ValidateTickerName(name); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	if handler == nil {
		return nil
	}
	if interval <= 0 {
		return c.captureInitErr(errors.Newf("non-positive interval '%v'", interval))
	}
	name = strings.ToLower(name)

	c.tickersLock.Lock()
	_, ok := c.tickers[name]
	if ok {
		c.tickersLock.Unlock()
		return c.captureInitErr(errors.Newf("ticker '%s' is already started", name))
	}
	c.tickers[name] = &tickerCallback{
		Name:     name,
		Handler:  handler,
		Interval: interval,
	}
	if c.IsStarted() {
		c.runTicker(c.tickers[name])
	}
	c.tickersLock.Unlock()

	return nil
}

// StopTicker stops a running ticker.
func (c *Connector) StopTicker(name string) error {
	if err := utils.ValidateTickerName(name); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	if !c.IsStarted() {
		return nil
	}
	name = strings.ToLower(name)

	c.tickersLock.Lock()
	job, ok := c.tickers[name]
	if !ok {
		c.tickersLock.Unlock()
		return errors.Newf("unknown ticker '%s'", name)
	}
	if job.Ticker != nil {
		job.Ticker.Stop()
		job.Ticker = nil
	}
	delete(c.tickers, name)
	c.tickersLock.Unlock()
	return nil
}

// stopTickers terminates all recurring jobs.
func (c *Connector) stopTickers() error {
	c.tickersLock.Lock()
	for _, job := range c.tickers {
		if job.Ticker != nil {
			job.Ticker.Stop()
			job.Ticker = nil
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
	ticker := job.Ticker
	go func() {
		c.LogDebug(c.Lifetime(), "Ticker started",
			"name", job.Name,
		)
		defer c.LogDebug(c.Lifetime(), "Ticker stopped",
			"name", job.Name,
		)
		for range ticker.C {
			if !c.IsStarted() {
				continue
			}

			// OpenTelemetry: create a span for the callback
			ctx, span := c.StartSpan(c.Lifetime(), job.Name, trc.Internal())

			atomic.AddInt32(&c.pendingOps, 1)
			startTime := time.Now()
			err := utils.CatchPanic(func() error {
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
			}
			dur := time.Since(startTime)
			atomic.AddInt32(&c.pendingOps, -1)
			_ = c.ObserveMetric(
				"microbus_callback_duration_seconds",
				dur.Seconds(),
				job.Name,
				func() string {
					if err != nil {
						return "ERROR"
					}
					return "OK"
				}(),
			)
			span.End()

			// Drain ticker, in case of a long-running job that spans multiple intervals
			skipped := 0
			done := false
			for !done {
				select {
				case <-ticker.C:
					skipped++
				default:
					done = true
				}
			}
			if skipped > 0 {
				c.LogWarn(c.Lifetime(), "Ticker skipped",
					"beats", skipped,
					"runtime", dur,
				)
			}
		}
	}()
}

// Now returns the current time in the UTC timezone.
// The time may be offset if the context was a clock shift was set on the context using the frame.
func (c *Connector) Now(ctx context.Context) time.Time {
	offset := frame.Of(ctx).ClockShift()
	return time.Now().UTC().Add(offset)
}

// NowX returns the current time in the UTC timezone.
// The time may be offset if the context was a clock shift was set on the context using the frame.
func (c *Connector) NowX(ctx context.Context) timex.Timex {
	offset := frame.Of(ctx).ClockShift()
	return timex.Now().UTC().Add(offset)
}
