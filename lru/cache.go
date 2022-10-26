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

// NewCache creates a new LRU cache.
func NewCache[K comparable, V any]() *Cache[K, V] {
	cache := &Cache[K, V]{
		buckets:   []*bucket[K, V]{},
		maxWeight: 16384,
		maxAge:    time.Hour,
		clock:     clock.New(),
	}

	// Prepare buckets
	nb := numBuckets
	for b := 0; b < nb; b++ {
		cache.buckets = append(cache.buckets, newBucket[K, V]())
	}

	// Calc next cycle time
	cache.cycleDuration = cache.maxAge / time.Duration(nb)
	cache.nextCycle = cache.clock.Now().Add(cache.cycleDuration)

	return cache
}

func (cache *Cache[K, V]) setClock(mock clock.Clock) {
	cache.clock = mock
	cache.nextCycle = cache.clock.Now().Add(cache.cycleDuration)
}

// Clear empties the cache.
func (cache *Cache[K, V]) Clear() {
	cache.lock.Lock()
	nb := len(cache.buckets)
	cache.buckets = []*bucket[K, V]{}
	for b := 0; b < nb; b++ {
		cache.buckets = append(cache.buckets, newBucket[K, V]())
	}
	cache.nextCycle = cache.clock.Now().Add(cache.cycleDuration)
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
	cache.nextCycle = cache.nextCycle.Add(cache.cycleDuration)
}

// cycleAge cycles the cache to gradually evict buckets holding old elements.
func (cache *Cache[K, V]) cycleAge() {
	nb := len(cache.buckets)
	now := cache.clock.Now()
	cycled := 0
	for i := 0; i < nb && !cache.nextCycle.After(now); i++ {
		cache.cycleOnce()
		cycled++
	}
	if cycled == nb {
		// Cache is fully cleared
		cache.nextCycle = cache.clock.Now().Add(cache.cycleDuration)
	}
}

// Store inserts an element to the cache.
// The weight must be 1 or greater and cannot exceed the cache's maximum weight limit.
func (cache *Cache[K, V]) Store(key K, value V, options ...StoreOption) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	if opts.Weight > cache.maxWeight {
		// Too heavy for this cache
		return
	}
	cache.lock.Lock()
	cache.cycleAge()
	cache.store(key, value, opts)
	cache.lock.Unlock()
}

func (cache *Cache[K, V]) store(key K, value V, opts cacheOptions) {
	// Remove the element from all buckets
	cache.delete(key)

	// Cycle once if bucket 0 is too full
	nb := len(cache.buckets)
	if cache.buckets[0].weight >= cache.maxWeight/nb {
		cache.cycleOnce()
	}

	// Clear older buckets if max weight would be exceeded
	for b := nb - 1; b >= 0 && cache.weight+opts.Weight > cache.maxWeight; b-- {
		cache.weight -= cache.buckets[b].weight
		cache.buckets[b].lookup = map[K]*element[V]{}
		cache.buckets[b].weight = 0
	}

	// Insert to bucket 0
	cache.buckets[0].lookup[key] = &element[V]{
		val:    value,
		weight: opts.Weight,
	}
	cache.buckets[0].weight += opts.Weight
	cache.weight += opts.Weight
}

// Exists indicates if the key is in the cache.
func (cache *Cache[K, V]) Exists(key K) bool {
	_, ok := cache.Load(key, NoBump())
	return ok
}

// Load looks up an element in the cache.
func (cache *Cache[K, V]) Load(key K, options ...LoadOption) (value V, ok bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	cache.lock.Lock()
	cache.cycleAge()
	value, ok = cache.load(key, opts)
	cache.lock.Unlock()
	return value, ok
}

func (cache *Cache[K, V]) load(key K, opts cacheOptions) (value V, ok bool) {
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
	if !opts.Bump || foundIn == 0 {
		return elem.val, true
	}

	// Remove the element from the older bucket
	delete(cache.buckets[foundIn].lookup, key)
	cache.buckets[foundIn].weight -= elem.weight
	cache.weight -= elem.weight

	// Cycle once if bucket 0 is too full
	if cache.buckets[0].weight >= cache.maxWeight/nb {
		cache.cycleOnce()
	}

	// Insert the element to bucket 0
	cache.buckets[0].lookup[key] = elem
	cache.buckets[0].weight += elem.weight
	cache.weight += elem.weight

	return elem.val, true
}

// LoadOrStore looks up an element in the cache.
// If the element is not found, the new value is stored and returned instead.
func (cache *Cache[K, V]) LoadOrStore(key K, newValue V, options ...LoadOrStoreOption) (value V, found bool) {
	opts := cacheOptions{
		Weight: 1,
		Bump:   true,
	}
	for _, opt := range options {
		opt(&opts)
	}
	cache.lock.Lock()
	cache.cycleAge()
	value, found = cache.load(key, opts)
	if !found {
		cache.store(key, newValue, opts)
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

// SetMaxAge sets the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore
// survive longer.
func (cache *Cache[K, V]) SetMaxWeight(weight int) error {
	if weight < 0 {
		return errors.New("negative weight")
	}
	cache.lock.Lock()
	reduced := weight < cache.maxWeight
	cache.maxWeight = weight

	if reduced {
		// Clear older buckets if max weight would be exceeded
		nb := len(cache.buckets)
		for b := nb - 1; b >= 0 && cache.weight > cache.maxWeight; b-- {
			cache.weight -= cache.buckets[b].weight
			cache.buckets[b].lookup = map[K]*element[V]{}
			cache.buckets[b].weight = 0
		}
	}

	cache.lock.Unlock()
	return nil
}

// MaxWeight returns the weight limit set for this cache.
func (cache *Cache[K, V]) MaxWeight() int {
	cache.lock.Lock()
	weight := cache.maxWeight
	cache.lock.Unlock()
	return weight
}

// SetMaxAge sets the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (cache *Cache[K, V]) SetMaxAge(ttl time.Duration) error {
	if ttl < 0 {
		return errors.New("negative TTL")
	}
	cache.lock.Lock()
	cache.maxAge = ttl
	nb := len(cache.buckets)
	cache.cycleDuration = cache.maxAge / time.Duration(nb)
	cache.nextCycle = cache.clock.Now().Add(cache.cycleDuration)
	cache.lock.Unlock()
	return nil
}

// MaxAge returns the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (cache *Cache[K, V]) MaxAge() time.Duration {
	cache.lock.Lock()
	ttl := cache.maxAge
	cache.lock.Unlock()
	return ttl
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
