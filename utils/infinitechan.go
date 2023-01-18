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

package utils

import (
	"sync"
	"time"
)

// InfiniteChan is backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are delivered to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
type InfiniteChan[T any] struct {
	ch        chan T
	queue     []T
	lock      sync.Mutex
	closed    bool
	count     int
	delivered int
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
	oc.lock.Lock()
	oc.tryDeliver()
	oc.lock.Unlock()
	return oc.ch
}

// Push pushes an element to the channel, if it has spare capacity.
// If not, the element if queued for delivery to the channel at a later time.
// Push therefore never blocks. It will panic if the channel is closed.
func (oc *InfiniteChan[T]) Push(elem T) {
	oc.lock.Lock()
	if oc.closed {
		oc.lock.Unlock()
		panic("push on closed channel")
	}
	oc.queue = append(oc.queue, elem)
	oc.count++
	oc.tryDeliver()
	oc.lock.Unlock()
}

// tryDeliver delivers to the channel as many elements as its spare capacity.
// Must be called under lock!
func (oc *InfiniteChan[T]) tryDeliver() (delivered int) {
	for {
		if len(oc.queue) == 0 || oc.closed {
			return delivered
		}
		select {
		case oc.ch <- oc.queue[0]:
			oc.queue = oc.queue[1:]
			delivered++
			oc.delivered++
		default:
			return delivered
		}
	}
}

// Close closes the channel after trying to deliver any queued items to the channel.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout.
// Close will spin-block until reading from the channel is finished or until the channel is
// abandoned and left unread for the idle timeout.
func (oc *InfiniteChan[T]) Close(idleTimeout time.Duration) (fullyDelivered bool) {
	lastDelivery := time.Now()
	for {
		oc.lock.Lock()
		n := oc.tryDeliver()
		if len(oc.queue) == 0 {
			oc.closed = true
			close(oc.ch)
			oc.lock.Unlock()
			return true // Fully delivered
		}
		oc.lock.Unlock()
		if n > 0 {
			lastDelivery = time.Now()
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if time.Since(lastDelivery) >= idleTimeout {
			// Nothing was read from the channel in more than the idle timeout
			oc.lock.Lock()
			oc.closed = true
			close(oc.ch)
			oc.lock.Unlock()
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}
