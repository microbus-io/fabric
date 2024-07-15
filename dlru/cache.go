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

package dlru

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/microbus-io/fabric/coreservices/control/controlapi"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
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
	var myCache dlru.Cache
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
	svc        Service
	hits       int64
	misses     int64
}

// Service is an interface abstraction of a microservice used by the distributed cache.
// The connector implements this interface.
type Service interface {
	service.PublisherSubscriber
	service.Identifier
	service.Logger
}

// NewCache starts a new cache for the service at a given path.
// For security reasons, it is advised to use a non-public port for the path, such as :444/token-cache .
func NewCache(ctx context.Context, svc Service, path string) (*Cache, error) {
	basePath := httpx.JoinHostAndPath(svc.Hostname(), path)
	basePath = strings.TrimSuffix(basePath, "/")

	c := &Cache{
		basePath: basePath,
		svc:      svc,
	}
	c.localCache = lru.NewCache[string, []byte]()
	c.localCache.SetMaxAge(time.Hour)
	c.localCache.SetMaxWeight(32 * 1024 * 1024) // 32MB

	err := c.start(ctx)
	if err != nil {
		c.stop(ctx)
		return nil, errors.Trace(err)
	}

	return c, nil
}

// LocalCache returns the underlying LRU cache that is backing the cache in this peer.
// Modifying the local cache is unadvisable and may result in inconsistencies.
// Access is provided mainly for testing purposes.
func (c *Cache) LocalCache() *lru.Cache[string, []byte] {
	return c.localCache
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
	err := c.svc.Subscribe("ANY", c.basePath+"/all", c.handleAll, sub.NoQueue())
	if err != nil {
		return errors.Trace(err)
	}
	err = c.svc.Subscribe("ANY", c.basePath+"/rescue", c.handleRescue, sub.DefaultQueue())
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// start unsubscribes from handling cache events from peers.
func (c *Cache) stop(ctx context.Context) error {
	c.svc.Unsubscribe("ANY", c.basePath+"/rescue")
	c.svc.Unsubscribe("ANY", c.basePath+"/all")
	return nil
}

// handleAll handles a broadcast when the primary connects with its peers.
func (c *Cache) handleAll(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from other hosts
	if frame.Of(r).FromHost() != c.svc.Hostname() {
		return errors.Newf("foreign host '%s'", frame.Of(r).FromHost())
	}
	switch r.URL.Query().Get("do") {
	case "load":
		return c.handleLoad(w, r)
	case "store":
		return c.handleStore(w, r)
	case "checksum":
		return c.handleChecksum(w, r)
	case "delete":
		return c.handleDelete(w, r)
	case "clear":
		return c.handleClear(w, r)
	case "weight":
		return c.handleWeight(w, r)
	case "len":
		return c.handleLen(w, r)
	default:
		return errors.Newf("invalid action '%s'", r.URL.Query().Get("do"))
	}
}

// handleWeight handles a broadcast when the primary tries to obtain the weight of the cache held by its peers.
func (c *Cache) handleWeight(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(strconv.Itoa(c.localCache.Weight())))
	return nil
}

// handleLen handles a broadcast when the primary tries to obtain the length of the cache held by its peers.
func (c *Cache) handleLen(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(strconv.Itoa(c.localCache.Len())))
	return nil
}

// handleClear handles a broadcast when the primary tries to clear the cache held by its peers.
func (c *Cache) handleClear(w http.ResponseWriter, r *http.Request) error {
	c.localCache.Clear()
	return nil
}

