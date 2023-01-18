/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
