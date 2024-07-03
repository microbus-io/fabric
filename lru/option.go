/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

// MaxAge indicates to ignore elements that have not been inserted or bumped recently.
// This option is applicable to load operations.
func MaxAge(maxAge time.Duration) Option {
	return func(opts *cacheOptions) {
		opts.MaxAge = maxAge
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
