package dlru

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
)

/*
Cache is an LRU cache that is distributed among the peers of a microservice.
The cache is tied to the microservice and is typically constructed in the OnStartup
callback of the microservice and destroyed in the OnShutdown.

	con := connector.New("www.example.com")
	var myCache *dlru.Cache
	con.SetOnStartup(func(ctx context.Context) error {
		myCache = dlru.NewCache(ctx, con, ":1234/cache", dlru.MaxMemoryMB(128))
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		myCache.Close(ctx)
	})
*/
type Cache struct {
	localCache *lru.Cache[string, []byte]
	basePath   string
	svc        service.Service
}

// NewCache starts a new cache for the service at a given path.
// It's recommended to use a non-standard port for the path.
func NewCache(ctx context.Context, svc service.Service, path string) (*Cache, error) {
	sub, err := sub.NewSub(svc.HostName(), path, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	c := &Cache{
		basePath: "https://" + strings.TrimSuffix(sub.Canonical(), "/"),
		svc:      svc,
	}
	c.localCache = lru.NewCache[string, []byte]()
	c.localCache.SetMaxAge(time.Hour)
	c.localCache.SetMaxWeight(32 * 1024 * 1024) // 32MB

	err = c.start(ctx)
	if err != nil {
		c.stop(ctx)
		return nil, errors.Trace(err)
	}

	return c, nil
}

// SetMaxAge sets the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (c *Cache) SetMaxAge(ttl time.Duration) error {
	err := c.localCache.SetMaxAge(ttl)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MaxAge returns the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (c *Cache) MaxAge() time.Duration {
	return c.localCache.MaxAge()
}

// SetMaxMemory limits the memory used by the cache.
func (c *Cache) SetMaxMemory(bytes int) error {
	err := c.localCache.SetMaxWeight(bytes)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// SetMaxMemoryMB limits the memory used by the cache.
func (c *Cache) SetMaxMemoryMB(megaBytes int) error {
	err := c.localCache.SetMaxWeight(megaBytes * 1024 * 1024)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MaxAge returns the age limit of elements in this cache.
// Elements that are bumped have their life span reset and will therefore survive longer.
func (c *Cache) MaxMemory() int {
	return c.localCache.MaxWeight()
}

// start subscribed to handle cache events from peers.
func (c *Cache) start(ctx context.Context) error {
	err := c.svc.Subscribe(c.basePath+"/all", c.handleAll, sub.NoQueue())
	if err != nil {
		return errors.Trace(err)
	}
	err = c.svc.Subscribe(c.basePath+"/rescue", c.handleStore, sub.DefaultQueue())
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// start unsubscribes from handling cache events from peers.
func (c *Cache) stop(ctx context.Context) error {
	c.svc.Unsubscribe(c.basePath + "/rescue")
	c.svc.Unsubscribe(c.basePath + "/all")
	return nil
}

func (c *Cache) handleAll(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from other hosts
	if frame.Of(r).FromHost() != c.svc.HostName() {
		return errors.Newf("foreign host '%s'", frame.Of(r).FromHost())
	}
	switch r.URL.Query().Get("do") {
	case "load":
		return c.handleLoad(w, r)
	case "store":
		return c.handleStore(w, r)
	case "delete":
		return c.handleDelete(w, r)
	case "clear":
		return c.handleClear(w, r)
	default:
		return errors.Newf("invalid action '%s'", r.URL.Query().Get("do"))
	}
}

func (c *Cache) handleClear(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		return nil
	}
	c.localCache.Clear()
	return nil
}

func (c *Cache) handleDelete(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		return nil
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	c.localCache.Delete(key)
	return nil
}

// handleLoad handles a broadcast when the primary tries to obtain copies held by its peers.
func (c *Cache) handleLoad(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	data, ok := c.localCache.Load(key)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	_, err := w.Write(data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// handleStore handles a broadcast when a new element is stored by the primary.
func (c *Cache) handleStore(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		return nil
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Trace(err)
	}
	weight := len(data)

	if c.localCache.Weight()+weight <= c.localCache.MaxWeight()/4 {
		// Keep a local copy if the cache is relatively empty
		c.localCache.Store(key, data, lru.Weight(weight))
	} else {
		// Delete the local copy, if present
		c.localCache.Delete(key)
	}
	return nil
}

// Close closed and clears the cache.
func (c *Cache) Close(ctx context.Context) error {
	err := c.stop(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	c.rescue(ctx)
	c.localCache.Clear()
	return nil
}

// rescue distributes the elements in the cache to random peers.
func (c *Cache) rescue(ctx context.Context) {
	// Nothing to rescue
	if c.localCache.Len() == 0 {
		return
	}

	// Count number of peers
	u := fmt.Sprintf("https://%s:888/ping", c.svc.HostName())
	ch := c.svc.Publish(ctx, pub.GET(u))
	peers := 0
	for r := range ch {
		_, err := r.Get()
		if err == nil {
			peers++
		}
	}
	if peers == 0 {
		return
	}

	// Send local cache content to random peers
	subCtx, cancel := context.WithTimeout(ctx, 2*time.Second) // Take no more than 2 seconds
	defer cancel()
	concurrent := 64 * peers // 64 requests per peer in parallel
	type kv struct {
		key string
		val []byte
	}
	rescueQueue := make(chan *kv, concurrent)
	go func() {
		for k, v := range c.localCache.ToMap() {
			rescueQueue <- &kv{k, v}
		}
		close(rescueQueue)
	}()
	var wg sync.WaitGroup
	var rescued int64
	t0 := time.Now()
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			for kv := range rescueQueue {
				u := fmt.Sprintf("%s/rescue?key=%s", c.basePath, kv.key)
				_, err := c.svc.Request(subCtx, pub.Method("PUT"), pub.URL(u), pub.Body(kv.val))
				if err != nil {
					break // Likely context timeout
				}
				atomic.AddInt64(&rescued, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(t0)
	c.svc.LogDebug(ctx, "Rescued cache elements", log.Int64("count", rescued), log.Duration("in", dur))
}

// Store an element in the cache.
func (c *Cache) Store(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return errors.New("missing key")
	}

	c.localCache.Delete(key)

	// Broadcast to peers
	u := fmt.Sprintf("%s/all?do=store&key=%s", c.basePath, key)
	ch := c.svc.Publish(ctx, pub.Method("PUT"), pub.URL(u), pub.Body(value))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}

	c.localCache.Store(key, value, lru.Weight(len(value)))

	return nil
}

// Load an element from the cache, locally or from peers.
// If the element is found, it is bumped to the head of the cache.
func (c *Cache) Load(ctx context.Context, key string, options ...LoadOption) (value []byte, ok bool, err error) {
	if key == "" {
		return nil, false, errors.New("missing key")
	}

	var opts cacheOptions
	for _, opt := range options {
		opt(&opts)
	}

	// Check local cache
	value, ok = c.localCache.Load(key, lru.Bump(opts.Bump))
	if ok {
		if !opts.Consensus {
			return value, true, nil
		}
	}

	// Check with peers
	// No need to pass options to peers
	u := fmt.Sprintf("%s/all?do=load&key=%s", c.basePath, key)
	ch := c.svc.Publish(ctx, pub.GET(u))
	for r := range ch {
		res, err := r.Get()
		if err != nil {
			return nil, false, errors.Trace(err)
		}
		if res.StatusCode != http.StatusOK {
			continue
		}
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, false, errors.Trace(err)
		}
		if value != nil && !bytes.Equal(value, data) {
			// Inconsistency detected
			return nil, false, nil
		}
		value = data
	}

	return value, value != nil, nil
}

// Delete an element from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("missing key")
	}

	c.localCache.Delete(key)

	// Broadcast to all peers
	u := fmt.Sprintf("%s/all?do=delete&key=%s", c.basePath, key)
	ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Clear the cache.
func (c *Cache) Clear(ctx context.Context) error {
	c.localCache.Clear()

	// Broadcast to all peers
	u := fmt.Sprintf("%s/all?do=clear", c.basePath)
	ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// LoadJSON loads an element from the cache and unmarshals it as JSON.
// If the element is found, it is bumped to the head of the cache.
func (c *Cache) LoadJSON(ctx context.Context, key string, value any, options ...LoadOption) (ok bool, err error) {
	if key == "" {
		return false, errors.New("missing key")
	}
	data, ok, err := c.Load(ctx, key, options...)
	if err != nil {
		return false, errors.Trace(err)
	}
	if !ok {
		return false, nil
	}
	err = json.Unmarshal(data, value)
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// StoreJSON marshals the value as JSON and stores it in the cache.
// JSON marshalling is not memory efficient and should be avoided if the cache is
// expected to store a lot of data.
func (c *Cache) StoreJSON(ctx context.Context, key string, value any) error {
	if key == "" {
		return errors.New("missing key")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.Store(ctx, key, data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
