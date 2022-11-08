package utils

import (
	"sync"
	"time"
)

// InfiniteChan is backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are pushed to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
type InfiniteChan[T any] struct {
	C     <-chan T
	ch    chan T
	queue []T
	idle  time.Duration
	lock  sync.Mutex

	count   int
	drained int
}

// NewInfiniteChan creates a new infinite channel backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are pushed to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
func NewInfiniteChan[T any](capacity int, idleTimeout time.Duration) *InfiniteChan[T] {
	oc := &InfiniteChan[T]{
		ch:   make(chan T, capacity),
		idle: idleTimeout,
	}
	oc.C = oc.ch
	return oc
}

// Push pushes an element to the channel.
func (oc *InfiniteChan[T]) Push(elem T) {
	// Add to queue
	oc.lock.Lock()
	oc.queue = append(oc.queue, elem)
	oc.count++
	oc.lock.Unlock()

	// Attempt to push to channel
	oc.tryDrainQueue()
}

// tryDrainQueue drains as many elements from the queue that fit in the channel.
func (oc *InfiniteChan[T]) tryDrainQueue() (drained int, done bool) {
	for {
		oc.lock.Lock()
		if len(oc.queue) == 0 {
			oc.lock.Unlock()
			return drained, true
		}
		elem := oc.queue[0]
		oc.lock.Unlock()

		select {
		case oc.ch <- elem:
			oc.lock.Lock()
			oc.queue = oc.queue[1:]
			drained++
			oc.drained++
			oc.lock.Unlock()
		default:
			return drained, false
		}
	}
}

// Close closes the channel after trying to drain any queued items.
// Queued elements may be dropped when the context is canceled, or if the channel is left unread
// for over the idle timeout,
func (oc *InfiniteChan[T]) Close() (fullyDrained bool) {
	lastDrain := time.Now()
	for {
		n, done := oc.tryDrainQueue()
		if done {
			// Fully drained
			close(oc.ch)
			return true
		}
		if n > 0 {
			lastDrain = time.Now()
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if time.Since(lastDrain) >= oc.idle {
			// Nothing was read from the channel in more than the idle timeout
			close(oc.ch)
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}
