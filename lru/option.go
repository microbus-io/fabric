/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package lru

import "time"

type cacheOptions struct {
	Bump   bool
	Weight int
	MaxAge time.Duration
}

// Option is used to customize cache operations.
type Option func(opts *cacheOptions)

// Bump indicates whether or not a loaded element is bumped to the head of the cache.
// Bumping is the default behavior.
// This option is applicable to load operations.
func Bump(bump bool) Option {
	return func(opts *cacheOptions) {
		opts.Bump = bump
	}
}

// NoBump indicates not to bump a loaded element to the head of the cache.
// This option is applicable to load operations.
func NoBump() Option {
	return Bump(false)
}

// MaxAge indicates to discard elements that have not been inserted or bumped recently.
// This option is applicable to load operations.
func MaxAge(ttl time.Duration) Option {
	return func(opts *cacheOptions) {
		opts.MaxAge = ttl
	}
}

// Weight sets the weight of the element stored in the cache.
// It must be 1 or greater and cannot exceed the cache's maximum weight limit.
// The default weight is 1.
// Elements are evicted when the total weight of all elements exceeds the cache's capacity.
// This option is applicable to store operations.
func Weight(weight int) Option {
	return func(opts *cacheOptions) {
		if weight > 0 {
			opts.Weight = weight
		}
	}
}
