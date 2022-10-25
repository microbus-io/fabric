package dlru

import (
	"time"

	"github.com/microbus-io/fabric/lru"
)

// Option is used to construct a new distributed cache.
type Option func(*Cache)

// MaxAge sets the maximum time that an element is stored in the cache.
// The age of an element is reset if and when it is bumped to the front of the cache.
func MaxAge(maxAge time.Duration) Option {
	return func(cache *Cache) {
		cache.localCacheOptions = append(cache.localCacheOptions, lru.MaxAge(maxAge))
	}
}

// MaxMemory limits the memory used by the cache.
func MaxMemory(bytes int) Option {
	return func(cache *Cache) {
		cache.localCacheOptions = append(cache.localCacheOptions, lru.MaxWeight(bytes))
	}
}

// MaxMemoryMB limits the memory used by the cache.
func MaxMemoryMB(megaBytes int) Option {
	return func(cache *Cache) {
		cache.localCacheOptions = append(cache.localCacheOptions, lru.MaxWeight(megaBytes*1024*1024))
	}
}

// StrictLoad indicates whether or not to check with all peers for consistency before returning an
// element from the cache.
// This option impacts performance and is off by default.
func StrictLoad(strict bool) Option {
	return func(cache *Cache) {
		cache.strictLoad = strict
	}
}

// RescueOnClose indicates whether or not to offload the content of the cache onto peers when it is closed.
// This option is on by default.
func RescueOnClose(rescue bool) Option {
	return func(cache *Cache) {
		cache.rescueOnClose = rescue
	}
}
