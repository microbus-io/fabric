package utils

import (
	"sync"
	"time"
)

// InfiniteChan is backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are delivered to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
type InfiniteChan[T any] struct {
	C     <-chan T
	ch    chan T
	queue []T
	idle  time.Duration
	lock  sync.Mutex

	count     int
	delivered int
}

// NewInfiniteChan creates a new infinite channel backed by a finite channel with the specified capacity.
// Overflowing elements are stored in a queue and are delivered to the channel when it has free capacity.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
func NewInfiniteChan[T any](capacity int, idleTimeout time.Duration) *InfiniteChan[T] {
	oc := &InfiniteChan[T]{
		ch:   make(chan T, capacity),
		idle: idleTimeout,
	}
	oc.C = oc.ch
	return oc
}

// Push pushes an element to the channel, if it has spare capacity. If not, the element
// if queued for delivery to the channel at a later time.
func (oc *InfiniteChan[T]) Push(elem T) {
	// Add to queue
	oc.lock.Lock()
	oc.queue = append(oc.queue, elem)
	oc.count++
	oc.lock.Unlock()

	// Attempt to deliver to channel
	oc.tryDeliver()
}

// tryDeliver delivers to the channel as many elements as its spare capacity.
func (oc *InfiniteChan[T]) tryDeliver() (delivered int, done bool) {
	for {
		oc.lock.Lock()
		if len(oc.queue) == 0 {
			oc.lock.Unlock()
			return delivered, true
		}
		elem := oc.queue[0]
		oc.lock.Unlock()

		select {
		case oc.ch <- elem:
			oc.lock.Lock()
			oc.queue = oc.queue[1:]
			delivered++
			oc.delivered++
			oc.lock.Unlock()
		default:
			return delivered, false
		}
	}
}

// Close closes the channel after trying to deliver any queued items to the channel.
// Queued elements may be dropped if the channel is closed and left unread for over the idle timeout,
func (oc *InfiniteChan[T]) Close() (fullyDelivered bool) {
	lastDrain := time.Now()
	for {
		n, done := oc.tryDeliver()
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
