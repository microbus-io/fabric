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

// TickerHandler handles the ticker callbacks.
type TickerHandler func(ctx context.Context) error

// StartTicker initiates a recurring job at a set interval.
// Tickers do not run when the connector is running in the TESTINGAPP deployment environment.
func (c *Connector) StartTicker(name string, interval time.Duration, handler TickerHandler, options ...cb.Option) error {
	if err := utils.ValidateTickerName(name); err != nil {
		return c.captureInitErr(errors.Trace(err))
	}

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	if _, ok := c.tickers[strings.ToLower(name)]; ok {
		return c.captureInitErr(errors.Newf("ticker '%s' is already started", name))
	}

	cb, err := cb.NewCallback(name, handler, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	cb.Interval = interval
	if interval <= 0 {
		return c.captureInitErr(errors.Newf("non-positive interval '%v'", interval))
	}
	c.tickers[strings.ToLower(name)] = cb
	if c.started {
		c.runTicker(cb)
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
func (c *Connector) runTicker(job *cb.Callback) {
	if c.deployment == TESTINGAPP {
		return // Tickers don't run during test suite execution
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
			started := c.Now()
			_ = c.doCallback(
				c.lifetimeCtx,
				job.TimeBudget,
				"Ticker callback",
				":0/"+utils.ToKebabCase(job.Name),
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

// Clock returns the clock of this connector.
func (c *Connector) Clock() clock.Clock {
	return c.clock
}

// Now returns the current time using the connector's clock, in the UTC timezone.
func (c *Connector) Now() time.Time {
	return c.clock.Now().UTC()
}

// SetClock sets an alternative clock for this connector,
// primarily to be used to inject a mock clock for testing.
func (c *Connector) SetClock(newClock clock.Clock) error {
	if c.Deployment() != LOCAL && c.Deployment() != TESTINGAPP {
		return errors.Newf("clock can't be changed in %s deployment", PROD)
	}

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	// All tickers must be stopped and restarted using the new clock
	for _, job := range c.tickers {
		if job.Ticker != nil {
			job.Ticker.Stop()
			job.Ticker = nil
		}
	}
	c.clock.Set(newClock)
	c.clockSet = true
	for _, job := range c.tickers {
		if c.started {
			c.runTicker(job)
		}
	}

	return nil
}
