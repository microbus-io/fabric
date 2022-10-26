package dlru

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/utils"
	"github.com/stretchr/testify/assert"
)

func TestDLRU_Lookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("lookup.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)
	defer alphaLRU.Close(ctx)

	beta := connector.New("lookup.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)
	defer betaLRU.Close(ctx)

	gamma := connector.New("lookup.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU, err := NewCache(ctx, gamma, "/cache")
	assert.NoError(t, err)
	defer gammaLRU.Close(ctx)

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
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)

	// Store values in alpha before starting beta and gamma
	n := 2048
	for i := 0; i < n; i++ {
		alphaLRU.localCache.Store(strconv.Itoa(i), []byte(strconv.Itoa(i)))
	}
	assert.Equal(t, n, alphaLRU.localCache.Len())

	beta := connector.New("rescue.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)
	defer betaLRU.Close(ctx)

	gamma := connector.New("rescue.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU, err := NewCache(ctx, gamma, "/cache")
	assert.NoError(t, err)
	defer gammaLRU.Close(ctx)

	assert.Zero(t, betaLRU.localCache.Len())
	assert.Zero(t, gammaLRU.localCache.Len())

	// Should distribute the elements to beta and gamma
	alphaLRU.Close(ctx)
	assert.Equal(t, n, betaLRU.localCache.Len()+gammaLRU.localCache.Len())

	var wg sync.WaitGroup
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
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	alphaLRU.SetMaxMemory(maxMem)
	assert.NoError(t, err)
	defer alphaLRU.Close(ctx)

	beta := connector.New("weight.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	betaLRU, err := NewCache(ctx, beta, "/cache")
	betaLRU.SetMaxMemory(maxMem)
	assert.NoError(t, err)
	defer betaLRU.Close(ctx)

	gamma := connector.New("weight.dlru")
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()
	gammaLRU, err := NewCache(ctx, gamma, "/cache")
	gammaLRU.SetMaxMemory(maxMem)
	assert.NoError(t, err)
	defer gammaLRU.Close(ctx)

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

	dlru, err := NewCache(context.Background(), connector.New("www.example.com"), "/path")
	dlru.SetMaxAge(5 * time.Hour)
	dlru.SetMaxMemoryMB(8)
	assert.NoError(t, err)

	assert.Equal(t, "https://www.example.com:443/path", dlru.basePath)
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
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)
	defer alphaLRU.Close(ctx)

	beta := connector.New("multicast.optim.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)
	defer betaLRU.Close(ctx)

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

	cache, err := NewCache(ctx, con, "/cache")
	assert.NoError(t, err)
	defer cache.Close(ctx)

	// Missing key
	for _, action := range []string{"load", "checksum", "store", "delete"} {
		w := utils.NewResponseRecorder()
		r, err := http.NewRequest("GET", "https://"+con.HostName()+"/cache/all?do="+action, nil)
		assert.NoError(t, err)
		frame.Of(r).SetFromHost(con.HostName())
		err = cache.handleAll(w, r)
		assert.Equal(t, "missing key", err.Error())
	}

	_, _, err = cache.Load(ctx, "")
	assert.Equal(t, "missing key", err.Error())
	_, err = cache.LoadJSON(ctx, "", nil)
	assert.Equal(t, "missing key", err.Error())
	err = cache.Store(ctx, "", nil)
	assert.Equal(t, "missing key", err.Error())
	err = cache.Delete(ctx, "")
	assert.Equal(t, "missing key", err.Error())

	// Invalid action
	w := utils.NewResponseRecorder()
	r, err := http.NewRequest("GET", "https://"+con.HostName()+"/cache/all?do=unknown", nil)
	assert.NoError(t, err)
	frame.Of(r).SetFromHost(con.HostName())
	err = cache.handleAll(w, r)
	assert.Contains(t, err.Error(), "invalid action")

	// Foreign host
	w = utils.NewResponseRecorder()
	r, err = http.NewRequest("GET", "https://"+con.HostName()+"/cache/all?do=load", nil)
	assert.NoError(t, err)
	frame.Of(r).SetFromHost("foreign.host")
	err = cache.handleAll(w, r)
	assert.Contains(t, err.Error(), "foreign host")
}

func TestDLRU_Inconsistency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	alpha := connector.New("inconsistency.dlru")
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	alphaLRU, err := NewCache(ctx, alpha, "/cache")
	assert.NoError(t, err)
	defer alphaLRU.Close(ctx)

	beta := connector.New("inconsistency.dlru")
	err = beta.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	betaLRU, err := NewCache(ctx, beta, "/cache")
	assert.NoError(t, err)
	defer betaLRU.Close(ctx)

	// Store an element in the cache
	err = alphaLRU.Store(ctx, "Foo", []byte("Bar"))
	assert.NoError(t, err)

	// Should be copied to both internal caches
	alphaVal, ok := alphaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(alphaVal))
	betaVal, ok := betaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(betaVal))

	// Should be loadable from either caches
	val, ok, err := alphaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(val))
	val, ok, err = betaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(val))

	// Corrupt the value in one of the internal caches
	betaLRU.localCache.Store("Foo", []byte("Bad"))

	// Loading should fail from either caches
	_, ok, err = alphaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = betaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	// The values are still in the internal caches
	alphaVal, ok = alphaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Bar", string(alphaVal))
	betaVal, ok = betaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Bad", string(betaVal))

	// Storing a new value should remedy the situation
	err = alphaLRU.Store(ctx, "Foo", []byte("Baz"))
	assert.NoError(t, err)

	// Should be copied to both internal caches
	alphaVal, ok = alphaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Baz", string(alphaVal))
	betaVal, ok = betaLRU.localCache.Load("Foo")
	assert.True(t, ok)
	assert.Equal(t, "Baz", string(betaVal))

	// Should be loadable from either caches
	val, ok, err = alphaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Baz", string(val))
	val, ok, err = betaLRU.Load(ctx, "Foo")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Baz", string(val))
}
