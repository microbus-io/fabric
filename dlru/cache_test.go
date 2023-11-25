/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package dlru_test

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestDLRU_Lookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("lookup.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("lookup.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	gamma := connector.New("lookup.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU := gamma.DistribCache()

	// Insert to alpha cache
	err = alphaLRU.Store(ctx, "A", []byte("AAA"))
	assert.NoError(t, err)
	jsonObject := struct {
		Num int    `json:"num"`
		Str string `json:"str"`
	}{
		123,
		"abc",
	}
	err = alphaLRU.StoreJSON(ctx, "B", jsonObject)
	assert.NoError(t, err)
	err = alphaLRU.StoreCompressedJSON(ctx, "C", jsonObject)
	assert.NoError(t, err)

	assert.Equal(t, 3, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, 0, gammaLRU.LocalCache().Len())

	// Should be loadable from all caches
	for _, c := range []*dlru.Cache{gammaLRU, betaLRU, alphaLRU} {
		val, ok, err := c.Load(ctx, "A")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "AAA", string(val))

		var jval struct {
			Num int    `json:"num"`
			Str string `json:"str"`
		}
		ok, err = c.LoadJSON(ctx, "B", &jval)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, jsonObject, jval)
		ok, err = c.LoadCompressedJSON(ctx, "C", &jval)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, jsonObject, jval)
	}

	// Delete from gamma cache
	err = gammaLRU.Delete(ctx, "A")
	assert.NoError(t, err)

	assert.Equal(t, 2, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, 0, gammaLRU.LocalCache().Len())

	// Should not be loadable from any of the caches
	for _, c := range []*dlru.Cache{gammaLRU, betaLRU, alphaLRU} {
		val, ok, err := c.Load(ctx, "A")
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", string(val))

		val, ok, err = c.Load(ctx, "B")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, `{"num":123,"str":"abc"}`, string(val))
	}

	// Clear the cache via beta
	err = betaLRU.Clear(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 0, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, 0, gammaLRU.LocalCache().Len())

	// Should not be loadable from any of the caches
	for _, c := range []*dlru.Cache{gammaLRU, betaLRU, alphaLRU} {
		val, ok, err := c.Load(ctx, "B")
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", string(val))
	}
}

