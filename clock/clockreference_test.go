package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type internalClock struct {
	calls map[string]bool
}

func (c *internalClock) After(d time.Duration) <-chan time.Time {
	c.calls["After"] = true
	return time.After(d)
}

func (c *internalClock) AfterFunc(d time.Duration, f func()) *Timer {
	c.calls["AfterFunc"] = true
	return &Timer{timer: time.AfterFunc(d, f)}
}

func (c *internalClock) Now() time.Time {
	c.calls["Now"] = true
	return time.Now()
}

func (c *internalClock) Since(t time.Time) time.Duration {
	c.calls["Since"] = true
	return time.Since(t)
}

func (c *internalClock) Until(t time.Time) time.Duration {
	c.calls["Until"] = true
	return time.Until(t)
}

func (c *internalClock) Sleep(d time.Duration) {
	c.calls["Sleep"] = true
	time.Sleep(d)
}

func (c *internalClock) Tick(d time.Duration) <-chan time.Time {
	c.calls["Tick"] = true
	return time.Tick(d)
}

func (c *internalClock) Ticker(d time.Duration) *Ticker {
	c.calls["Ticker"] = true
	t := time.NewTicker(d)
	return &Ticker{C: t.C, ticker: t}
}

func (c *internalClock) Timer(d time.Duration) *Timer {
	c.calls["Timer"] = true
	t := time.NewTimer(d)
	return &Timer{C: t.C, timer: t}
}

func (c *internalClock) WithDeadline(parent context.Context, d time.Time) (context.Context, context.CancelFunc) {
	c.calls["WithDeadline"] = true
	return context.WithDeadline(parent, d)
}

func (c *internalClock) WithTimeout(parent context.Context, t time.Duration) (context.Context, context.CancelFunc) {
	c.calls["WithTimeout"] = true
	return context.WithTimeout(parent, t)
}

func TestClock_Reference(t *testing.T) {
	t.Parallel()

	iClock := &internalClock{
		calls: map[string]bool{},
	}
	refClock := NewClockReference(iClock)
	assert.Same(t, iClock, refClock.Get())
	refClock.Set(refClock)
	assert.Same(t, iClock, refClock.Get())
	refClock.Set(iClock)
	assert.Same(t, iClock, refClock.Get())

	refClock.After(time.Millisecond)
	timer := refClock.AfterFunc(time.Millisecond, func() {})
	timer.Stop()
	now := refClock.Now()
	refClock.Since(now)
	refClock.Sleep(time.Millisecond)
	refClock.Tick(time.Millisecond)
	ticker := refClock.Ticker(time.Millisecond)
	ticker.Stop()
	timer = refClock.Timer(time.Millisecond)
	timer.Stop()
	refClock.Until(now)
	_, cancel := refClock.WithDeadline(context.Background(), now)
	cancel()
	_, cancel = refClock.WithTimeout(context.Background(), time.Millisecond)
	cancel()

	for _, f := range []string{"After", "AfterFunc", "Now", "Since", "Sleep", "Tick", "Ticker", "Timer", "Until", "WithDeadline", "WithTimeout"} {
		assert.True(t, iClock.calls[f])
	}
}
