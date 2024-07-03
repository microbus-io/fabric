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

package dlru

import "time"

type cacheOptions struct {
	Bump             bool
	ConsistencyCheck bool
	Replicate        bool
	MaxAge           time.Duration
}

// LoadOption customizes loading from the cache.
type LoadOption func(opts *cacheOptions)

// NoBump prevents a loaded element from being bumped to the head of the cache.
func NoBump() LoadOption {
	return func(opts *cacheOptions) {
		opts.Bump = false
	}
}

// Bump controls whether a loaded element is bumped to the head of the cache.
// The default behavior is to bump.
func Bump(bump bool) LoadOption {
	return func(opts *cacheOptions) {
		opts.Bump = bump
	}
}

// ConsistencyCheck controls whether to validate that all peers have the same value.
// Skipping the consistency check improves performance significantly when the value is available locally.
// The default behavior is to perform the consistency check.
func ConsistencyCheck(check bool) LoadOption {
	return func(opts *cacheOptions) {
		opts.ConsistencyCheck = check
	}
}

// MaxAge indicates to discard elements that have not been inserted or bumped recently.
func MaxAge(ttl time.Duration) LoadOption {
	return func(opts *cacheOptions) {
		opts.MaxAge = ttl
	}
}

// StoreOption customizes storing in the cache.
type StoreOption func(opts *cacheOptions)

// Replicate controls whether to replicate the stored element to all peers.
// Replication reduces the capacity of the cache but may increase performance
// when used in conjunction with skipping the consistency check on load.
// The default behavior is not to replicate.
func Replicate(replicate bool) StoreOption {
	return func(opts *cacheOptions) {
		opts.Replicate = replicate
	}
}
