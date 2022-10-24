package lru

import (
	"testing"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestLRU_Load(t *testing.T) {
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

func TestLRU_LoadOrPut(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Put("a", "aaa")

	v, found := cache.LoadOrPut("a", "AAA")
	assert.True(t, found)
	assert.Equal(t, "aaa", v)

	cache.Delete("a")

	v, found = cache.LoadOrPut("a", "AAA")
	assert.False(t, found)
	assert.Equal(t, "AAA", v)

	v, found = cache.Load("a")
	assert.True(t, found)
	assert.Equal(t, "AAA", v)
}

func TestLRU_WeightLimit(t *testing.T) {
	t.Parallel()

	maxWt := 2 * numBuckets
	cache := NewCache[int, string](
		MaxWeight(maxWt),
		BumpOnLoad(false),
	)

	cache.Store(999, "too big", maxWt+1)
	_, ok := cache.Load(999)
	assert.False(t, ok)
	assert.Equal(t, 0, cache.Weight())

	// Fill in the cache
	// head> [13,14,15,16] [9,10,11,12] [5,6,7,8] [1,2,3,4] _ _ _ _ <tail
	for i := 1; i <= maxWt; i += 4 {
		cache.cycleOnce()
		cache.Store(i, "Foo", 1)
		cache.Store(i+1, "Foo", 1)
		cache.Store(i+2, "Foo", 1)
		cache.Store(i+3, "Foo", 1)
	}

	// Oldest bucket should get evicted
	// head> 101 _ _ _ _ [13,14,15,16] [9,10,11,12] [5,6,7,8] <tail
	cache.Store(101, "Foo", 1)
	for i := 1; i < 4; i++ {
		_, ok = cache.Load(i)
		assert.False(t, ok, "%d", i)
	}
	for i := 5; i <= maxWt; i++ {
		_, ok = cache.Load(i)
		assert.True(t, ok, "%d", i)
	}
	assert.Equal(t, maxWt-3, cache.Weight())

	// There should be room for three more elements
	// head> [101,102,103,104] _ _ _ _ [13,14,15,16] [9,10,11,12] [5,6,7,8] <tail
	cache.Store(102, "Foo", 1)
	cache.Store(103, "Foo", 1)
	cache.Store(104, "Foo", 1)
	for i := 1; i < 4; i++ {
		_, ok = cache.Load(i)
		assert.False(t, ok, "%d", i)
	}
	for i := 5; i <= maxWt; i++ {
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
		MaxWeight(numBuckets),
		BumpOnLoad(true),
	)

	// Fill in the cache
	// head> 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= numBuckets; i++ {
		cache.cycleOnce()
		cache.Put(i, "X")
	}
	assert.Equal(t, numBuckets, cache.Len())

	// Loading element 1 should bump it to the top of the cache
	// head> [1,8] 7 6 5 4 3 2 _ <tail
	_, ok := cache.Load(1)
	assert.True(t, ok)
	assert.Equal(t, numBuckets, cache.Len())

	// Loading element 8 should make no difference
	// head> [8,1] 7 6 5 4 3 2 _ <tail
	_, ok = cache.Load(8)
	assert.True(t, ok)
	assert.Equal(t, numBuckets, cache.Len())

	// Putting element 9 should drop 2 off the tail
	cache.Put(9, "X")
	// head> _ _ [9,8,1] 7 6 5 4 3 <tail
	assert.Equal(t, numBuckets, cache.Len())
	_, ok = cache.Load(2)
	assert.False(t, ok)

	// Loading element 4 should bump it to the top of the cache
	// head> 4 _ [9,8,1] 7 6 5 _ 3 <tail
	_, ok = cache.Load(4)
	assert.True(t, ok)
	assert.Equal(t, numBuckets, cache.Len())

	// Cycle once to cause 3 to drop off the tail
	// head> _ 4 _ [9,8,1] 7 6 5 _ <tail
	cache.cycleOnce()
	_, ok = cache.Load(3)
	assert.False(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())

	// Loading element 4 should bump it to the top of the cache
	// head> 4 _ _ [9,8,1] 7 6 5 _ <tail
	_, ok = cache.Load(4)
	assert.True(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())
}

func BenchmarkLRU_Store(b *testing.B) {
	cache := NewCache[int, int](
		MaxWeight(b.N * 2),
	)
	for i := 0; i < b.N; i++ {
		cache.Put(i, i)
	}

	// On 2021 MacBook Pro M1 15":
	// 294 ns/op
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
	// 197 ns/op
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
	// 198 ns/op
}
