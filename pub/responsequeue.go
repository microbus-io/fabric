/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package pub

import (
	"iter"
	"sync"
)

// NewSoloResponseQueue returns an iterator that yields a single response.
func NewSoloResponseQueue(r *Response) iter.Seq[*Response] {
	return func(yield func(*Response) bool) {
		yield(r)
	}
}

// ResponseQueue is a mutex-protected FIFO queue of *Response that supports
// push, close, and an iterator pattern for pulling responses.
type ResponseQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  []*Response
	cursor int
	closed bool
}

// NewResponseQueue creates a new ResponseQueue.
func NewResponseQueue(sz int) *ResponseQueue {
	q := &ResponseQueue{}
	if sz > 0 {
		q.queue = make([]*Response, 0, sz)
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Push appends a response to the end of the queue.
func (q *ResponseQueue) Push(r *Response) {
	q.mu.Lock()
	q.queue = append(q.queue, r)
	q.mu.Unlock()
	q.cond.Signal()
}

// Close signals that no more responses will be pushed.
func (q *ResponseQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Broadcast()
}

// Len returns the number of responses in the queue.
func (q *ResponseQueue) Len() int {
	q.mu.Lock()
	n := len(q.queue)
	q.mu.Unlock()
	return n
}

// PeekHead blocks until at least one response is in the queue, then returns it.
// It returns nil if the queue is closed with no responses.
func (q *ResponseQueue) PeekHead() *Response {
	q.mu.Lock()
	for len(q.queue) == 0 && !q.closed {
		q.cond.Wait()
	}
	var r *Response
	if len(q.queue) > 0 {
		r = q.queue[0]
	}
	q.mu.Unlock()
	return r
}

// Q returns an iterator that yields responses in FIFO order.
// It blocks when the queue is drained until more responses are pushed or the queue is closed.
func (q *ResponseQueue) Q() iter.Seq[*Response] {
	return func(yield func(*Response) bool) {
		for {
			q.mu.Lock()
			for q.cursor >= len(q.queue) && !q.closed {
				q.cond.Wait()
			}
			if q.cursor >= len(q.queue) {
				q.mu.Unlock()
				return
			}
			r := q.queue[q.cursor]
			q.queue[q.cursor] = nil // Release the consumed response rather than hold it for the whole iteration
			q.cursor++
			q.mu.Unlock()
			if !yield(r) {
				return
			}
		}
	}
}
