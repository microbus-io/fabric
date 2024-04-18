/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/timex"
	"github.com/microbus-io/fabric/utils"
)

// TickerHandler handles the ticker callbacks.
type TickerHandler func(ctx context.Context) error

// StartTicker initiates a recurring job at a set interval.
// Tickers do not run when the connector is running in the TESTINGAPP deployment environment.
func (c *Connector) StartTicker(name string, interval time.Duration, handler TickerHandler) error {
	if err := utils.ValidateTickerName(name); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	name = strings.ToLower(name)

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	if _, ok := c.tickers[name]; ok {
		return c.captureInitErr(errors.Newf("ticker '%s' is already started", name))
	}
	if interval <= 0 {
		return c.captureInitErr(errors.Newf("non-positive interval '%v'", interval))
	}
	c.tickers[name] = &callback{
		Name:     name,
		Handler:  handler,
		Interval: interval,
	}
	if c.started {
		c.runTicker(c.tickers[name])
	}

	return nil
}

// stopTickers terminates all recurring jobs.
func (c *Connector) stopTickers() error {
	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	for _, job := range c.tickers {
		if job.Ticker != nil {
			job.Ticker.Stop()
			job.Ticker = nil
		}
	}
	return nil
}

// runTickers starts goroutines to run all tickers.
func (c *Connector) runTickers() {
	for _, job := range c.tickers {
		c.runTicker(job)
	}
}

// runTicker starts a goroutine to run the ticker.
func (c *Connector) runTicker(job *callback) {
	if c.deployment == TESTINGAPP {
		c.LogDebug(c.Lifetime(), "Ticker disabled while testing", log.String("name", job.Name))
		return
	}
	c.tickersLock.Lock()
	if job.Ticker == nil {
		job.Ticker = time.NewTicker(job.Interval)
	} else {
		c.tickersLock.Unlock()
		return // Already running
	}
	ticker := job.Ticker
	c.tickersLock.Unlock()
	go func() {
		c.LogDebug(c.Lifetime(), "Ticker started", log.String("name", job.Name))
		for range ticker.C {
			if !c.started {
				continue
			}

			// Call the callback
			atomic.AddInt32(&c.pendingOps, 1)
			started := time.Now()
			_ = c.doCallback(
				c.lifetimeCtx,
				job.Name,
				func(ctx context.Context) error {
					return job.Handler.(TickerHandler)(ctx)
				},
			)
			dur := time.Since(started)
			atomic.AddInt32(&c.pendingOps, -1)

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
				c.LogWarn(c.Lifetime(), "Ticker skipped", log.Int("beats", skipped), log.Duration("runtime", dur))
			}
		}
		c.LogDebug(c.Lifetime(), "Ticker stopped", log.String("name", job.Name))
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
