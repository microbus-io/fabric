package lru

import (
	"errors"
	"sync"
	"time"

	"github.com/microbus-io/fabric/clock"
)

const numBuckets = 8 // Hard-coded 8 buckets

// element is a container for a single value and its weight.
type element[V any] struct {
	val    V
	weight int
}

// bucket holds a portion of the elements in the cache.
// There are 8 buckets. Bucket 0 always holds the freshest elements.
// Bucket 0 will be cycled to become bucket 1 when it fills up or when an
// amount of time has lapsed.
type bucket[K comparable, V any] struct {
	lookup map[K]*element[V]
	weight int
}

// newBucket initializes a new bucket.
func newBucket[K comparable, V any]() *bucket[K, V] {
	return &bucket[K, V]{
		lookup: map[K]*element[V]{},
	}
}

// Cache is an LRU cache that enforces a maximum weight capacity and age limit for its elements.
// The LRU cache performs locking internally and is thread-safe.
type Cache[K comparable, V any] struct {
	buckets       []*bucket[K, V]
	nextCycle     time.Time
	cycleDuration time.Duration
	lock          sync.Mutex
	weight        int
	maxWeight     int
	maxAge        time.Duration
	clock         clock.Clock
}

// NewCache creates a new LRU cache with a weight capacity of 16384 and a maximum age of 1hr.
func NewCache[K comparable, V any]() *Cache[K, V] {
	c := &Cache[K, V]{
		buckets:   []*bucket[K, V]{},
		maxWeight: 16384,
		maxAge:    time.Hour,
		clock:     clock.NewClock(),
	}

	// Prepare buckets
	nb := numBuckets
	for b := 0; b < nb; b++ {
		c.buckets = append(c.buckets, newBucket[K, V]())
	}

	// Calc next cycle time
	c.cycleDuration = c.maxAge / time.Duration(nb)
	c.nextCycle = c.clock.Now().Add(c.cycleDuration)

	return c
}

func (c *Cache[K, V]) setClock(mock clock.Clock) {
	c.clock = mock
	c.nextCycle = c.clock.Now().Add(c.cycleDuration)
}

// Clear empties the cache.
func (c *Cache[K, V]) Clear() {
	c.lock.Lock()
	nb := len(c.buckets)
	c.buckets = []*bucket[K, V]{}
	for b := 0; b < nb; b++ {
		c.buckets = append(c.buckets, newBucket[K, V]())
	}
	c.nextCycle = c.clock.Now().Add(c.cycleDuration)
	c.weight = 0
	c.lock.Unlock()
}

// cycleOnce cycles the buckets once.
// Bucket 0 will become 1, 1 will become 2, etc.
// Bucket 8 drops off the tail.
// A new bucket 0 is initialized to hold the freshest elements.
func (c *Cache[K, V]) cycleOnce() {
	nb := len(c.buckets)
	c.weight -= c.buckets[nb-1].weight
	for b := nb - 2; b >= 0; b-- {
		c.buckets[b+1] = c.buckets[b]
	}
	c.buckets[0] = newBucket[K, V]()
	c.nextCycle = c.nextCycle.Add(c.cycleDuration)
}

// cycleAge cycles the cache to gradually evict buckets holding old elements.
func (c *Cache[K, V]) cycleAge() {
	nb := len(c.buckets)
	now := c.clock.Now()
	cycled := 0
	for i := 0; i < nb && !c.nextCycle.After(now); i++ {
		c.cycleOnce()
		cycled++
	}
	if cycled == nb {
		// Cache is fully cleared
		c.nextCycle = c.clock.Now().Add(c.cycleDuration)
	}
}

// Store inserts an element to the cache.
// The weight must be 1 or greater and cannot exceed the cache's maximum weight limit.
func (c *Cache[K, V]) Store(key K, value V, options ...StoreOption) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	if opts.Weight > c.maxWeight {
		// Too heavy for this cache
		return
	}
	c.lock.Lock()
	c.cycleAge()
	c.store(key, value, opts)
	c.lock.Unlock()
}

