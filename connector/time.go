package connector

import (
	"context"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/utils"
)

type tickerCallback struct {
	cb.Callback
	Interval time.Duration
	Ticker   *clock.Ticker
}

// StartTicker initiates a recurring job at a set interval.
func (c *Connector) StartTicker(name string, interval time.Duration, handler func(context.Context) error) error {
	match, err := regexp.MatchString(`^[a-zA-Z]+[a-zA-Z0-9]*$`, name)
	if err != nil {
		return errors.Trace(err)
	}
	if !match {
		return errors.Newf("invalid ticker name: %s", name)
	}

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	if _, ok := c.tickers[strings.ToLower(name)]; ok {
		return errors.Newf("ticker name already in use: %s", name)
	}

	job := &tickerCallback{
		Callback: cb.Callback{
			Name:    name,
			Handler: handler,
		},
		Interval: interval,
	}
	c.tickers[strings.ToLower(name)] = job
	if c.started {
		c.runTicker(job)
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
	c.tickers = map[string]*tickerCallback{}
	return nil
}

// runAllTickers starts goroutines to run all tickers.
func (c *Connector) runAllTickers() {
	for _, job := range c.tickers {
		c.runTicker(job)
	}
}

// runTicker starts a goroutine to run the ticker.
func (c *Connector) runTicker(job *tickerCallback) {
	if job.Ticker == nil {
		job.Ticker = c.clock.Ticker(job.Interval)
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
			started := c.clock.Now()
			callbackCtx := c.lifetimeCtx
			cancel := func() {}
			if job.Timeout > 0 {
				callbackCtx, cancel = context.WithTimeout(c.lifetimeCtx, job.Timeout)
			}
			err := utils.CatchPanic(func() error {
				return job.Handler(callbackCtx)
			})
			cancel()
			if err != nil {
				c.LogError(c.Lifetime(), "Ticker callback", log.Error(err))
			}
			dur := c.clock.Since(started)
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

// Now returns the current time using the connector's clock.
func (c *Connector) Now() time.Time {
	return c.clock.Now()
}

// SetClock sets an alternative clock for this connector,
// primarily to be used to inject a mock clock for testing.
func (c *Connector) SetClock(newClock clock.Clock) error {
	if _, ok := newClock.(*clock.Mock); ok && c.Deployment() == PROD {
		return errors.New("mock clock not allowed in PROD deployment environment")
	}
	if !c.started {
		return nil
	}

	c.tickersLock.Lock()
	defer c.tickersLock.Unlock()

	// All tickers must be stopped and restarted using the new clock
	for _, job := range c.tickers {
		job.Ticker.Stop()
		job.Ticker = nil
	}
	c.clock.Set(newClock)
	for _, job := range c.tickers {
		c.runTicker(job)
	}

	return nil
}
