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
	"testing"
	"time"

	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestLRU_Load(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Store("a", "aaa")
	cache.Store("b", "bbb")
	cache.Store("c", "ccc")
	testarossa.True(t, cache.cohesion())

	v, ok := cache.Load("a")
	testarossa.True(t, ok)
	testarossa.Equal(t, "aaa", v)

	v, ok = cache.Load("b")
	testarossa.True(t, ok)
	testarossa.Equal(t, "bbb", v)

	v, ok = cache.Load("c")
	testarossa.True(t, ok)
	testarossa.Equal(t, "ccc", v)

	v, ok = cache.Load("d")
	testarossa.False(t, ok)
	testarossa.Equal(t, "", v)

	m := cache.ToMap()
	testarossa.NotEqual(t, 0, len(m["a"]))
	testarossa.NotEqual(t, 0, len(m["b"]))
	testarossa.NotEqual(t, 0, len(m["c"]))
	testarossa.Zero(t, len(m["d"]))

	testarossa.True(t, cache.cohesion())
}

func TestLRU_LoadOrStore(t *testing.T) {
	t.Parallel()

	cache := NewCache[string, string]()
	cache.Store("a", "aaa")

	v, found := cache.LoadOrStore("a", "AAA")
	testarossa.True(t, found)
	testarossa.Equal(t, "aaa", v)

	cache.Delete("a")

	v, found = cache.LoadOrStore("a", "AAA")
	testarossa.False(t, found)
	testarossa.Equal(t, "AAA", v)

	v, found = cache.Load("a")
	testarossa.True(t, found)
	testarossa.Equal(t, "AAA", v)

	testarossa.True(t, cache.cohesion())
}

func TestLRU_MaxWeight(t *testing.T) {
	t.Parallel()

	maxWt := 16
	cache := NewCache[int, string]()
	cache.SetMaxWeight(maxWt)

	cache.Store(999, "Too Big", Weight(maxWt+1))
	_, ok := cache.Load(999)
	testarossa.False(t, ok)
	testarossa.Zero(t, cache.Weight())

	// Fill in the cache
	// head> 16 15 14 13 12 11 10 9 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= maxWt; i++ {
		cache.Store(i, "Light", Weight(1))
	}
	for i := 1; i <= maxWt; i++ {
		testarossa.True(t, cache.Exists(i), "%d", i)
	}
	testarossa.Equal(t, maxWt, cache.Weight())

	// One more element causes an eviction
	// head> 101 16 15 14 13 12 11 10 9 8 7 6 5 4 3 2 <tail
	cache.Store(101, "Light", Weight(1))
	testarossa.False(t, cache.Exists(1), "%d", 1)
	for i := 2; i <= maxWt; i++ {
		testarossa.True(t, cache.Exists(i), "%d", i)
	}
	testarossa.True(t, cache.Exists(101), "%d", 101)
	testarossa.Equal(t, maxWt, cache.Weight())

	// Heavy element will cause eviction of 2 elements
	// head> 103! 101 16 15 14 13 12 11 10 9 8 7 6 5 4 <tail
	cache.Store(103, "Heavy", Weight(2))
	for i := 1; i < 3; i++ {
		testarossa.False(t, cache.Exists(i), "%d", i)
	}
	for i := 4; i <= maxWt; i++ {
		testarossa.True(t, cache.Exists(i), "%d", i)
	}
	testarossa.True(t, cache.Exists(101), "%d", 101)
	testarossa.True(t, cache.Exists(103), "%d", 103)
	testarossa.Equal(t, maxWt, cache.Weight())

	// Super heavy element will cause eviction of 5 elements
	// head> 104!! 103! 101 16 15 14 13 12 11 10 9 <tail
	cache.Store(104, "Super heavy", Weight(5))
	for i := 1; i < 9; i++ {
		testarossa.False(t, cache.Exists(i), "%d", i)
	}
	for i := 9; i <= maxWt; i++ {
		testarossa.True(t, cache.Exists(i), "%d", i)
	}
	testarossa.True(t, cache.Exists(101), "%d", 101)
	testarossa.True(t, cache.Exists(103), "%d", 103)
	testarossa.True(t, cache.Exists(104), "%d", 104)
	testarossa.Equal(t, maxWt, cache.Weight())

	testarossa.True(t, cache.cohesion())
}

func TestLRU_ChangeMaxWeight(t *testing.T) {
	t.Parallel()

	maxWt := 16
	cache := NewCache[int, string]()
	cache.SetMaxWeight(maxWt)

	for i := 1; i <= maxWt; i++ {
		cache.Store(i, "1", Weight(1))
	}
	testarossa.Equal(t, maxWt, cache.Weight())

	// Halve the weight limit
	cache.SetMaxWeight(maxWt / 2)

	testarossa.Equal(t, maxWt/2, cache.Weight())

	testarossa.True(t, cache.cohesion())
}

func TestLRU_Clear(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	testarossa.Zero(t, cache.Len())
	testarossa.Zero(t, cache.Weight())

	n := 6
	sum := 0
	for i := 1; i <= n; i++ {
		cache.Store(i, "X", Weight(i))
		sum += i
	}
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		testarossa.True(t, ok)
		testarossa.Equal(t, "X", v)
	}
	testarossa.Equal(t, n, cache.Len())
	testarossa.Equal(t, sum, cache.Weight())

	cache.Clear()
	for i := 1; i <= n; i++ {
		v, ok := cache.Load(i)
		testarossa.False(t, ok)
		testarossa.Equal(t, "", v)
	}
	testarossa.Zero(t, cache.Len())
	testarossa.Zero(t, cache.Weight())

	testarossa.True(t, cache.cohesion())
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
			testarossa.False(t, cache.Exists(n-span))
		} else {
			sim[n] = "X"
			cache.Store(n, "X")
			testarossa.True(t, cache.Exists(n))
		}
	}

	for i := 0; i < span; i++ {
		v, _ := cache.Load(i)
		testarossa.Equal(t, sim[i], v)
	}

	testarossa.True(t, cache.cohesion())
}

