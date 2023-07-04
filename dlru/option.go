/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package dlru

type cacheOptions struct {
	Bump             bool
	ConsistencyCheck bool
	Replicate        bool
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
