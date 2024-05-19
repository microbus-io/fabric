/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

// InfiniteChan is backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are delivered to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
type InfiniteChan[T any] struct {
	ch     chan T
	queue  []T
	mux    sync.Mutex
	locks  int // For testing
	closed atomic.Bool
	pushed atomic.Int32
	queued atomic.Int32
}

// MakeInfiniteChan creates a new infinite channel backed by a finite buffered channel with the specified capacity.
// Overflowing elements are stored in a queue and are delivered to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout.
func MakeInfiniteChan[T any](capacity int) *InfiniteChan[T] {
	oc := &InfiniteChan[T]{
		ch: make(chan T, capacity),
	}
	return oc
}

// C is the underlying finite buffered channel.
// Reading from this channel will block if no elements are available.
func (oc *InfiniteChan[T]) C() <-chan T {
	// If less than capacity was pushed, then can return immediately
	if oc.pushed.Load() <= int32(cap(oc.ch)) {
		return oc.ch
	}

	oc.mux.Lock()
	oc.locks++
	oc.tryDeliver()
	oc.mux.Unlock()
	return oc.ch
}

// Push pushes an element to the channel, if it has spare capacity.
// If not, the element if queued for delivery to the channel at a later time.
// Push therefore never blocks. It will panic if the channel is closed.
func (oc *InfiniteChan[T]) Push(elem T) {
	if oc.closed.Load() {
		panic("push on closed channel")
	}

	// The first elements under capacity can be pushed directly to the channel with no need for locking
	c := oc.pushed.Add(1)
	if c <= int32(cap(oc.ch)) {
		oc.ch <- elem
		return
	}

	// Queue the element
	oc.mux.Lock()
	oc.locks++
	oc.queue = append(oc.queue, elem)
	oc.queued.Store(int32(len(oc.queue)))
	oc.tryDeliver()
	oc.mux.Unlock()
}

// tryDeliver delivers to the channel as many elements as its spare capacity.
// Must be called under lock!
func (oc *InfiniteChan[T]) tryDeliver() (delivered int) {
	for {
		if len(oc.queue) == 0 || oc.closed.Load() {
			oc.queued.Store(int32(len(oc.queue)))
			return delivered
		}
		select {
		case oc.ch <- oc.queue[0]:
			oc.queue = oc.queue[1:]
			delivered++
		default:
			oc.queued.Store(int32(len(oc.queue)))
			return delivered
		}
	}
}

// Close closes the channel after trying to deliver any queued items to the channel.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout.
// Close will spin-block until reading from the channel is finished or until the channel is
// abandoned and left unread for the idle timeout.
func (oc *InfiniteChan[T]) Close(idleTimeout time.Duration) (fullyDelivered bool) {
	// Check if already closed
	if oc.closed.Load() {
		return len(oc.queue) == 0
	}
	// If less than capacity was pushed, then can return immediately
	if oc.pushed.Load() <= int32(cap(oc.ch)) {
		oc.closed.Store(true)
		close(oc.ch)
		return true
	}

	lastDelivery := time.Now()
	for {
		if oc.queued.Load() == 0 {
			oc.closed.Store(true)
			close(oc.ch)
			return true
		}
		oc.mux.Lock()
		oc.locks++
		n := oc.tryDeliver()
		oc.mux.Unlock()
		if n > 0 {
			lastDelivery = time.Now()
			continue
		}
		if time.Since(lastDelivery) >= idleTimeout {
			// Nothing was read from the channel in more than the idle timeout
			oc.closed.Store(true)
			close(oc.ch)
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}
