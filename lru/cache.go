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

package lru

import (
	"sync"
	"time"
)

// node is a container for a single value, its weight and expiration.
type node[K comparable, V any] struct {
	key      K
	value    V
	weight   int
	inserted time.Time
	newer    *node[K, V]
	older    *node[K, V]
}

// Cache is an LRU cache that enforces a maximum weight capacity and age limit for its elements.
// The LRU cache performs locking internally and is thread-safe.
// It also keeps track of the hit/miss statistics.
type Cache[K comparable, V any] struct {
	lookup     map[K]*node[K, V]
	newest     *node[K, V]
	oldest     *node[K, V]
	weight     int
	maxWeight  int
	maxAge     time.Duration
	hits       int
	misses     int
	lock       sync.Mutex
	timeOffset time.Duration // For testing only
}

// New creates a new LRU cache.
func New[K comparable, V any](maxWeight int, maxAge time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		lookup:    make(map[K]*node[K, V]),
		maxWeight: maxWeight,
		maxAge:    maxAge,
	}
}

// Clear empties the cache but does not reset the hit/miss statistics.
func (c *Cache[K, V]) Clear() {
	c.lock.Lock()
	c.clear()
	c.lock.Unlock()
}

func (c *Cache[K, V]) clear() {
	c.lookup = make(map[K]*node[K, V])
	c.weight = 0
	c.newest = nil
	c.oldest = nil
}

// Store inserts an element to the front of the cache.
// The weight must be 1 or greater and cannot exceed the cache's maximum weight limit.
func (c *Cache[K, V]) Store(key K, value V, options ...Option) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
		MaxAge: c.maxAge,
	}
	for _, opt := range options {
		opt(&opts)
	}
	if opts.Weight > c.maxWeight {
		// Too heavy for this cache
		return
	}
	c.lock.Lock()
	now := c.now()
	c.delete(key, now)
	c.store(key, value, opts, now)
	c.lock.Unlock()
}

func (c *Cache[K, V]) store(key K, value V, opts cacheOptions, now time.Time) {
	if c.maxWeight <= 0 || c.maxAge <= 0 {
		return
	}
	c.maybeTrimOldest(2, now)
	// Create and store the node
	nd := &node[K, V]{
		key:      key,
		value:    value,
		weight:   opts.Weight,
		inserted: now,
		older:    c.newest,
	}
	c.lookup[key] = nd
	if c.newest != nil {
		c.newest.newer = nd
	}
	c.newest = nd
	if c.oldest == nil {
		c.oldest = nd
	}
	c.weight += nd.weight

	c.diet()
}

func (c *Cache[K, V]) diet() {
	for c.weight > c.maxWeight && c.oldest != nil {
		oldest := c.oldest
		c.oldest = oldest.newer
		oldest.newer = nil
		delete(c.lookup, oldest.key)
		c.weight -= oldest.weight
	}
	if c.oldest != nil {
		c.oldest.older = nil
	} else {
		c.newest = nil
	}
}

// now returns the current time adjusted by the test-only offset. It is computed once per public operation and
// threaded through the internal helpers, so a single load or store reads the clock exactly once.
func (c *Cache[K, V]) now() time.Time {
	return time.Now().Add(c.timeOffset)
}

// maybeTrimOldest reaps up to k expired entries from the oldest (tail) end of the cache. Store and bump stamp the
// current time and move the node to the front, so the list is ordered by insertion time with the oldest at the tail:
// if the oldest entry has not exceeded the cache's max age, none have, and the scan stops. Expiry is measured against
// the cache-wide max age only - an item read with a shorter per-call max age is treated as expired by that read but
// not physically reclaimed until it exceeds the max age. k bounds the work so a single operation never stalls on a
// large backlog; the rest is reaped by later operations. Must not call delete, which calls this.
func (c *Cache[K, V]) maybeTrimOldest(k int, now time.Time) {
	if c.maxAge <= 0 {
		return
	}
	for range k {
		if c.oldest == nil || now.Sub(c.oldest.inserted) <= c.maxAge {
			return
		}
		oldest := c.oldest
		delete(c.lookup, oldest.key)
		c.oldest = oldest.newer
		if c.oldest != nil {
			c.oldest.older = nil
		} else {
			c.newest = nil
		}
		c.weight -= oldest.weight
		oldest.newer = nil
	}
}

// Exists indicates if the key is in the cache.
func (c *Cache[K, V]) Exists(key K) bool {
	_, ok := c.Load(key, NoBump())
	return ok
}

// Load looks up an element in the cache.
func (c *Cache[K, V]) Load(key K, options ...Option) (value V, ok bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
		MaxAge: c.maxAge,
	}
	for _, opt := range options {
		opt(&opts)
	}
	c.lock.Lock()
	value, ok = c.load(key, opts, c.now())
	c.lock.Unlock()
	return value, ok
}

