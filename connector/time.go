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
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/utils"
)

// StartTicker initiates a recurring job at a set interval.
func (c *Connector) StartTicker(name string, interval time.Duration, handler service.TickerHandler, options ...cb.Option) error {
	if err := utils.ValidateTickerName(name); err != nil {
		return errors.Trace(err)
	}

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	if _, ok := c.tickers[strings.ToLower(name)]; ok {
		return errors.Newf("ticker '%s' is already started", name)
	}

	cb, err := cb.NewCallback(name, handler, options...)
	if err != nil {
		return errors.Trace(err)
	}
	cb.Interval = interval
	c.tickers[strings.ToLower(name)] = cb
	if c.started {
		c.runTicker(cb)
	}

	return nil
}

// StopTicker terminates a recurring job.
func (c *Connector) StopTicker(name string) error {
	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	if job, ok := c.tickers[strings.ToLower(name)]; ok {
		if job.Ticker != nil {
			job.Ticker.Stop()
		}
		delete(c.tickers, strings.ToLower(name))
	}
	return nil
}

// StopAllTickers terminates all recurring jobs.
func (c *Connector) StopAllTickers() error {
	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	for _, job := range c.tickers {
		if job.Ticker != nil {
			job.Ticker.Stop()
		}
	}
	c.tickers = map[string]*cb.Callback{}
	return nil
}

// runAllTickers starts goroutines to run all tickers.
func (c *Connector) runAllTickers() {
	for _, job := range c.tickers {
		c.runTicker(job)
	}
}

// runTicker starts a goroutine to run the ticker.
func (c *Connector) runTicker(job *cb.Callback) {
	if job.Ticker == nil {
		job.Ticker = time.NewTicker(job.Interval)
	} else {
		// Already running
		return
	}
	go func() {
		c.LogDebug(c.Lifetime(), "Ticker started", log.String("name", job.Name))
		for range job.Ticker.C {
			if !c.started {
				continue
			}

			// Call the callback
			atomic.AddInt32(&c.pendingOps, 1)
			started := c.Now()
			callbackCtx := c.lifetimeCtx
			cancel := func() {}
			if job.TimeBudget > 0 {
				callbackCtx, cancel = context.WithTimeout(c.lifetimeCtx, job.TimeBudget)
			}
			err := utils.CatchPanic(func() error {
				return job.Handler.(service.TickerHandler)(callbackCtx)
			})
			cancel()
			if err != nil {
				err = errors.Trace(err, c.hostName, job.Name)
				c.LogError(c.Lifetime(), "Ticker callback", log.Error(err), log.String("ticker", job.Name))
			}
			dur := time.Since(started)
			atomic.AddInt32(&c.pendingOps, -1)

			// Drain ticker, in case of a long-running job that spans multiple intervals
			skipped := 0
			done := false
			for !done {
				select {
				case <-job.Ticker.C:
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
	if c.Deployment() == PROD {
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
