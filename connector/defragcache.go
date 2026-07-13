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

package connector

import (
	"sync"
	"time"
)

// defraggable is the shared surface of the two fragment assemblers the connector holds, so one cache and one
// sweeper serve both request and response reassembly.
type defraggable interface {
	LastActivity() time.Duration
}

// defragEntry is one in-progress transfer.
type defragEntry[T defraggable] struct {
	defragger T
	deadline  time.Time
}

// defragCache holds in-progress fragment reassemblies keyed by fromID|msgID. Entries are created only by
// fragment 1, torn down on completion, and swept on inactivity or lifetime expiry.
//
// The cache owns its sweeper goroutine. It starts lazily when the first entry is admitted and stops itself once
// the cache drains, so a connector that never reassembles a fragment runs no sweeper at all, and the goroutine
// lingers only while some transfer is still hanging (a lost fragment). The sweeper exists solely to reclaim those
// hung transfers; a transfer that completes normally is removed actively by remove.
type defragCache[T defraggable] struct {
	mu          sync.Mutex
	entries     map[string]*defragEntry[T]
	inactivity  time.Duration // gap after which a transfer is stale; doubles as the sweep interval
	stopping    bool          // set at shutdown; blocks new sweepers
	sweeperDone chan struct{} // non-nil while a sweeper runs, closed by it on exit
}

func newDefragCache[T defraggable]() *defragCache[T] {
	return &defragCache[T]{entries: map[string]*defragEntry[T]{}}
}

// configureSweeper sets the inactivity threshold (derived from the connector's measured round-trip) and clears the
// stopping flag so a restarted connector can sweep again. Called at startup, before any fragment can arrive. The
// sweeper ticks at the same interval: there is no point checking for staleness more often than a transfer can
// become stale.
func (dc *defragCache[T]) configureSweeper(inactivity time.Duration) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.inactivity = inactivity
	dc.stopping = false
}

// stop blocks new sweepers and waits for a running one to exit (on its next tick, having seen the stopping flag).
// Idempotent.
func (dc *defragCache[T]) stop() {
	dc.mu.Lock()
	dc.stopping = true
	done := dc.sweeperDone
	dc.mu.Unlock()
	if done != nil {
		<-done
	}
}

// len reports the number of entries currently held.
func (dc *defragCache[T]) len() int {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return len(dc.entries)
}

// admit registers a new transfer under key, carrying the lifetime deadline, and lazily starts the sweeper. It
// returns false if an entry already exists for the key.
func (dc *defragCache[T]) admit(key string, d T, deadline time.Time) (admitted bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if _, exists := dc.entries[key]; exists {
		return false
	}
	dc.entries[key] = &defragEntry[T]{defragger: d, deadline: deadline}
	dc.maybeStartSweeperLocked()
	return true
}

// maybeStartSweeperLocked starts the sweeper goroutine if one is not already running and the cache is neither
// shutting down nor unconfigured. Caller holds mu.
func (dc *defragCache[T]) maybeStartSweeperLocked() {
	if dc.stopping || dc.sweeperDone != nil || dc.inactivity <= 0 {
		return
	}
	done := make(chan struct{})
	dc.sweeperDone = done
	go dc.sweepLoop(done, dc.inactivity)
}

// sweepLoop sweeps on each tick and exits once the cache is empty (its normal end, after the last transfer
// completes) or the cache is stopping. It clears sweeperDone and closes done on exit so a later admit starts a
// fresh sweeper and stop can join it.
func (dc *defragCache[T]) sweepLoop(done chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		dc.mu.Lock()
		dc.sweepLocked(time.Now(), dc.inactivity)
		if len(dc.entries) == 0 || dc.stopping {
			if dc.sweeperDone == done {
				dc.sweeperDone = nil
			}
			dc.mu.Unlock()
			close(done)
			return
		}
		dc.mu.Unlock()
	}
}

// defragger returns the assembler for a transfer, or ok=false if the key is absent.
func (dc *defragCache[T]) defragger(key string) (d T, ok bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	entry, exists := dc.entries[key]
	if !exists {
		var zero T
		return zero, false
	}
	return entry.defragger, true
}

// remove drops a transfer, whether it completed or hit a framing violation. A fragment arriving afterwards finds
// no entry and is rejected.
func (dc *defragCache[T]) remove(key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.entries, key)
}

// sweep drops transfers that have stalled (no fragment within inactivity) or outlived their deadline. Exposed for
// tests to drive expiry deterministically; the sweeper goroutine uses sweepLocked with the configured timing.
func (dc *defragCache[T]) sweep(now time.Time, inactivity time.Duration) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.sweepLocked(now, inactivity)
}

// sweepLocked performs one sweep pass. Caller holds mu.
func (dc *defragCache[T]) sweepLocked(now time.Time, inactivity time.Duration) {
	for key, entry := range dc.entries {
		if entry.defragger.LastActivity() > inactivity || now.After(entry.deadline) {
			delete(dc.entries, key)
		}
	}
}

// clear drops every entry. Called during shutdown teardown so a restarted connector does not inherit the
// in-flight transfers of its previous run.
func (dc *defragCache[T]) clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	clear(dc.entries)
}
