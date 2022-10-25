package lru

import (
	"time"

	"github.com/microbus-io/fabric/clock"
)

// cacheOptions collects the options set to construct the cache.
type cacheOptions struct {
	maxWeight int
	maxAge    time.Duration
	clock     clock.Clock
}

// Option is used to construct an LRU cache
type Option func(cache *cacheOptions)

// MaxAge sets the maximum time that an element is stored in the cache.
// The age of an element is reset if and when it is bumped to the front of the cache.
func MaxAge(maxAge time.Duration) Option {
	return func(co *cacheOptions) {
		if maxAge > 0 {
			co.maxAge = maxAge
		}
	}
}

// MaxWeight sets the maximum weight that the cache can carry.
// Each element inserted into the cache carries a weight.
// Elements are evicted when the total weight exceeds the maximum.
func MaxWeight(maxWt int) Option {
	return func(co *cacheOptions) {
		if maxWt > 0 {
			co.maxWeight = maxWt
		}
	}
}

// mockClock sets a mock clock for testing purposes.
func mockClock(mockClock clock.Clock) Option {
	return func(co *cacheOptions) {
		co.clock = mockClock
	}
}
