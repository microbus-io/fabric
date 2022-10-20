/*
Adapted from https://github.com/benbjohnson/clock/releases/tag/v1.3.0

The MIT License (MIT)

# Copyright (c) 2014 Ben Johnson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package clock

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Ensure that WithDeadline is cancelled when deadline exceeded.
func TestMock_WithDeadline(t *testing.T) {
	m := NewMock()
	ctx, _ := m.WithDeadline(context.Background(), m.Now().Add(time.Second))
	m.Add(time.Second)
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Error("invalid type of error returned when deadline exceeded")
		}
	default:
		t.Error("context is not cancelled when deadline exceeded")
	}
}

// Ensure that WithDeadline does nothing when the deadline is later than the current deadline.
func TestMock_WithDeadlineLaterThanCurrent(t *testing.T) {
	m := NewMock()
	ctx, _ := m.WithDeadline(context.Background(), m.Now().Add(time.Second))
	ctx, _ = m.WithDeadline(ctx, m.Now().Add(10*time.Second))
	m.Add(time.Second)
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Error("invalid type of error returned when deadline exceeded")
		}
	default:
		t.Error("context is not cancelled when deadline exceeded")
	}
}

// Ensure that WithDeadline cancel closes Done channel with context.Canceled error.
func TestMock_WithDeadlineCancel(t *testing.T) {
	m := NewMock()
	ctx, cancel := m.WithDeadline(context.Background(), m.Now().Add(time.Second))
	cancel()
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("invalid type of error returned after cancellation")
		}
	case <-time.After(time.Second):
		t.Error("context is not cancelled after cancel was called")
	}
}

// Ensure that WithDeadline closes child contexts after it was closed.
func TestMock_WithDeadlineCancelledWithParent(t *testing.T) {
	m := NewMock()
	parent, cancel := context.WithCancel(context.Background())
	ctx, _ := m.WithDeadline(parent, m.Now().Add(time.Second))
	cancel()
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("invalid type of error returned after cancellation")
		}
	case <-time.After(time.Second):
		t.Error("context is not cancelled when parent context is cancelled")
	}
}

// Ensure that WithDeadline cancelled immediately when deadline has already passed.
func TestMock_WithDeadlineImmediate(t *testing.T) {
	m := NewMock()
	ctx, _ := m.WithDeadline(context.Background(), m.Now().Add(-time.Second))
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Error("invalid type of error returned when deadline has already passed")
		}
	default:
		t.Error("context is not cancelled when deadline has already passed")
	}
}

// Ensure that WithTimeout is cancelled when deadline exceeded.
func TestMock_WithTimeout(t *testing.T) {
	m := NewMock()
	ctx, _ := m.WithTimeout(context.Background(), time.Second)
	m.Add(time.Second)
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Error("invalid type of error returned when time is over")
		}
	default:
		t.Error("context is not cancelled when time is over")
	}
}
