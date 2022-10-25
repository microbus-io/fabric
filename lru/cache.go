package lru

import (
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
	options       cacheOptions
	buckets       []*bucket[K, V]
	nextCycle     time.Time
	cycleDuration time.Duration
	lock          sync.Mutex
	weight        int
}

// NewCache creates a new LRU cache.
func NewCache[K comparable, V any](options ...Option) *Cache[K, V] {
	cache := &Cache[K, V]{
		buckets: []*bucket[K, V]{},
		options: cacheOptions{
			maxWeight:  10000,
			maxAge:     time.Hour,
			bumpOnLoad: true,
			clock:      clock.New(),
		},
	}
	for _, opt := range options {
		opt(&cache.options)
	}

	// Prepare buckets
	nb := numBuckets
	for b := 0; b < nb; b++ {
		cache.buckets = append(cache.buckets, newBucket[K, V]())
	}

	// Calc next cycle time
	cache.cycleDuration = cache.options.maxAge / time.Duration(nb)
	cache.nextCycle = cache.options.clock.Now().Add(cache.cycleDuration)

	return cache
}

// Clear empties the cache.
func (cache *Cache[K, V]) Clear() {
	cache.lock.Lock()
	nb := len(cache.buckets)
	cache.buckets = []*bucket[K, V]{}
	for b := 0; b < nb; b++ {
		cache.buckets = append(cache.buckets, newBucket[K, V]())
	}
	cache.nextCycle = cache.options.clock.Now().Add(cache.cycleDuration)
	cache.weight = 0
	cache.lock.Unlock()
}

// cycleOnce cycles the buckets once.
// Bucket 0 will become 1, 1 will become 2, etc.
// Bucket 8 drops off the tail.
// A new bucket 0 is initialized to hold the freshest elements.
func (cache *Cache[K, V]) cycleOnce() {
	nb := len(cache.buckets)
	cache.weight -= cache.buckets[nb-1].weight
	for b := nb - 2; b >= 0; b-- {
		cache.buckets[b+1] = cache.buckets[b]
	}
	cache.buckets[0] = newBucket[K, V]()
}

// cycleAge cycles the cache to gradually evict buckets holding old elements.
func (cache *Cache[K, V]) cycleAge() {
	nb := len(cache.buckets)
	now := cache.options.clock.Now()
	cycled := 0
	for i := 0; i < nb && !cache.nextCycle.After(now); i++ {
		cache.cycleOnce()
		cache.nextCycle = cache.nextCycle.Add(cache.cycleDuration)
		cycled++
	}
	if cycled == nb {
		// Cache is fully cleared
		cache.nextCycle = cache.options.clock.Now().Add(cache.cycleDuration)
	}
}

// Put stores an element with a weight of 1.
func (cache *Cache[K, V]) Put(key K, value V) {
	cache.Store(key, value, 1)
}

// Store inserts an element to the cache with the indicated weight.
// The weight must be 1 or greater and cannot exceed the cache's maximum weight limit.
func (cache *Cache[K, V]) Store(key K, value V, weight int) {
	if weight > cache.options.maxWeight {
		return
	}
	if weight < 1 {
		weight = 1
	}

	cache.lock.Lock()
	cache.cycleAge()
	cache.store(key, value, weight)
	cache.lock.Unlock()
}

func (cache *Cache[K, V]) store(key K, value V, weight int) {
	// Remove the element from all buckets
	cache.delete(key)

	// Cycle if cache weight is exceeded
	for cache.weight+weight > cache.options.maxWeight {
		cache.cycleOnce()
		cache.nextCycle = cache.nextCycle.Add(cache.cycleDuration)
	}

	// Insert to bucket 0
	cache.buckets[0].lookup[key] = &element[V]{
		val:    value,
		weight: weight,
	}
	cache.buckets[0].weight += weight
	cache.weight += weight
}

// Load looks up an element in the cache.
func (cache *Cache[K, V]) Load(key K) (value V, ok bool) {
	cache.lock.Lock()
	cache.cycleAge()
	value, ok = cache.load(key)
	cache.lock.Unlock()
	return value, ok
}

func (cache *Cache[K, V]) load(key K) (value V, ok bool) {
	// Scan from newest to oldest
	nb := len(cache.buckets)
	var foundIn int
	var elem *element[V]
	var found bool
	for b := 0; b < nb; b++ {
		elem, found = cache.buckets[b].lookup[key]
		if found {
			foundIn = b
			break
		}
	}
	if !found {
		return value, false
	}
	if !cache.options.bumpOnLoad || foundIn == 0 {
		return elem.val, true
	}

	// Remove the element from the older bucket
	delete(cache.buckets[foundIn].lookup, key)
	cache.buckets[foundIn].weight -= elem.weight

	// Insert the element to bucket 0
	cache.buckets[0].lookup[key] = elem
	cache.buckets[0].weight += elem.weight

	return elem.val, true
}

// LoadOrPut looks up an element in the cache.
// If the element is not found, the new value is stored and returned instead.
func (cache *Cache[K, V]) LoadOrPut(key K, newValue V) (value V, found bool) {
	return cache.LoadOrStore(key, newValue, 1)
}

// LoadOrStore looks up an element in the cache.
// If the element is not found, the new value is stored and returned instead.
func (cache *Cache[K, V]) LoadOrStore(key K, newValue V, weight int) (value V, found bool) {
	cache.lock.Lock()
	cache.cycleAge()
	value, found = cache.load(key)
	if !found {
		cache.store(key, newValue, weight)
		value = newValue
	}
	cache.lock.Unlock()
	return value, found
}

// Delete removes an element from the cache.
func (cache *Cache[K, V]) Delete(key K) {
	cache.lock.Lock()
	cache.cycleAge()
	cache.delete(key)
	cache.lock.Unlock()
}

func (cache *Cache[K, V]) delete(key K) {
	// Remove the element from all buckets
	nb := len(cache.buckets)
	for b := 0; b < nb; b++ {
		e, ok := cache.buckets[b].lookup[key]
		if ok {
			cache.buckets[b].weight -= e.weight
			cache.weight -= e.weight
			delete(cache.buckets[b].lookup, key)
		}
	}
}

// Weight returns the total weight of all the elements in the cache.
func (cache *Cache[K, V]) Weight() int {
	cache.lock.Lock()
	cache.cycleAge()
	weight := cache.weight
	cache.lock.Unlock()
	return weight
}

// Len returns the number of elements in the cache.
func (cache *Cache[K, V]) Len() int {
	cache.lock.Lock()
	cache.cycleAge()
	count := 0
	nb := len(cache.buckets)
	for b := 0; b < nb; b++ {
		count += len(cache.buckets[b].lookup)
	}
	cache.lock.Unlock()
	return count
}

// MaxWeight returns the weight limit set for this cache.
func (cache *Cache[K, V]) MaxWeight() int {
	return cache.options.maxWeight
}

// MaxWeight returns the age limit set for this cache.
func (cache *Cache[K, V]) MaxAge() time.Duration {
	return cache.options.maxAge
}

// IsBumpOnLoad returns whether bump on load is enabled or not.
func (cache *Cache[K, V]) IsBumpOnLoad() bool {
	return cache.options.bumpOnLoad
}

// ToMap returns the elements currently in the cache in a newly allocated map.
func (cache *Cache[K, V]) ToMap() map[K]V {
	cache.lock.Lock()
	m := map[K]V{}
	nb := len(cache.buckets)
	for b := 0; b < nb; b++ {
		for k, elem := range cache.buckets[b].lookup {
			m[k] = elem.val
		}
	}
	cache.lock.Unlock()
	return m
}
