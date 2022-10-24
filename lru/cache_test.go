package lru

import (
	"testing"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestLRU_Lookup(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Put("a", "aaa")
	cache.Put("b", "bbb")
	cache.Put("c", "ccc")

	v, ok := cache.Load("a")
	assert.True(t, ok)
	assert.Equal(t, "aaa", v)

	v, ok = cache.Load("b")
	assert.True(t, ok)
	assert.Equal(t, "bbb", v)

	v, ok = cache.Load("c")
	assert.True(t, ok)
	assert.Equal(t, "ccc", v)

	v, ok = cache.Load("d")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestLRU_WeightLimit(t *testing.T) {
	t.Parallel()

	maxWt := 2 * numBuckets
	cache := NewCache[int, string](
		MaxWeight(maxWt),
		BumpOnLoad(false),
	)

	cache.Store(999, "too big", 3)
	_, ok := cache.Load(999)
	assert.False(t, ok)
	assert.Equal(t, 0, cache.Weight())

	for i := 1; i <= maxWt+1; i++ {
		cache.Store(i, "Foo", 1)
	}

	// First two elements should get evicted with the oldest bucket
	_, ok = cache.Load(1)
	assert.False(t, ok)
	_, ok = cache.Load(2)
	assert.False(t, ok)
	for i := 3; i <= maxWt+1; i++ {
		_, ok = cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, maxWt-1, cache.Weight())

	// There should be room for one more element
	cache.Store(maxWt+2, "Foo", 1)
	_, ok = cache.Load(1)
	assert.False(t, ok)
	_, ok = cache.Load(2)
	assert.False(t, ok)
	for i := 3; i <= maxWt+2; i++ {
		_, ok = cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, maxWt, cache.Weight())
}

func TestLRU_Clear(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	assert.Equal(t, 0, cache.Len())
	assert.Equal(t, 0, cache.Weight())

	n := 6
	sum := 0
	for i := 1; i <= n; i++ {
		cache.Store(i, "X", i)
		sum += i
	}
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		assert.True(t, ok)
		assert.Equal(t, "X", v)
	}
	assert.Equal(t, n, cache.Len())
	assert.Equal(t, sum, cache.Weight())

	cache.Clear()
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		assert.False(t, ok)
		assert.Equal(t, "", v)
	}
	assert.Equal(t, 0, cache.Len())
	assert.Equal(t, 0, cache.Weight())
}

func TestLRU_Delete(t *testing.T) {
	t.Parallel()

	span := 10
	sim := map[int]string{}
	cache := NewCache[int, string]()
	for i := 0; i < 2048; i++ {
		n := rand.Intn(span * 2)
		if n >= span {
			delete(sim, n-span)
			cache.Delete(n - span)

			_, ok := cache.Load(n - span)
			assert.False(t, ok)
		} else {
			sim[n] = "X"
			cache.Put(n, "X")

			_, ok := cache.Load(n)
			assert.True(t, ok)
		}
	}

	for i := 0; i < span; i++ {
		v, _ := cache.Load(i)
		assert.Equal(t, sim[i], v)
	}
}

func TestLRU_MaxAge(t *testing.T) {
	t.Parallel()

	seconds := numBuckets * 10 // 80 seconds
	clock := clock.NewMock()
	cache := NewCache[int, string](
		MaxAge(time.Second*time.Duration(seconds)),
		BumpOnLoad(false),
		mockClock(clock),
	)

	for i := 1; i <= seconds-1; i++ {
		cache.Put(i, "X")
		clock.Add(time.Second)
	}

	for i := 1; i <= seconds-1; i++ {
		_, ok := cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, seconds-1, cache.Len())

	// The 80th second will cause the oldest bucket to be evicted
	clock.Add(time.Second)
	for i := 11; i <= seconds-1; i++ {
		_, ok := cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, seconds-1-10, cache.Len())

	// Another 10 seconds will remove the next bucket
	clock.Add(10 * time.Second)
	for i := 21; i <= seconds-1; i++ {
		_, ok := cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, seconds-1-20, cache.Len())

	// Another 9 seconds should not cause an eviction
	clock.Add(9 * time.Second)
	assert.Equal(t, seconds-1-20, cache.Len())

	// Fast forward should clear the entire cache
	clock.Add(time.Second * time.Duration(seconds))
	assert.Equal(t, 0, cache.Len())
}

func TestLRU_BumpOnLoad(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string](
		MaxWeight(numBuckets), // 1 element per bucket
		BumpOnLoad(true),
	)

	// Fill in the cache
	// head> 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= numBuckets; i++ {
		cache.Put(i, "X")
	}
	assert.Equal(t, numBuckets, cache.Len())

	// Loading element 1 should bump it to the top of the cache
	// head> 1 8 7 6 5 4 3 2 <tail
	_, ok := cache.Load(1)
	assert.True(t, ok)
	assert.Equal(t, numBuckets, cache.Len())

	// Loading element 8 should bump it to the top of the cache ahead of 1.
	// Element 2 will drop off the tail.
	// head> 8 1 _ 7 6 5 4 3 <tail
	_, ok = cache.Load(8)
	assert.True(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())
	_, ok = cache.Load(2)
	assert.False(t, ok)

	// Loading element 1 should bump it to the top of the cache
	// Element 3 will drop off the tail.
	// head> 1 8 _ _ 7 6 5 4 <tail
	_, ok = cache.Load(1)
	assert.True(t, ok)
	_, ok = cache.Load(3)
	assert.False(t, ok)
}

func BenchmarkLRU_Store(b *testing.B) {
	cache := NewCache[int, int](
		MaxWeight(b.N * 2),
	)
	for i := 0; i < b.N; i++ {
		cache.Put(i, i)
	}

	// On 2021 MacBook Pro M1 15":
	// 315 ns/op
}

func BenchmarkLRU_LoadNoBump(b *testing.B) {
	cache := NewCache[int, int](
		MaxWeight(b.N*2),
		BumpOnLoad(false),
	)
	for i := 0; i < b.N; i++ {
		cache.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Load(i)
	}

	// On 2021 MacBook Pro M1 15":
	// 245 ns/op
}

func BenchmarkLRU_LoadWithBump(b *testing.B) {
	cache := NewCache[int, int](
		MaxWeight(b.N*2),
		BumpOnLoad(true),
	)
	for i := 0; i < b.N; i++ {
		cache.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Load(i)
	}

	// On 2021 MacBook Pro M1 15":
	// 440 ns/op
}