// handleDelete handles a broadcast when the primary tries to delete copies held by its peers.
func (c *Cache) handleDelete(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		return nil
	}
	key := r.URL.Query().Get("key")
	if key != "" {
		c.localCache.Delete(key)
		return nil
	}
	prefix := r.URL.Query().Get("prefix")
	if prefix != "" {
		c.localCache.DeletePredicate(func(key string) bool {
			return strings.HasPrefix(key, prefix)
		})
		return nil
	}
	contains := r.URL.Query().Get("contains")
	if contains != "" {
		c.localCache.DeletePredicate(func(key string) bool {
			return strings.Contains(key, contains)
		})
		return nil
	}
	return errors.New("missing key")
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
	bump := r.URL.Query().Get("bump") == "true"
	ttl, err := time.ParseDuration(r.URL.Query().Get("ttl"))
	if err != nil {
		return errors.Trace(err)
	}
	data, ok := c.localCache.Load(key, lru.Bump(bump), lru.MaxAge(ttl))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	_, err = w.Write(data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// handleStore handles a broadcast when the primary replicates its value to peers.
func (c *Cache) handleStore(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	value, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Trace(err)
	}
	c.localCache.Store(key, value, lru.Weight(len(value)))
	return nil
}

// handleChecksum handles a broadcast when the primary tries to validate the copy it has.
func (c *Cache) handleChecksum(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from self
	if frame.Of(r).FromID() == c.svc.ID() {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	// Get arguments
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	checksum := r.URL.Query().Get("checksum")
	if checksum == "" {
		return errors.New("missing checksum")
	}
	remoteSum, err := hex.DecodeString(checksum)
	if err != nil {
		return errors.Trace(err)
	}

	// Locate the local copy
	data, ok := c.localCache.Load(key, lru.NoBump())
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	hash := sha256.New()
	_, err = hash.Write(data)
	if err != nil {
		return errors.Trace(err)
	}
	localSum := hash.Sum(nil)

	if !bytes.Equal(localSum, remoteSum) {
		// Delete the inconsistent element
		c.localCache.Delete(key)
		c.svc.LogInfo(r.Context(), "Inconsistent cache elem deleted",
			"key", key,
		)
	}

	// Return the sum to the remote
	_, err = w.Write(localSum)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// handleRescue stores an element that is offloaded when a peer shuts down.
func (c *Cache) handleRescue(w http.ResponseWriter, r *http.Request) error {
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

	c.localCache.Store(key, data, lru.Weight(weight))
	return nil
}

// Close closes and clears the cache.
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
	ch := controlapi.NewMulticastClient(c.svc).ForHost(c.svc.Hostname()).Ping(ctx)
	peers := 0
	for r := range ch {
		_, err := r.Get()
		if err == nil && frame.Of(r.HTTPResponse).FromID() != c.svc.ID() {
			peers++
		}
	}
	if peers == 0 {
		return
	}

	// Send local cache content to random peers
	elemMap := c.localCache.ToMap()
	type kv struct {
		key string
		val []byte
	}
	rescueQueue := make(chan *kv, len(elemMap))
	for k, v := range elemMap {
		rescueQueue <- &kv{k, v}
	}
	close(rescueQueue)
	var wg sync.WaitGroup
	var rescued int64
	t0 := time.Now()
	concurrent := runtime.NumCPU() * 4
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			for kv := range rescueQueue {
				if time.Since(t0) >= 4*time.Second {
					break
				}
				u := fmt.Sprintf("%s/rescue?key=%s", c.basePath, url.QueryEscape(kv.key))
				_, err := c.svc.Request(ctx, pub.Method("PUT"), pub.URL(u), pub.Body(kv.val))
				if err != nil {
					break
				}
				atomic.AddInt64(&rescued, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(t0)
	c.svc.LogDebug(ctx, "Rescued cache elements",
		"count", rescued,
		"in", dur,
	)
}

// Store an element in the cache.
func (c *Cache) Store(ctx context.Context, key string, value []byte, options ...StoreOption) error {
	if key == "" {
		return errors.New("missing key")
	}

	opts := cacheOptions{
		Replicate: false,
	}
	for _, opt := range options {
		opt(&opts)
	}

	// Delete locally
	c.localCache.Delete(key)

	if !opts.Replicate {
		// Delete at peers
		u := fmt.Sprintf("%s/all?do=delete&key=%s", c.basePath, url.QueryEscape(key))
		ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
		for r := range ch {
			_, err := r.Get()
			if err != nil {
				return errors.Trace(err)
			}
		}
	} else {
		// Replicate to peers
		u := fmt.Sprintf("%s/all?do=store&key=%s", c.basePath, url.QueryEscape(key))
		ch := c.svc.Publish(ctx, pub.Method("PUT"), pub.URL(u), pub.Body(value))
		for r := range ch {
			_, err := r.Get()
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	// Store in local cache
	c.localCache.Store(key, value, lru.Weight(len(value)))

	return nil
}

// Load an element from the cache, locally or from peers.
// If the element is found, it is bumped to the head of the cache.
func (c *Cache) Load(ctx context.Context, key string, options ...LoadOption) (value []byte, ok bool, err error) {
	if key == "" {
		return nil, false, errors.New("missing key")
	}

	opts := cacheOptions{
		Bump:             true,
		ConsistencyCheck: true,
		MaxAge:           c.MaxAge(),
	}
	for _, opt := range options {
		opt(&opts)
	}

	// Check local cache
	value, ok = c.localCache.Load(key, lru.Bump(opts.Bump), lru.MaxAge(opts.MaxAge))
	if ok {
		if !opts.ConsistencyCheck {
			atomic.AddInt64(&c.hits, 1)
			return value, true, nil
		}
		// Calculate checksum of local copy
		hash := sha256.New()
		_, err := hash.Write(value)
		if err != nil {
			return nil, false, errors.Trace(err)
		}
		localSum := hash.Sum(nil)

		// Validate with peers
		u := fmt.Sprintf("%s/all?do=checksum&checksum=%s&key=%s", c.basePath, hex.EncodeToString(localSum), url.QueryEscape(key))
		ch := c.svc.Publish(ctx, pub.GET(u))
		for r := range ch {
			res, err := r.Get()
			if err != nil {
				return nil, false, errors.Trace(err)
			}
			if res.StatusCode != http.StatusOK {
				continue
			}
			if value == nil {
				// Already deleted
				continue
			}
			remoteSum, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, false, errors.Trace(err)
			}
			if !bytes.Equal(localSum, remoteSum) {
				// Inconsistency detected
				c.localCache.Delete(key)
				c.svc.LogInfo(ctx, "Cache inconsistency",
					"key", key,
				)
				value = nil
				ok = false
			}
		}
		if ok {
			atomic.AddInt64(&c.hits, 1)
		} else {
			atomic.AddInt64(&c.misses, 1)
		}
		return value, ok, nil
	}

	// Load from peers
	value = nil
	ok = false
	u := fmt.Sprintf("%s/all?do=load&bump=%v&key=%s&ttl=%s", c.basePath, opts.Bump, url.QueryEscape(key), opts.MaxAge.String())
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
		if opts.ConsistencyCheck && value != nil && !bytes.Equal(value, data) {
			// Inconsistency detected
			c.svc.LogInfo(ctx, "Cache inconsistency",
				"key", key,
			)
			return nil, false, nil
		}
		value = data
		ok = true
	}
	if ok {
		atomic.AddInt64(&c.hits, 1)
	} else {
		atomic.AddInt64(&c.misses, 1)
	}
	return value, ok, nil
}

// Delete an element from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("missing key")
	}

	c.localCache.Delete(key)

	// Broadcast to all peers
	u := fmt.Sprintf("%s/all?do=delete&key=%s", c.basePath, url.QueryEscape(key))
	ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// DeletePrefix all element from the cache whose keys have the given prefix.
func (c *Cache) DeletePrefix(ctx context.Context, keyPrefix string) error {
	if keyPrefix == "" {
		return errors.New("missing prefix")
	}

	c.localCache.DeletePredicate(func(key string) bool {
		return strings.HasPrefix(key, keyPrefix)
	})

	// Broadcast to all peers
	u := fmt.Sprintf("%s/all?do=delete&prefix=%s", c.basePath, url.QueryEscape(keyPrefix))
	ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// DeleteContains all element from the cache whose keys contain the given substring.
func (c *Cache) DeleteContains(ctx context.Context, keySubstring string) error {
	if keySubstring == "" {
		return errors.New("missing substring")
	}

	c.localCache.DeletePredicate(func(key string) bool {
		return strings.Contains(key, keySubstring)
	})

	// Broadcast to all peers
	u := fmt.Sprintf("%s/all?do=delete&contains=%s", c.basePath, url.QueryEscape(keySubstring))
	ch := c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u))
	for r := range ch {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Weight is the total memory used by all the shards of the cache.
func (c *Cache) Weight(ctx context.Context) (int, error) {
	// Broadcast to all peers (and self)
	u := fmt.Sprintf("%s/all?do=weight", c.basePath)
	ch := c.svc.Publish(ctx, pub.Method("GET"), pub.URL(u))
	totalWeight := 0
	for r := range ch {
		res, err := r.Get()
		if err != nil {
			return 0, errors.Trace(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, errors.Trace(err)
		}
		wt, err := strconv.Atoi(string(body))
		if err != nil {
			return 0, errors.Trace(err)
		}
		totalWeight += wt
	}
	return totalWeight, nil
}

// Len is the total number of elements stored in all the shards of the cache.
func (c *Cache) Len(ctx context.Context) (int, error) {
	// Broadcast to all peers (and self)
	u := fmt.Sprintf("%s/all?do=len", c.basePath)
	ch := c.svc.Publish(ctx, pub.Method("GET"), pub.URL(u))
	totalLen := 0
	for r := range ch {
		res, err := r.Get()
		if err != nil {
			return 0, errors.Trace(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, errors.Trace(err)
		}
		len, err := strconv.Atoi(string(body))
		if err != nil {
			return 0, errors.Trace(err)
		}
		totalLen += len
	}
	return totalLen, nil
}

// Clear the cache.
func (c *Cache) Clear(ctx context.Context) error {
	// Broadcast to all peers (and self)
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

// LoadJSON loads a JSON element from the cache.
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
func (c *Cache) StoreJSON(ctx context.Context, key string, value any, options ...StoreOption) error {
	if key == "" {
		return errors.New("missing key")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.Store(ctx, key, data, options...)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// LoadCompressedJSON loads a compressed JSON element from the cache.
// If the element is found, it is bumped to the head of the cache.
func (c *Cache) LoadCompressedJSON(ctx context.Context, key string, value any, options ...LoadOption) (ok bool, err error) {
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
	err = json.NewDecoder(brotli.NewReader(bytes.NewReader(data))).Decode(value)
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// StoreCompressedJSON marshals the value as JSON and stores it in the cache compressed.
// JSON marshalling is not memory efficient and should be avoided if the cache is
// expected to store a lot of data.
func (c *Cache) StoreCompressedJSON(ctx context.Context, key string, value any, options ...StoreOption) error {
	if key == "" {
		return errors.New("missing key")
	}
	var data bytes.Buffer
	br := brotli.NewWriter(&data)
	err := json.NewEncoder(br).Encode(value)
	if err != nil {
		return errors.Trace(err)
	}
	err = br.Close()
	if err != nil {
		return errors.Trace(err)
	}
	err = c.Store(ctx, key, data.Bytes(), options...)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Hits returns the total number of cache hits.
// This number can technically overflow.
func (c *Cache) Hits() int {
	hits := atomic.LoadInt64(&c.hits)
	return int(hits)
}

// Misses returns the total number of cache misses.
// This number can technically overflow.
func (c *Cache) Misses() int {
	misses := atomic.LoadInt64(&c.misses)
	return int(misses)
}