func (c *Cache[K, V]) load(key K, opts cacheOptions, now time.Time) (value V, ok bool) {
	c.maybeTrimOldest(2, now)
	nd, ok := c.lookup[key]
	if ok && now.Sub(nd.inserted) > opts.MaxAge {
		c.delete(key, now)
		ok = false
	}
	if !ok {
		c.misses++
		if c.misses < 0 { // Overflow
			c.misses = 1
			c.hits = 0
		}
		return value, false
	}
	c.hits++
	if c.hits < 0 { // Overflow
		c.hits = 1
		c.misses = 0
	}

	if opts.Bump {
		if nd.newer != nil {
			nd.newer.older = nd.older
		} else {
			c.newest = nd.older
		}
		if nd.older != nil {
			nd.older.newer = nd.newer
		} else {
			c.oldest = nd.newer
		}
		nd.newer = nil
		nd.older = c.newest
		if c.newest != nil {
			c.newest.newer = nd
		}
		c.newest = nd
		nd.inserted = now // Bumping renews the life of the element
	}
	return nd.value, true
}

// LoadOrStore looks up an element in the cache.
// If the element is not found, the new value is stored instead.
func (c *Cache[K, V]) LoadOrStore(key K, newValue V, options ...Option) (value V, found bool) {
	return c.LoadOrStoreFunc(key, func() V { return newValue }, options...)
}

// LoadOrStoreFunc looks up an element in the cache.
// If the element is not found, the new value is created and stored instead.
func (c *Cache[K, V]) LoadOrStoreFunc(key K, newValue func() V, options ...Option) (value V, found bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
		MaxAge: c.maxAge,
	}
	for _, opt := range options {
		opt(&opts)
	}
	c.lock.Lock()
	now := c.now()
	value, found = c.load(key, opts, now)
	if !found {
		value = newValue()
		c.store(key, value, opts, now)
	}
	c.lock.Unlock()
	return value, found
}

// Delete removes an element from the cache by key.
func (c *Cache[K, V]) Delete(key K) {
	c.lock.Lock()
	c.delete(key, c.now())
	c.lock.Unlock()
}

// Delete removes elements from the cache whose keys match the predicate function.
func (c *Cache[K, V]) DeletePredicate(predicate func(key K) bool) {
	c.lock.Lock()
	now := c.now()
	toDelete := []K{}
	for k := range c.lookup {
		if predicate(k) {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		c.delete(k, now)
	}
	c.lock.Unlock()
}

func (c *Cache[K, V]) delete(key K, now time.Time) {
	nd, ok := c.lookup[key]
	if !ok {
		return
	}
	delete(c.lookup, key)
	if nd.newer != nil {
		nd.newer.older = nd.older
	} else {
		c.newest = nd.older
	}
	if nd.older != nil {
		nd.older.newer = nd.newer
	} else {
		c.oldest = nd.newer
	}
	nd.older = nil
	nd.newer = nil
	c.weight -= nd.weight
	c.maybeTrimOldest(2, now)
}

// Weight returns the total weight of all the elements in the cache.
func (c *Cache[K, V]) Weight() int {
	c.lock.Lock()
	w := c.weight
	c.lock.Unlock()
	return w
}

// Len returns the number of elements in the cache.
func (c *Cache[K, V]) Len() int {
	c.lock.Lock()
	l := len(c.lookup)
	c.lock.Unlock()
	return l
}

// Hits returns the total number of cache hits.
// This number can technically overflow.
func (c *Cache[K, V]) Hits() int {
	c.lock.Lock()
	hits := c.hits
	c.lock.Unlock()
	return hits
}

// Misses returns the total number of cache misses.
// This number can technically overflow.
func (c *Cache[K, V]) Misses() int {
	c.lock.Lock()
	misses := c.misses
	c.lock.Unlock()
	return misses
}

// SetMaxAge sets the total weight limit of elements in this cache.
func (c *Cache[K, V]) SetMaxWeight(weight int) error {
	c.lock.Lock()
	reduced := weight < c.maxWeight
	c.maxWeight = weight
	if weight <= 0 {
		c.clear()
	} else if reduced {
		c.diet()
	}
	c.lock.Unlock()
	return nil
}

// MaxWeight returns the weight limit set for this cache.
func (c *Cache[K, V]) MaxWeight() int {
	c.lock.Lock()
	weight := c.maxWeight
	c.lock.Unlock()
	return weight
}

// SetMaxAge sets the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (c *Cache[K, V]) SetMaxAge(ttl time.Duration) error {
	c.lock.Lock()
	reduced := ttl < c.maxAge
	c.maxAge = ttl
	if ttl <= 0 {
		c.clear()
	} else if reduced {
		c.diet()
	}
	c.lock.Unlock()
	return nil
}

// MaxAge returns the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (c *Cache[K, V]) MaxAge() time.Duration {
	c.lock.Lock()
	ttl := c.maxAge
	c.lock.Unlock()
	return ttl
}

// ToMap returns the elements currently in the cache in a newly allocated map.
func (c *Cache[K, V]) ToMap() map[K]V {
	c.lock.Lock()
	m := make(map[K]V, len(c.lookup))
	for k, v := range c.lookup {
		m[k] = v.value
	}
	c.lock.Unlock()
	return m
}