func TestLRU_DeletePredicate(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	for i := 1; i <= 10; i++ {
		cache.Store(i, "X")
	}
	testarossa.Equal(t, 10, cache.Len())
	cache.DeletePredicate(func(key int) bool {
		return key <= 5
	})
	testarossa.Equal(t, 5, cache.Len())
	for i := 1; i <= 10; i++ {
		testarossa.Equal(t, i > 5, cache.Exists(i))
	}

	testarossa.True(t, cache.cohesion())
}

func TestLRU_MaxAge(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxAge(time.Second * 35)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30 // t=30
	cache.Store(30, "X")
	testarossa.True(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.Equal(t, 2, cache.Len())

	// Elements older than the max age of the cache should expire
	cache.timeOffset += time.Second * 10 // t=40
	cache.Store(40, "X")
	testarossa.Equal(t, 3, cache.Len()) // 0 element is still cached
	testarossa.False(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(40))
	testarossa.Equal(t, 2, cache.Len()) // 0 element was evicted on failed load

	cache.timeOffset += time.Second * 30 // t=70
	testarossa.False(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(40))
	testarossa.Equal(t, 1, cache.Len()) // 30 element was evicted on failed load

	// The load option overrides the cache's default max age
	_, ok := cache.Load(40, MaxAge(29*time.Second))
	testarossa.False(t, ok)

	testarossa.True(t, cache.cohesion())
}

func TestLRU_BumpMaxAge(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxAge(time.Second * 30)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 20
	_, ok := cache.Load(0, Bump(true))
	testarossa.True(t, ok)
	cache.timeOffset += time.Second * 20
	_, ok = cache.Load(0, Bump(true))
	testarossa.True(t, ok)
}

func TestLRU_ReduceMaxAge(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxAge(time.Minute)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30
	cache.Store(30, "X")
	cache.timeOffset += time.Second * 20
	cache.Store(60, "X")
	testarossa.True(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(60))
	testarossa.Equal(t, 3, cache.Len())

	// Halve the age limit
	cache.SetMaxAge(30 * time.Second)

	testarossa.False(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(60))
	testarossa.Equal(t, 2, cache.Len()) // 0 element was evicted on failed load

	testarossa.True(t, cache.cohesion())
}

func TestLRU_IncreaseMaxAge(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxAge(time.Minute)

	cache.Store(0, "X")
	cache.timeOffset += time.Second * 30
	cache.Store(30, "X")
	cache.timeOffset += time.Second * 20
	cache.Store(60, "X")
	testarossa.True(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(60))
	testarossa.Equal(t, 3, cache.Len())

	// Double the age limit
	cache.SetMaxAge(time.Minute * 2)
	cache.timeOffset += time.Second * 30
	cache.Store(90, "X")

	testarossa.True(t, cache.Exists(0))
	testarossa.True(t, cache.Exists(30))
	testarossa.True(t, cache.Exists(60))
	testarossa.True(t, cache.Exists(90))
	testarossa.Equal(t, 4, cache.Len())

	testarossa.True(t, cache.cohesion())
}

func TestLRU_Bump(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()
	cache.SetMaxWeight(8)

	// Fill in the cache
	// head> 8 7 6 5 4 3 2 1 <tail
	for i := 1; i <= 8; i++ {
		cache.Store(i, "X")
	}
	testarossa.Equal(t, 8, cache.Len())

	// Loading element 2 should bump it to the head of the cache
	// head> 2 8 7 6 5 4 3 1 <tail
	_, ok := cache.Load(2)
	testarossa.True(t, ok)
	testarossa.Equal(t, 8, cache.Len())
	testarossa.True(t, cache.Exists(1))

	// Storing element 9 should evict 1
	// head> 9 2 8 7 6 5 4 3 <tail
	cache.Store(9, "X")
	testarossa.Equal(t, 8, cache.Len())
	testarossa.False(t, cache.Exists(1))

	// Storing element 10 evicts 3
	// head> 10 9 2 8 7 6 5 4 <tail
	cache.Store(10, "X")
	testarossa.Equal(t, 8, cache.Len())
	testarossa.False(t, cache.Exists(1))
	testarossa.False(t, cache.Exists(3))
	testarossa.True(t, cache.Exists(4))

	// Load element 4 without bumping it to the head of the queue
	// Storing element 11 evicts 4
	// head> 11 10 9 2 8 7 6 5 <tail
	cache.Load(4, NoBump())
	cache.Store(11, "X")
	testarossa.Equal(t, 8, cache.Len())
	testarossa.False(t, cache.Exists(4))
	testarossa.True(t, cache.Exists(5))

	testarossa.True(t, cache.cohesion())
}

func TestLRU_RandomCohesion(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string]()

	for step := 0; step < 100000; step++ {
		key := rand.Intn(8)
		wt := rand.Intn(4) + 1
		maxAge := time.Duration(rand.Intn(30)) * time.Second
		bump := rand.Intn(1) == 0
		op := rand.Intn(7)
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

	testarossa.True(t, cache.cohesion())
}

func BenchmarkLRU_Store(b *testing.B) {
	cache := NewCache[int, int]()
	cache.SetMaxWeight(b.N * 2)
	for i := 0; i < b.N; i++ {
		cache.Store(i, i)
	}

	// On 2021 MacBook Pro M1 16":
	// 288 ns/op
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
	// 193 ns/op
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
	// 190 ns/op
}
