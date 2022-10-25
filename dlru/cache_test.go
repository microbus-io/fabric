package dlru

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestDLRU_Lookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("lookup.dlru")
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)

	beta := connector.New("lookup.dlru")
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)

	gamma := connector.New("lookup.dlru")
	gammaLRU, err := NewCache(ctx, gamma, "/cache")
	assert.NoError(t, err)

	err = alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()

	// Insert to alpha cache
	err = alphaLRU.Store(ctx, "A", []byte("AAA"))
	assert.NoError(t, err)
	bbb := struct {
		Num int    `json:"num"`
		Str string `json:"str"`
	}{
		123,
		"abc",
	}
	err = alphaLRU.StoreJSON(ctx, "B", bbb)
	assert.NoError(t, err)
	assert.Equal(t, 2, alphaLRU.localCache.Len())
	assert.Equal(t, 2, betaLRU.localCache.Len())
	assert.Equal(t, 2, gammaLRU.localCache.Len())

	// Should be loadable from all caches
	for _, c := range []*Cache{gammaLRU, betaLRU, alphaLRU} {
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
		assert.Equal(t, bbb, jval)
	}

	// Delete from gamma cache
	err = gammaLRU.Delete(ctx, "A")
	assert.NoError(t, err)
	assert.Equal(t, 1, alphaLRU.localCache.Len())
	assert.Equal(t, 1, betaLRU.localCache.Len())
	assert.Equal(t, 1, gammaLRU.localCache.Len())

	// Should not be loadable from any of the caches
	for _, c := range []*Cache{gammaLRU, betaLRU, alphaLRU} {
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
	assert.Equal(t, 0, alphaLRU.localCache.Len())
	assert.Equal(t, 0, betaLRU.localCache.Len())
	assert.Equal(t, 0, gammaLRU.localCache.Len())

	// Should not be loadable from any of the caches
	for _, c := range []*Cache{gammaLRU, betaLRU, alphaLRU} {
		val, ok, err := c.Load(ctx, "B")
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", string(val))
	}
}

func TestDLRU_Rescue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("rescue.dlru")
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)

	beta := connector.New("rescue.dlru")
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)

	gamma := connector.New("rescue.dlru")
	gammaLRU, err := NewCache(ctx, gamma, "/cache")
	assert.NoError(t, err)

	err = alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	// Store values in alpha before starting beta and gamma
	n := 1024
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			err := alphaLRU.Store(ctx, strconv.Itoa(i), []byte(strconv.Itoa(i)))
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, n, alphaLRU.localCache.Len())

	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()

	assert.Zero(t, betaLRU.localCache.Len())
	assert.Zero(t, gammaLRU.localCache.Len())

	// Should distribute the elements to beta and gamma
	alphaLRU.Close(ctx)
	assert.Equal(t, n, betaLRU.localCache.Len()+gammaLRU.localCache.Len())

	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			val, ok, err := betaLRU.Load(ctx, strconv.Itoa(i))
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, strconv.Itoa(i), string(val))

			val, ok, err = gammaLRU.Load(ctx, strconv.Itoa(i))
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, strconv.Itoa(i), string(val))

			wg.Done()
		}()
	}
	wg.Wait()
}

func TestDLRU_Weight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	maxMem := 4096

	alpha := connector.New("weight.dlru")
	alphaLRU, err := NewCache(ctx, alpha, "/cache", MaxMemory(maxMem))
	assert.NoError(t, err)

	beta := connector.New("weight.dlru")
	betaLRU, err := NewCache(ctx, beta, "/cache", MaxMemory(maxMem))
	assert.NoError(t, err)

	gamma := connector.New("weight.dlru")
	gammaLRU, err := NewCache(ctx, gamma, "/cache", MaxMemory(maxMem))
	assert.NoError(t, err)

	err = alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()

	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()

	// Insert 1/4 of max memory
	payload := rand.AlphaNum64(maxMem / 4)
	err = alphaLRU.Store(ctx, "A", []byte(payload))
	assert.NoError(t, err)

	// Should be replicated to all three caches
	// alpha: A
	// beta: A
	// gamma: A
	assert.Equal(t, 1, alphaLRU.localCache.Len())
	assert.Equal(t, 1, betaLRU.localCache.Len())
	assert.Equal(t, 1, gammaLRU.localCache.Len())
	assert.Equal(t, maxMem/4, alphaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, betaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, gammaLRU.localCache.Weight())

	// Insert another 1/4
	err = alphaLRU.Store(ctx, "B", []byte(payload))
	assert.NoError(t, err)

	// Should only be added to alpha
	// alpha: A B
	// beta: A
	// gamma: A
	assert.Equal(t, 2, alphaLRU.localCache.Len())
	assert.Equal(t, 1, betaLRU.localCache.Len())
	assert.Equal(t, 1, gammaLRU.localCache.Len())
	assert.Equal(t, maxMem/2, alphaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, betaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, gammaLRU.localCache.Weight())

	// Insert 3 more into alpha
	err = alphaLRU.Store(ctx, "C", []byte(payload))
	assert.NoError(t, err)
	err = alphaLRU.Store(ctx, "D", []byte(payload))
	assert.NoError(t, err)
	err = alphaLRU.Store(ctx, "E", []byte(payload))
	assert.NoError(t, err)

	// Alpha will have A evicted
	// alpha: E D C B
	// beta: A
	// gamma: A
	assert.Equal(t, 4, alphaLRU.localCache.Len())
	assert.Equal(t, 1, betaLRU.localCache.Len())
	assert.Equal(t, 1, gammaLRU.localCache.Len())
	assert.Equal(t, maxMem, alphaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, betaLRU.localCache.Weight())
	assert.Equal(t, maxMem/4, gammaLRU.localCache.Weight())

	for _, k := range []string{"A", "B", "C", "D", "E"} {
		val, ok, err := gammaLRU.Load(ctx, k)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, payload, string(val))
	}
}

func TestDLRU_Options(t *testing.T) {
	t.Parallel()

	dlru, err := NewCache(
		context.Background(),
		connector.New("www.example.com"),
		"/path",
		MaxAge(5*time.Hour),
		MaxMemoryMB(8),
		BumpOnLoad(false),
		StrictLoad(true),
		RescueOnClose(false),
	)
	assert.NoError(t, err)

	assert.Equal(t, 5*time.Hour, dlru.localCache.MaxAge())
	assert.Equal(t, 8*1024*1024, dlru.localCache.MaxWeight())
	assert.False(t, dlru.localCache.IsBumpOnLoad())
	assert.Equal(t, "https://www.example.com:443/path", dlru.basePath)
	assert.True(t, dlru.strictLoad)
	assert.False(t, dlru.rescueOnClose)
}
