/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"testing"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestLRU_Load(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Store("a", "aaa")
	cache.Store("b", "bbb")
	cache.Store("c", "ccc")

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

	m := cache.ToMap()
	assert.NotEmpty(t, m["a"])
	assert.NotEmpty(t, m["b"])
	assert.NotEmpty(t, m["c"])
	assert.Empty(t, m["d"])
}

func TestLRU_LoadOrStore(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Store("a", "aaa")

	v, found := cache.LoadOrStore("a", "AAA")
	assert.True(t, found)
	assert.Equal(t, "aaa", v)

	cache.Delete("a")

	v, found = cache.LoadOrStore("a", "AAA")
	assert.False(t, found)
	assert.Equal(t, "AAA", v)

	v, found = cache.Load("a")
	assert.True(t, found)
	assert.Equal(t, "AAA", v)
}

func TestLRU_MaxWeight(t *testing.T) {
	t.Parallel()

	maxWt := 2 * numBuckets
	cache := NewCache[int, string]()
	cache.SetMaxWeight(maxWt)

	cache.Store(999, "Too Big", Weight(maxWt+1))
	_, ok := cache.Load(999)
	assert.False(t, ok)
	assert.Equal(t, 0, cache.Weight())

	// Fill in the cache
	// head> [16,15] [14,13] [12,11] [10,9] [8,7] [6,5] [4,3] [2,1] <tail
	for i := 1; i <= maxWt; i++ {
		cache.Store(i, "Light", Weight(1))
	}
	for i := 1; i <= maxWt; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, maxWt, cache.Weight())

	// One more element causes an eviction
	// head> 101 [16,15] [14,13] [12,11] [10,9] [8,7] [6,5] [4,3] <tail
	cache.Store(101, "Light", Weight(1))
	for i := 1; i < 2; i++ {
		assert.False(t, cache.Exists(i), "%d", i)
	}
	for i := 3; i <= maxWt; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, maxWt-1, cache.Weight())

	// Heavy element will cause eviction of [4,3]
	// head> [101,103!] [16,15] [14,13] [12,11] [10,9] [8,7] [6,5] _ <tail
	cache.Store(103, "Heavy", Weight(2))
	for i := 1; i < 4; i++ {
		assert.False(t, cache.Exists(i), "%d", i)
	}
	for i := 5; i <= maxWt; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, maxWt-1, cache.Weight())

	// Super heavy element will cause eviction of [8,7] [6,5]
	// head> 104!! [101,103!] [16,15] [14,13] [12,11] [10,9] _ _ <tail
	cache.Store(104, "Super heavy", Weight(5))
	for i := 1; i < 9; i++ {
		assert.False(t, cache.Exists(i), "%d", i)
	}
	for i := 9; i <= maxWt; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, maxWt, cache.Weight())
}

func TestLRU_ChangeMaxWeight(t *testing.T) {
	t.Parallel()

	maxWt := 2 * numBuckets
	cache := NewCache[int, string]()
	cache.SetMaxWeight(maxWt)

	for i := 0; i < numBuckets*2; i++ {
		cache.Store(i, "1", Weight(1))
	}
	assert.Equal(t, maxWt, cache.Weight())

	// Halve the weight limit
	cache.SetMaxWeight(maxWt / 2)

	assert.Equal(t, maxWt/2, cache.Weight())
}

func TestLRU_Clear(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	assert.Equal(t, 0, cache.Len())
	assert.Equal(t, 0, cache.Weight())

	n := 6
	sum := 0
	for i := 1; i <= n; i++ {
		cache.Store(i, "X", Weight(i))
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
			assert.False(t, cache.Exists(n-span))
		} else {
			sim[n] = "X"
			cache.Store(n, "X")
			assert.True(t, cache.Exists(n))
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
	cache := NewCache[int, string]()
	cache.SetMaxAge(time.Second * time.Duration(seconds))
	clock := clock.NewMock()
	cache.setClock(clock)

	for i := 1; i <= seconds-1; i++ {
		cache.Store(i, "X")
		clock.Add(time.Second)
	}

	for i := 1; i <= seconds-1; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, seconds-1, cache.Len())

	// The 80th second will cause the oldest bucket to be evicted
	clock.Add(time.Second)
	for i := 11; i <= seconds-1; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, seconds-1-10, cache.Len())

	// Another 10 seconds will remove the next bucket
	clock.Add(10 * time.Second)
	for i := 21; i <= seconds-1; i++ {
		assert.True(t, cache.Exists(i), "%d", i)
	}
	assert.Equal(t, seconds-1-20, cache.Len())

	// Another 9 seconds should not cause an eviction
	clock.Add(9 * time.Second)
	assert.Equal(t, seconds-1-20, cache.Len())

	// Fast forward should clear the entire cache
	clock.Add(time.Second * time.Duration(seconds))
	assert.Equal(t, 0, cache.Len())
}

