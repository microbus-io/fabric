/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package clock

import (
	"context"
	"sync"
	"time"
)

// ClockReference is a clock that internally references another clock.
type ClockReference struct {
	clock Clock
	lock  sync.Mutex
}

// NewClockReference creates a new clock that references another clock.
func NewClockReference(internal Clock) *ClockReference {
	sh := &ClockReference{}
	sh.Set(internal)
	return sh
}

// Set sets the referenced clock.
func (c *ClockReference) Set(internal Clock) {
	var nc Clock
	if ref, ok := internal.(*ClockReference); ok {
		// Avoid cyclical references
		nc = ref.Get()
	} else {
		nc = internal
	}
	c.lock.Lock()
	c.clock = nc
	c.lock.Unlock()
}

// Get returns the referenced clock.
func (c *ClockReference) Get() (internal Clock) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock
}

func (c *ClockReference) After(d time.Duration) <-chan time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.After(d)
}

func (c *ClockReference) AfterFunc(d time.Duration, f func()) *Timer {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.AfterFunc(d, f)
}

func (c *ClockReference) Now() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Now()
}

func (c *ClockReference) Since(t time.Time) time.Duration {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Since(t)
}

func (c *ClockReference) Until(t time.Time) time.Duration {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Until(t)
}

func (c *ClockReference) Sleep(d time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.clock.Sleep(d)
}

func (c *ClockReference) Tick(d time.Duration) <-chan time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Tick(d)
}

func (c *ClockReference) Ticker(d time.Duration) *Ticker {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Ticker(d)
}

func (c *ClockReference) Timer(d time.Duration) *Timer {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.Timer(d)
}

func (c *ClockReference) WithDeadline(parent context.Context, d time.Time) (context.Context, context.CancelFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.WithDeadline(parent, d)
}

func (c *ClockReference) WithTimeout(parent context.Context, t time.Duration) (context.Context, context.CancelFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.clock.WithTimeout(parent, t)
}

var (
	// Type checking
	_ Clock = &ClockReference{}
)
