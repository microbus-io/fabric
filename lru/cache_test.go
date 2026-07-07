/*
Copyright (c) 2023-2025 Microbus LLC and various contributors

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
	"math/rand/v2"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"
)

func TestLRU_Load(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[string, string](1024, time.Hour)
	cache.Store("a", "aaa")
	cache.Store("b", "bbb")
	cache.Store("c", "ccc")
	assert.True(integrity(cache))

	v, ok := cache.Load("a")
	assert.True(ok)
	assert.Equal("aaa", v)

	v, ok = cache.Load("b")
	assert.True(ok)
	assert.Equal("bbb", v)

	v, ok = cache.Load("c")
	assert.True(ok)
	assert.Equal("ccc", v)

	v, ok = cache.Load("d")
	assert.False(ok)
	assert.Equal("", v)

	m := cache.ToMap()
	assert.NotEqual(0, len(m["a"]))
	assert.NotEqual(0, len(m["b"]))
	assert.NotEqual(0, len(m["c"]))
	assert.Zero(len(m["d"]))

	assert.True(integrity(cache))
}

func TestLRU_LoadOrStore(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[string, string](1024, time.Hour)
	cache.Store("a", "aaa")

	v, found := cache.LoadOrStore("a", "AAA")
	assert.True(found)
	assert.Equal("aaa", v)

	cache.Delete("a")

	v, found = cache.LoadOrStore("a", "AAA")
	assert.False(found)
	assert.Equal("AAA", v)

	v, found = cache.Load("a")
	assert.True(found)
	assert.Equal("AAA", v)

	assert.True(integrity(cache))
}

func TestLRU_MaxWeight(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	maxWt := 16
	cache := New[int, string](maxWt, time.Hour)

	cache.Store(999, "Too Big", Weight(maxWt+1))
	_, ok := cache.Load(999)
	assert.False(ok)
	assert.Zero(cache.Weight())

	// Fill in the cache
	// head> 16 15 14 13 12 11 10 9 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= maxWt; i++ {
		cache.Store(i, "Light", Weight(1))
	}
	for i := 1; i <= maxWt; i++ {
		assert.True(cache.Exists(i), "%d", i)
	}
	assert.Equal(maxWt, cache.Weight())

	// One more element causes an eviction
	// head> 101 16 15 14 13 12 11 10 9 8 7 6 5 4 3 2 <tail
	cache.Store(101, "Light", Weight(1))
	assert.False(cache.Exists(1), "%d", 1)
	for i := 2; i <= maxWt; i++ {
		assert.True(cache.Exists(i), "%d", i)
	}
	assert.True(cache.Exists(101), "%d", 101)
	assert.Equal(maxWt, cache.Weight())

	// Heavy element will cause eviction of 2 elements
	// head> 103! 101 16 15 14 13 12 11 10 9 8 7 6 5 4 <tail
	cache.Store(103, "Heavy", Weight(2))
	for i := 1; i < 3; i++ {
		assert.False(cache.Exists(i), "%d", i)
	}
	for i := 4; i <= maxWt; i++ {
		assert.True(cache.Exists(i), "%d", i)
	}
	assert.True(cache.Exists(101), "%d", 101)
	assert.True(cache.Exists(103), "%d", 103)
	assert.Equal(maxWt, cache.Weight())

	// Super heavy element will cause eviction of 5 elements
	// head> 104!! 103! 101 16 15 14 13 12 11 10 9 <tail
	cache.Store(104, "Super heavy", Weight(5))
	for i := 1; i < 9; i++ {
		assert.False(cache.Exists(i), "%d", i)
	}
	for i := 9; i <= maxWt; i++ {
		assert.True(cache.Exists(i), "%d", i)
	}
	assert.True(cache.Exists(101), "%d", 101)
	assert.True(cache.Exists(103), "%d", 103)
	assert.True(cache.Exists(104), "%d", 104)
	assert.Equal(maxWt, cache.Weight())

	assert.True(integrity(cache))
}

func TestLRU_ChangeMaxWeight(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	maxWt := 16
	cache := New[int, string](maxWt, time.Hour)

	for i := 1; i <= maxWt; i++ {
		cache.Store(i, "1", Weight(1))
	}
	assert.Equal(maxWt, cache.Weight())

	// Halve the weight limit
	cache.SetMaxWeight(maxWt / 2)

	assert.Equal(maxWt/2, cache.Weight())

	assert.True(integrity(cache))
}

func TestLRU_Clear(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Hour)
	assert.Zero(cache.Len())
	assert.Zero(cache.Weight())

	n := 6
	sum := 0
	for i := 1; i <= n; i++ {
		cache.Store(i, "X", Weight(i))
		sum += i
	}
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		assert.True(ok)
		assert.Equal("X", v)
	}
	assert.Equal(n, cache.Len())
	assert.Equal(sum, cache.Weight())

	cache.Clear()
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		assert.False(ok)
		assert.Equal("", v)
	}
	assert.Zero(cache.Len())
	assert.Zero(cache.Weight())

	assert.True(integrity(cache))
}

func TestLRU_Delete(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	span := 10
	sim := map[int]string{}
	cache := New[int, string](1024, time.Hour)
	for range 2048 {
		n := rand.IntN(span * 2)
		if n >= span {
			delete(sim, n-span)
			cache.Delete(n - span)
			assert.False(cache.Exists(n - span))
		} else {
			sim[n] = "X"
			cache.Store(n, "X")
			assert.True(cache.Exists(n))
		}
	}

	for i := range span {
		v, _ := cache.Load(i)
		assert.Equal(sim[i], v)
	}

	assert.True(integrity(cache))
}

func TestLRU_DeletePredicate(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Hour)
	for i := 1; i <= 10; i++ {
		cache.Store(i, "X")
	}
	assert.Equal(10, cache.Len())
	cache.DeletePredicate(func(key int) bool {
		return key <= 5
	})
	assert.Equal(5, cache.Len())
	for i := 1; i <= 10; i++ {
		assert.Equal(i > 5, cache.Exists(i))
	}

	assert.True(integrity(cache))
}

func TestLRU_MaxAge(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Second*35)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30 // t=30
	cache.Store(30, "X")
	assert.True(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.Equal(2, cache.Len())

	// Elements older than the max age of the cache should expire
	cache.timeOffset += time.Second * 10 // t=40
	cache.Store(40, "X")
	assert.Equal(2, cache.Len()) // 0 element was trimmed from the tail on store
	assert.False(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.True(cache.Exists(40))
	assert.Equal(2, cache.Len())

	cache.timeOffset += time.Second * 30 // t=70
	assert.False(cache.Exists(30))
	assert.True(cache.Exists(40))
	assert.Equal(1, cache.Len()) // 30 element was trimmed from the tail

	// The load option overrides the cache's default max age
	_, ok := cache.Load(40, MaxAge(29*time.Second))
	assert.False(ok)

	assert.True(integrity(cache))
}

func TestLRU_TrimOldest(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// Weight limit is generous, so diet() never fires: an expired entry that is never read again would linger
	// under the old lazy scheme. maybeTrimOldest reclaims it on unrelated activity.
	cache := New[int, string](1024, time.Second*30)

	cache.Store(1, "X")
	cache.Store(2, "X")
	assert.Equal(2, cache.Len())

	// Advance past the max age of entries 1 and 2, then touch an unrelated key. The tail is reaped incrementally,
	// two per operation, even though the cache is nowhere near its weight limit.
	cache.timeOffset += time.Second * 40
	cache.Store(3, "X") // trims 1 and 2 from the tail (k=2)
	assert.Equal(1, cache.Len())
	assert.False(cache.Exists(1))
	assert.False(cache.Exists(2))
	assert.True(cache.Exists(3))
	assert.True(integrity(cache))
}

func TestLRU_TrimOldestUsesMaxAge(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// maybeTrimOldest measures expiry against the cache-wide max age, not any shorter per-call age, so an entry
	// within the max age is left in place by unrelated activity - only an explicit short-max-age read would evict it.
	cache := New[int, string](1024, time.Minute)

	cache.Store(1, "X")
	cache.timeOffset += time.Second * 40 // past a 30s age, but within the 60s cache max

	// Unrelated activity does not trim entry 1: it is still within the cache max age.
	cache.Store(2, "X")
	assert.Equal(2, cache.Len())
	assert.True(cache.Exists(1))

	// Once past the cache-wide max age, the same unrelated activity trims it from the tail.
	cache.timeOffset += time.Second * 30 // t=70, past the 60s max
	cache.Store(3, "X")
	assert.False(cache.Exists(1))
	assert.True(cache.Exists(2))
	assert.True(cache.Exists(3))
	assert.True(integrity(cache))
}

func TestLRU_BumpMaxAge(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Second*30)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 20
	_, ok := cache.Load(0, Bump(true))
	assert.True(ok)
	cache.timeOffset += time.Second * 20
	_, ok = cache.Load(0, Bump(true))
	assert.True(ok)
}

func TestLRU_ReduceMaxAge(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Minute)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30
	cache.Store(30, "X")
	cache.timeOffset += time.Second * 20
	cache.Store(60, "X")
	assert.True(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.True(cache.Exists(60))
	assert.Equal(3, cache.Len())

	// Halve the age limit
	cache.SetMaxAge(30 * time.Second)

	assert.False(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.True(cache.Exists(60))
	assert.Equal(2, cache.Len()) // 0 element was evicted on failed load

	assert.True(integrity(cache))
}

func TestLRU_IncreaseMaxAge(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Minute)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30
	cache.Store(30, "X")
	cache.timeOffset += time.Second * 20
	cache.Store(60, "X")
	assert.True(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.True(cache.Exists(60))
	assert.Equal(3, cache.Len())

	// Double the age limit
	cache.SetMaxAge(time.Minute * 2)
	cache.timeOffset += time.Second * 30
	cache.Store(90, "X")

	assert.True(cache.Exists(0))
	assert.True(cache.Exists(30))
	assert.True(cache.Exists(60))
	assert.True(cache.Exists(90))
	assert.Equal(4, cache.Len())

	assert.True(integrity(cache))
}

func TestLRU_Bump(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](8, time.Hour)

	// Fill in the cache
	// head> 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= 8; i++ {
		cache.Store(i, "X")
	}
	assert.Equal(8, cache.Len())

	// Loading element 2 should bump it to the head of the cache
	// head> 2 8 7 6 5 4 3 1 <tail
	_, ok := cache.Load(2)
	assert.True(ok)
	assert.Equal(8, cache.Len())
	assert.True(cache.Exists(1))

	// Storing element 9 should evict 1
	// head> 9 2 8 7 6 5 4 3 <tail
	cache.Store(9, "X")
	assert.Equal(8, cache.Len())
	assert.False(cache.Exists(1))

	// Storing element 10 evicts 3
	// head> 10 9 2 8 7 6 5 4 <tail
	cache.Store(10, "X")
	assert.Equal(8, cache.Len())
	assert.False(cache.Exists(1))
	assert.False(cache.Exists(3))
	assert.True(cache.Exists(4))

	// Load element 4 without bumping it to the head of the queue
	// Storing element 11 evicts 4
	// head> 11 10 9 2 8 7 6 5 <tail
	cache.Load(4, NoBump())
	cache.Store(11, "X")
	assert.Equal(8, cache.Len())
	assert.False(cache.Exists(4))
	assert.True(cache.Exists(5))

	assert.True(integrity(cache))
}

func TestLRU_RandomCohesion(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	cache := New[int, string](1024, time.Hour)

	for step := 0; step < 100000; step++ {
		key := rand.IntN(8)
		wt := rand.IntN(4) + 1
		maxAge := time.Duration(rand.IntN(30)) * time.Second
		bump := rand.IntN(1) == 0
		op := rand.IntN(7)
		switch op {
		case 0, 1, 2:
			cache.Store(key, "X", Weight(wt))
		case 3, 4:
			cache.Load(key, MaxAge(maxAge), Bump(bump))
		case 5:
			cache.LoadOrStore(key, "Y", Weight(wt), MaxAge(maxAge), Bump(bump))
		case 6:
			cache.Delete(key)
		}
	}

	assert.True(integrity(cache))
}

func BenchmarkLRU_Store(b *testing.B) {
	cache := New[int, int](b.N*2, time.Hour)
	for i := range b.N {
		cache.Store(i, i)
	}

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/lru
	// cpu: Apple M1 Pro
	// BenchmarkLRU_Store-10    	 3914073	       327.8 ns/op	     164 B/op	       2 allocs/op
}

func BenchmarkLRU_LoadNoBump(b *testing.B) {
	cache := New[int, int](b.N*2, time.Hour)
	for i := range b.N {
		cache.Store(i, i)
	}
	b.ResetTimer()
	for i := range b.N {
		cache.Load(i, NoBump())
	}

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/lru
	// cpu: Apple M1 Pro
	// BenchmarkLRU_LoadNoBump-10    	 5143230	       287.4 ns/op	      24 B/op	       1 allocs/op
}

func BenchmarkLRU_LoadBump(b *testing.B) {
	cache := New[int, int](b.N*2, time.Hour)
	for i := range b.N {
		cache.Store(i, i)
	}
	b.ResetTimer()
	for i := range b.N {
		cache.Load(i)
	}

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/lru
	// cpu: Apple M1 Pro
	// BenchmarkLRU_LoadBump-10    	 6879088	       283.3 ns/op	      24 B/op	       1 allocs/op
}

// BenchmarkLRU_LoadBumpParallel measures the contended bump path: every goroutine takes the one exclusive mutex to
// reorder the LRU list. This is the single-mutex ceiling the review flagged.
func BenchmarkLRU_LoadBumpParallel(b *testing.B) {
	const workingSet = 4096 // power of two for the mask below
	cache := New[int, int](workingSet*2, time.Hour)
	for i := range workingSet {
		cache.Store(i, i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Load(i & (workingSet - 1))
			i++
		}
	})

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/lru
	// cpu: Apple M1 Pro
	// BenchmarkLRU_LoadBumpParallel-10    	 4514941	       250.8 ns/op	      24 B/op	       1 allocs/op
	// ~4M bump-loads/sec aggregate under 10-way contention, on par with the serial rate: the single mutex is not the
	// bottleneck, and it sustains millions of reads/sec - orders of magnitude above the rate a roundtrip-saving
	// cache sees.
}

// BenchmarkLRU_LoadNoBumpParallel measures the contended read-only path. It still serializes on the same mutex
// today (Load always locks), which is why an RWMutex only helps if most reads are made bump-free first.
func BenchmarkLRU_LoadNoBumpParallel(b *testing.B) {
	const workingSet = 4096 // power of two for the mask below
	cache := New[int, int](workingSet*2, time.Hour)
	for i := range workingSet {
		cache.Store(i, i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Load(i&(workingSet-1), NoBump())
			i++
		}
	})

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/lru
	// cpu: Apple M1 Pro
	// BenchmarkLRU_LoadNoBumpParallel-10    	 5365264	       230.9 ns/op	      24 B/op	       1 allocs/op
	// ~4.3M read-only-loads/sec aggregate under 10-way contention: faster than the bump path (no list surgery) but
	// still serialized on the same mutex, so a bump-free read gains concurrency only once the lock itself is relaxed.
}

// integrity checks the internal structure of the cache.
func integrity[K comparable, V any](c *Cache[K, V]) bool {
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