func (c *Cache[K, V]) store(key K, value V, opts cacheOptions) {
	// Remove the element from all buckets
	c.delete(key)

	// Cycle once if bucket 0 is too full
	nb := len(c.buckets)
	if c.buckets[0].weight >= c.maxWeight/nb {
		c.cycleOnce()
	}

	// Clear older buckets if max weight would be exceeded
	for b := nb - 1; b >= 0 && c.weight+opts.Weight > c.maxWeight; b-- {
		c.weight -= c.buckets[b].weight
		c.buckets[b].lookup = map[K]*element[V]{}
		c.buckets[b].weight = 0
	}

	// Insert to bucket 0
	c.buckets[0].lookup[key] = &element[V]{
		val:    value,
		weight: opts.Weight,
	}
	c.buckets[0].weight += opts.Weight
	c.weight += opts.Weight
}

// Exists indicates if the key is in the cache.
func (c *Cache[K, V]) Exists(key K) bool {
	_, ok := c.Load(key, NoBump())
	return ok
}

// Load looks up an element in the cache.
func (c *Cache[K, V]) Load(key K, options ...LoadOption) (value V, ok bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	c.lock.Lock()
	c.cycleAge()
	value, ok = c.load(key, opts)
	c.lock.Unlock()
	return value, ok
}

func (c *Cache[K, V]) load(key K, opts cacheOptions) (value V, ok bool) {
	// Scan from newest to oldest
	nb := len(c.buckets)
	var foundIn int
	var elem *element[V]
	var found bool
	for b := 0; b < nb; b++ {
		elem, found = c.buckets[b].lookup[key]
		if found {
			foundIn = b
			break
		}
	}
	if !found {
		return value, false
	}
	if !opts.Bump || foundIn == 0 {
		return elem.val, true
	}

	// Remove the element from the older bucket
	delete(c.buckets[foundIn].lookup, key)
	c.buckets[foundIn].weight -= elem.weight
	c.weight -= elem.weight

	// Cycle once if bucket 0 is too full
	if c.buckets[0].weight >= c.maxWeight/nb {
		c.cycleOnce()
	}

	// Insert the element to bucket 0
	c.buckets[0].lookup[key] = elem
	c.buckets[0].weight += elem.weight
	c.weight += elem.weight

	return elem.val, true
}

// LoadOrStore looks up an element in the cache.
// If the element is not found, the new value is stored and returned instead.
func (c *Cache[K, V]) LoadOrStore(key K, newValue V, options ...LoadOrStoreOption) (value V, found bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	c.lock.Lock()
	c.cycleAge()
	value, found = c.load(key, opts)
	if !found {
		c.store(key, newValue, opts)
		value = newValue
	}
	c.lock.Unlock()
	return value, found
}

// Delete removes an element from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.lock.Lock()
	c.cycleAge()
	c.delete(key)
	c.lock.Unlock()
}

func (c *Cache[K, V]) delete(key K) {
	// Remove the element from all buckets
	nb := len(c.buckets)
	for b := 0; b < nb; b++ {
		e, ok := c.buckets[b].lookup[key]
		if ok {
			c.buckets[b].weight -= e.weight
			c.weight -= e.weight
			delete(c.buckets[b].lookup, key)
		}
	}
}

// Weight returns the total weight of all the elements in the cache.
func (c *Cache[K, V]) Weight() int {
	c.lock.Lock()
	c.cycleAge()
	weight := c.weight
	c.lock.Unlock()
	return weight
}

// Len returns the number of elements in the cache.
func (c *Cache[K, V]) Len() int {
	c.lock.Lock()
	c.cycleAge()
	count := 0
	nb := len(c.buckets)
	for b := 0; b < nb; b++ {
		count += len(c.buckets[b].lookup)
	}
	c.lock.Unlock()
	return count
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
		// Clear older buckets if max weight would be exceeded
		nb := len(c.buckets)
		for b := nb - 1; b >= 0 && c.weight > c.maxWeight; b-- {
			c.weight -= c.buckets[b].weight
			c.buckets[b].lookup = map[K]*element[V]{}
			c.buckets[b].weight = 0
		}
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
	if ttl < 0 {
		return errors.New("negative TTL")
	}
	c.lock.Lock()
	c.maxAge = ttl
	nb := len(c.buckets)
	c.cycleDuration = c.maxAge / time.Duration(nb)
	c.nextCycle = c.clock.Now().Add(c.cycleDuration)
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
	m := map[K]V{}
	nb := len(c.buckets)
	for b := 0; b < nb; b++ {
		for k, elem := range c.buckets[b].lookup {
			m[k] = elem.val
		}
	}
	c.lock.Unlock()
	return m
}