func TestLRU_ChangeMaxAge(t *testing.T) {
	t.Parallel()

	seconds := numBuckets * 10 // 80 seconds
	maxAge := time.Second * time.Duration(seconds)
	cache := NewCache[int, string]()
	cache.SetMaxAge(maxAge)
	clock := clock.NewMock()
	cache.setClock(clock)

	for i := 1; i <= seconds-1; i++ {
		cache.Store(i, "X")
		clock.Add(time.Second)
	}
	assert.Equal(t, seconds-1, cache.Len())

	// Halve the age limit
	cache.SetMaxAge(maxAge / 2)

	// Cache should empty within half the time
	clock.Add(maxAge / 4)
	assert.Equal(t, seconds/2-1, cache.Len())
	clock.Add(maxAge / 4)
	assert.Equal(t, 0, cache.Len())
}

func TestLRU_Bump(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxWeight(numBuckets)

	// Fill in the cache
	// head> 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= numBuckets; i++ {
		cache.Store(i, "X")
	}
	assert.Equal(t, numBuckets, cache.Len())

	// Loading element 2 should bump it to the head of the cache
	// head> 2 8 7 6 5 4 3 _ <tail
	_, ok := cache.Load(2)
	assert.True(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())
	_, ok = cache.Load(1)
	assert.False(t, ok)

	// Storing element 9 should fit
	// head> 9 2 8 7 6 5 4 3 <tail
	cache.Store(9, "X")
	assert.Equal(t, numBuckets, cache.Len())

	// Storing element 10 evicts 3
	// head> 10 9 2 8 7 6 5 4 <tail
	cache.Store(10, "X")
	assert.Equal(t, numBuckets, cache.Len())
	_, ok = cache.Load(3)
	assert.False(t, ok)

	// Loading element 4 should bump it to the head of the cache
	// head> 4 10 9 2 8 7 6 5 <tail
	_, ok = cache.Load(4)
	assert.True(t, ok)
	assert.Equal(t, numBuckets, cache.Len())

	// Cycle once to cause 5 to drop off the tail
	// head> _ 4 10 9 2 8 7 6 <tail
	cache.cycleOnce()
	_, ok = cache.Load(5)
	assert.False(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())

	// Loading element 4 should bump it to the head of the cache
	// head> 4 _ 10 9 2 8 7 6 <tail
	_, ok = cache.Load(4)
	assert.True(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())

	// Loading element 6 without bumping it to the head of the cache
	// head> 4 _ 10 9 2 8 7 6 <tail
	_, ok = cache.Load(6, NoBump())
	assert.True(t, ok)
	assert.Equal(t, numBuckets-1, cache.Len())

	// Cycle once to cause 6 to drop off the tail
	// head> _ 4 _ 10 9 2 8 7 <tail
	cache.cycleOnce()
	assert.False(t, cache.Exists(6))
	assert.Equal(t, numBuckets-2, cache.Len())
}

func BenchmarkLRU_Store(b *testing.B) {
	cache := NewCache[int, int]()
	cache.SetMaxWeight(b.N * 2)
	for i := 0; i < b.N; i++ {
		cache.Store(i, i)
	}

	// On 2021 MacBook Pro M1 16":
	// 330 ns/op
}

func BenchmarkLRU_LoadNoBump(b *testing.B) {
	cache := NewCache[int, int]()
	cache.SetMaxWeight(b.N * 2)
	for i := 0; i < b.N; i++ {
		cache.Store(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Load(i, NoBump())
	}

	// On 2021 MacBook Pro M1 16":
	// 240 ns/op
}

func BenchmarkLRU_LoadBump(b *testing.B) {
	cache := NewCache[int, int]()
	cache.SetMaxWeight(b.N * 2)
	for i := 0; i < b.N; i++ {
		cache.Store(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Load(i)
	}

	// On 2021 MacBook Pro M1 16":
	// 450 ns/op
}
