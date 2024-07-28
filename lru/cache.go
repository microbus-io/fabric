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

import (
	"errors"
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

// NewCache creates a new LRU cache with a weight capacity of 16384 and a maximum age of 1hr.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		lookup:    make(map[K]*node[K, V]),
		maxWeight: 16384,
		maxAge:    time.Hour,
	}
}

// Clear empties the cache but does not reset the hit/miss statistics.
func (c *Cache[K, V]) Clear() {
	c.lock.Lock()
	c.lookup = make(map[K]*node[K, V])
	c.weight = 0
	c.newest = nil
	c.oldest = nil
	c.lock.Unlock()
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
	c.delete(key)
	c.store(key, value, opts)
	c.lock.Unlock()
}

func (c *Cache[K, V]) store(key K, value V, opts cacheOptions) {
	// Create and store the node
	nd := &node[K, V]{
		key:      key,
		value:    value,
		weight:   opts.Weight,
		inserted: time.Now().Add(c.timeOffset),
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
	value, ok = c.load(key, opts)
	c.lock.Unlock()
	return value, ok
}

func (c *Cache[K, V]) load(key K, opts cacheOptions) (value V, ok bool) {
	nd, ok := c.lookup[key]
	if ok && time.Now().Add(c.timeOffset).Sub(nd.inserted) > opts.MaxAge {
		c.delete(key)
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
		nd.inserted = time.Now().Add(c.timeOffset) // Bumping renews the life of the element
	}
	return nd.value, true
}

// LoadOrStore looks up an element in the cache.
// If the element is not found, the new value is stored and returned instead.
func (c *Cache[K, V]) LoadOrStore(key K, newValue V, options ...Option) (value V, found bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
		MaxAge: c.maxAge,
	}
	for _, opt := range options {
		opt(&opts)
	}
	c.lock.Lock()
	value, found = c.load(key, opts)
	if !found {
		c.store(key, newValue, opts)
		value = newValue
	}
	c.lock.Unlock()
	return value, found
}

// Delete removes an element from the cache by key.
func (c *Cache[K, V]) Delete(key K) {
	c.lock.Lock()
	c.delete(key)
	c.lock.Unlock()
}

// Delete removes elements from the cache whose keys match the predicate function.
func (c *Cache[K, V]) DeletePredicate(predicate func(key K) bool) {
	c.lock.Lock()
	toDelete := []K{}
	for k := range c.lookup {
		if predicate(k) {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		c.delete(k)
	}
	c.lock.Unlock()
}

func (c *Cache[K, V]) delete(key K) {
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
// If not specified, elements default to a weight of 1.
func (c *Cache[K, V]) SetMaxWeight(weight int) error {
	if weight < 0 {
		return errors.New("negative weight")
	}
	c.lock.Lock()
	reduced := weight < c.maxWeight
	c.maxWeight = weight
	if reduced {
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
	if ttl <= 0 {
		return errors.New("non-positive TTL")
	}
	c.lock.Lock()
	c.maxAge = ttl
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

// cohesion is used for testing purposes only.
func (c *Cache[K, V]) cohesion() bool {
	a := []K{}
	count := 0
	for nd := c.newest; nd != nil; nd = nd.older {
		a = append(a, nd.key)
		if c.lookup[nd.key] != nd {
			return false
		}
		count++
		if count > 1000000 {
			return false
		}
	}
	if len(a) != len(c.lookup) {
		return false
	}
	count = 0
	for nd := c.oldest; nd != nil; nd = nd.newer {
		if len(a) == 0 {
			return false
		}
		if a[len(a)-1] != nd.key {
			return false
		}
		a = a[:len(a)-1]
		count++
		if count > 1000000 {
			return false
		}
	}
	return true
}