func TestDLRU_Replicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("replicate.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("replicate.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	gamma := connector.New("replicate.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU := gamma.DistribCache()

	// Insert to alpha cache
	err = alphaLRU.Store(ctx, "A", []byte("AAA"), dlru.Replicate(true))
	assert.NoError(t, err)
	jsonObject := struct {
		Num int    `json:"num"`
		Str string `json:"str"`
	}{
		123,
		"abc",
	}
	err = alphaLRU.StoreJSON(ctx, "B", jsonObject, dlru.Replicate(true))
	assert.NoError(t, err)
	err = alphaLRU.StoreCompressedJSON(ctx, "C", jsonObject, dlru.Replicate(true))
	assert.NoError(t, err)

	assert.Equal(t, 3, alphaLRU.LocalCache().Len())
	assert.Equal(t, 3, betaLRU.LocalCache().Len())
	assert.Equal(t, 3, gammaLRU.LocalCache().Len())

	// Delete from gamma cache
	err = gammaLRU.Delete(ctx, "A")
	assert.NoError(t, err)

	assert.Equal(t, 2, alphaLRU.LocalCache().Len())
	assert.Equal(t, 2, betaLRU.LocalCache().Len())
	assert.Equal(t, 2, gammaLRU.LocalCache().Len())

	// Clear the cache via beta
	err = betaLRU.Clear(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 0, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, 0, gammaLRU.LocalCache().Len())
}

func TestDLRU_Rescue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("rescue.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	alphaLRU := alpha.DistribCache()

	// Store values in alpha before starting beta and gamma
	n := 2048
	numChan := make(chan int, n)
	for i := 0; i < n; i++ {
		numChan <- i
	}
	close(numChan)
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU()*4; i++ {
		wg.Add(1)
		go func() {
			for i := range numChan {
				err := alphaLRU.Store(ctx, strconv.Itoa(i), []byte(strconv.Itoa(i)))
				assert.NoError(t, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, n, alphaLRU.LocalCache().Len())

	beta := connector.New("rescue.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	gamma := connector.New("rescue.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU := gamma.DistribCache()

	assert.Zero(t, betaLRU.LocalCache().Len())
	assert.Zero(t, gammaLRU.LocalCache().Len())

	// Should distribute the elements to beta and gamma
	err = alpha.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, n, betaLRU.LocalCache().Len()+gammaLRU.LocalCache().Len())

	numChan = make(chan int, n)
	for i := 0; i < n; i++ {
		numChan <- i
	}
	close(numChan)
	for i := 0; i < runtime.NumCPU()*4; i++ {
		wg.Add(1)
		go func() {
			for i := range numChan {
				val, ok, err := betaLRU.Load(ctx, strconv.Itoa(i))
				assert.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, strconv.Itoa(i), string(val))

				val, ok, err = gammaLRU.Load(ctx, strconv.Itoa(i))
				assert.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, strconv.Itoa(i), string(val))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestDLRU_MaxMemory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	maxMem := 4096

	alpha := connector.New("max.memory.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()
	alphaLRU.SetMaxMemory(maxMem)

	beta := connector.New("max.memory.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()
	betaLRU.SetMaxMemory(maxMem)

	// Insert enough to max out the memory limit
	payload := rand.AlphaNum64(maxMem / 4)
	err = alphaLRU.Store(ctx, "A", []byte(payload))
	assert.NoError(t, err)
	err = alphaLRU.Store(ctx, "B", []byte(payload))
	assert.NoError(t, err)
	err = alphaLRU.Store(ctx, "C", []byte(payload))
	assert.NoError(t, err)
	err = alphaLRU.Store(ctx, "D", []byte(payload))
	assert.NoError(t, err)

	// Should be stored in alpha
	// alpha: D C B A
	// beta:
	assert.Equal(t, 4, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, maxMem, alphaLRU.LocalCache().Weight())
	assert.Equal(t, 0, betaLRU.LocalCache().Weight())

	// Insert another 1/4
	err = alphaLRU.Store(ctx, "E", []byte(payload))
	assert.NoError(t, err)

	// Alpha will have A evicted
	// alpha: E D C B
	// beta:
	assert.Equal(t, 4, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
	assert.Equal(t, maxMem, alphaLRU.LocalCache().Weight())
	assert.Equal(t, 0, betaLRU.LocalCache().Weight())

	for _, k := range []string{"A", "B", "C", "D", "E"} {
		val, ok, err := betaLRU.Load(ctx, k)
		assert.NoError(t, err)
		assert.Equal(t, k != "A", ok)
		if ok {
			assert.Equal(t, payload, string(val))
		}
	}
}

func TestDLRU_WeightAndLen(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("weight.and.len.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("weight.and.len.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	payload := rand.AlphaNum64(1024)
	err = alphaLRU.Store(ctx, "A", []byte(payload))
	assert.NoError(t, err)

	wt, _ := alphaLRU.Weight(ctx)
	assert.Equal(t, 1024, wt)
	len, _ := alphaLRU.Len(ctx)
	assert.Equal(t, 1, len)

	wt, _ = betaLRU.Weight(ctx)
	assert.Equal(t, 1024, wt)
	len, _ = betaLRU.Len(ctx)
	assert.Equal(t, 1, len)

	err = betaLRU.Store(ctx, "B", []byte(payload))
	assert.NoError(t, err)

	wt, _ = alphaLRU.Weight(ctx)
	assert.Equal(t, 2048, wt)
	len, _ = alphaLRU.Len(ctx)
	assert.Equal(t, 2, len)

	wt, _ = betaLRU.Weight(ctx)
	assert.Equal(t, 2048, wt)
	len, _ = betaLRU.Len(ctx)
	assert.Equal(t, 2, len)
}

func TestDLRU_Options(t *testing.T) {
	t.Parallel()

	dlru, err := dlru.NewCache(context.Background(), connector.New("www.example.com"), "/path")
	dlru.SetMaxAge(5 * time.Hour)
	dlru.SetMaxMemoryMB(8)
	assert.NoError(t, err)

	assert.Equal(t, 5*time.Hour, dlru.MaxAge())
	assert.Equal(t, 8*1024*1024, dlru.MaxMemory())
}

func TestDLRU_MulticastOptim(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("multicast.optim.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("multicast.optim.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// First operation is slow
	t0 := time.Now()
	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
	assert.NoError(t, err)
	dur := time.Since(t0)
	assert.True(t, dur >= connector.AckTimeout)

	// Second operation is fast, even if not the same action
	t0 = time.Now()
	err = alphaLRU.Clear(ctx)
	assert.NoError(t, err)
	dur = time.Since(t0)
	assert.True(t, dur < connector.AckTimeout)
}

func TestDLRU_InvalidRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	con := connector.New("invalid.requests.dlru")
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	cache, err := dlru.NewCache(ctx, con, "/cache")
	assert.NoError(t, err)
	defer cache.Close(ctx)

	_, _, err = cache.Load(ctx, "")
	assert.Equal(t, "missing key", err.Error())
	_, err = cache.LoadJSON(ctx, "", nil)
	assert.Equal(t, "missing key", err.Error())
	err = cache.Store(ctx, "", nil)
	assert.Equal(t, "missing key", err.Error())
	err = cache.StoreJSON(ctx, "", nil)
	assert.Equal(t, "missing key", err.Error())
	err = cache.Delete(ctx, "")
	assert.Equal(t, "missing key", err.Error())
}

func TestDLRU_Inconsistency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("inconsistency.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("inconsistency.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	// Store an element in the cache
	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
	assert.NoError(t, err)

	// Should be stored in alpha
	assert.Equal(t, 1, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())

	// Should be loadable from either caches
	val, ok, err := alphaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(val))
	val, ok, err = betaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(val))

	// Store a different value in beta
	betaLRU.LocalCache().Store("Foo", []byte("Bad"))

	// Loading without the consistency check should succeed and return different results
	val, ok, err = alphaLRU.Load(ctx, "Foo", dlru.ConsistencyCheck(false))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(val))
	val, ok, err = betaLRU.Load(ctx, "Foo", dlru.ConsistencyCheck(false))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bad", string(val))

	// Loading with a consistency check should fail from either caches
	_, ok, err = alphaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = betaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	// The inconsistent values should be removed
	assert.Equal(t, 0, alphaLRU.LocalCache().Len())
	assert.Equal(t, 0, betaLRU.LocalCache().Len())
}

func TestDLRU_MaxAge(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("maxage.actions.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("maxage.actions.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU := beta.DistribCache()

	// Store an element in the cache
	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
	assert.NoError(t, err)

	// Wait a second and load it back
	// Do not bump so that the life of the element is not renewed
	time.Sleep(time.Second)
	cached, ok, err := betaLRU.Load(ctx, "Foo", dlru.NoBump())
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, cached, []byte("Bar"))
	}

	// Use a max age option when loading
	_, ok, err = betaLRU.Load(ctx, "Foo", dlru.MaxAge(time.Millisecond*990))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestDLRU_RandomActions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("random.actions.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	beta := connector.New("random.actions.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	gamma := connector.New("random.actions.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()

	caches := []*dlru.Cache{
		alpha.DistribCache(),
		beta.DistribCache(),
		gamma.DistribCache(),
	}

	state := map[string][]byte{}
	for i := 0; i < 10000; i++ {
		cache := caches[rand.Intn(len(caches))]
		key := strconv.Itoa(rand.Intn(20))
		switch rand.Intn(4) {
		case 1, 2: // Load
			bump := rand.Intn(2) == 1
			val1, ok1, err := cache.Load(ctx, key, dlru.Bump(bump))
			assert.NoError(t, err)
			val2, ok2 := state[key]
			assert.Equal(t, ok2, ok1)
			assert.Equal(t, val2, val1)

		case 3: // Store
			val := []byte(rand.AlphaNum32(15))
			err := cache.Store(ctx, key, val)
			assert.NoError(t, err)
			state[key] = val

		case 4: // Delete
			err := cache.Delete(ctx, key)
			assert.NoError(t, err)
			delete(state, key)
		}
	}
}

func BenchmarkDLRU_Store(b *testing.B) {
	ctx := context.Background()

	alpha := connector.New("benchmark.store.dlru")
	err := alpha.Startup()
	assert.NoError(b, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("benchmark.store.dlru")
	err = beta.Startup()
	assert.NoError(b, err)
	defer beta.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
		assert.NoError(b, err)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 16": 193309 ns/op
}

func BenchmarkDLRU_Load(b *testing.B) {
	ctx := context.Background()

	alpha := connector.New("benchmark.load.dlru")
	err := alpha.Startup()
	assert.NoError(b, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("benchmark.load.dlru")
	err = beta.Startup()
	assert.NoError(b, err)
	defer beta.Shutdown()

	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok, err := alphaLRU.Load(ctx, "Foo")
		assert.NoError(b, err)
		assert.True(b, ok)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 16": 165499 ns/op
}

func BenchmarkDLRU_LoadNoConsistencyCheck(b *testing.B) {
	ctx := context.Background()

	alpha := connector.New("benchmark.load.dlru")
	err := alpha.Startup()
	assert.NoError(b, err)
	defer alpha.Shutdown()
	alphaLRU := alpha.DistribCache()

	beta := connector.New("benchmark.load.dlru")
	err = beta.Startup()
	assert.NoError(b, err)
	defer beta.Shutdown()

	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"), dlru.Replicate(true))
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok, err := alphaLRU.Load(ctx, "Foo", dlru.ConsistencyCheck(false))
		assert.NoError(b, err)
		assert.True(b, ok)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 16": 78 ns/op
}

func TestDLRU_Interface(t *testing.T) {
	t.Parallel()

	c := connector.New("example")
	_ = dlru.Service(c)
}
