/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"iter"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/lru"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/seamster"
	"golang.org/x/sync/singleflight"
)

// brotliMagicWord prefixes a brotli-compressed value so compressed and uncompressed values coexist.
var brotliMagicWord = []byte{0x91, 0x19, 0x62, 0x66}

// Service is an interface abstraction of a microservice used by the distributed cache.
// The connector implements this interface.
type Service interface {
	service.PublisherSubscriber
	service.Identifier
	service.Logger
	service.Meter
}

// faultSkipLeave, when armed, makes broadcastLeave skip announcing this replica's departure,
// simulating a lost leave so the periodic discovery path can be exercised in tests.
const faultSkipLeave = "skipLeave"

// faultStaleGen, when armed, makes the next outgoing Store or Load request carry a stale generation,
// so the recipient's generation gate rejects it. Drives the gen-mismatch path in tests.
const faultStaleGen = "staleGen"

// faultSkipOffload, when armed, makes offload return without shedding, leaving displaced keys on this
// replica so the previous-generation retry path can be exercised in tests.
const faultSkipOffload = "skipOffload"

// checkpointOffloadBeforeStore fires in offload just before a displaced key's soft store is sent to its
// new owner - the key is still held locally and nothing has been deleted yet, so a test can observe the
// cache mid-shed. Free in production (the seams are disabled).
const checkpointOffloadBeforeStore = "offloadBeforeStore"

// checkpointBeforeSend fires in Store once the owner and generation have been chosen and stamped onto the
// request, but before it is sent, so a test can change the topology underneath an in-flight store.
const checkpointBeforeSend = "beforeSend"

// defaultOffloadDuration caps how long offload runs, both when shedding everything on shutdown and
// when shedding displaced keys to a new joiner. It is kept short so Close does not linger. The
// previous-generation retry window is derived as twice this value, giving the offload time to drain
// plus margin for the change to propagate.
const defaultOffloadDuration = 4 * time.Second

/*
Cache is a reworking of [Cache] that routes each operation to a single owning peer via rendezvous hashing,
rather than multicasting every operation to all peers. It is tied to the microservice and is typically constructed
in the OnStartup callback of the microservice and destroyed in the OnShutdown.

	con := connector.New("www.example.com")
	var myCache dlru.Cache
	con.SetOnStartup(func(ctx context.Context) error {
		myCache = dlru.NewCache(ctx, con, ":444/my-cache")
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		myCache.Close(ctx)
	})
*/
type Cache struct {
	localCache      *lru.Cache[string, []byte]
	basePath        string
	svc             Service
	allSubName      string
	oneSubName      string
	mux             sync.RWMutex
	pingInterval    time.Duration
	members         []string
	generation      string
	prevMembers     []string
	prevGeneration  string
	genChangedAt    time.Time
	offloadDuration time.Duration
	discoverNow     chan struct{}
	hits            atomic.Int64
	misses          atomic.Int64
	sf              singleflight.Group
	seams           *seamster.Seamster
	stopCancel      context.CancelFunc
	wg              sync.WaitGroup
}

// NewCache starts a new cache for the service at a given path.
// For security reasons, it is advised to use a non-public port for the path, such as :444/my-cache .
// By default, the cache size limit is set to 32MB and the TTL to 1 hour.
func NewCache(ctx context.Context, svc Service, path string) (*Cache, error) {
	basePath := httpx.JoinHostAndPath(svc.Hostname(), path)
	basePath = strings.TrimSuffix(basePath, "/")
	_, underTest := utils.Testing()
	c := &Cache{
		basePath:        basePath,
		svc:             svc,
		localCache:      lru.New[string, []byte](32<<20, time.Hour),
		pingInterval:    time.Minute,
		offloadDuration: defaultOffloadDuration,
		discoverNow:     make(chan struct{}, 1),
		seams:           seamster.New(underTest),
	}
	err := c.start(ctx)
	if err != nil {
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
	err := c.localCache.SetMaxWeight(megaBytes << 20)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// MaxMemory returns the memory limit of the cache.
func (c *Cache) MaxMemory() int {
	return c.localCache.MaxWeight()
}

// SetPingInterval sets how often this replica discovers its peers to recompute the membership set.
func (c *Cache) SetPingInterval(interval time.Duration) error {
	if interval <= 0 {
		return errors.New("non-positive interval")
	}
	c.mux.Lock()
	c.pingInterval = interval
	c.mux.Unlock()
	return nil
}

// PingInterval returns how often this replica discovers its peers.
func (c *Cache) PingInterval() time.Duration {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.pingInterval
}

// SetOffloadDuration caps how long offload runs on shutdown and when shedding to a joiner. The
// connector clamps this to the time budget it allows during shutdown. It also sets the
// previous-generation retry window to twice this value.
func (c *Cache) SetOffloadDuration(d time.Duration) error {
	if d <= 0 {
		return errors.New("non-positive duration")
	}
	c.mux.Lock()
	c.offloadDuration = d
	c.mux.Unlock()
	return nil
}

// OffloadDuration returns the offload time cap.
func (c *Cache) OffloadDuration() time.Duration {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.offloadDuration
}

// start registers the cache's two dispatching subscriptions and activates them on the bus.
// The "All" sub is a no-queue multicast reached by every peer; the "One" sub is a default-queue
// unicast load-balanced among peers. Both are marked [sub.Manual] so the connector's automatic
// activation passes skip them, then activated here so the cache is reachable from inside OnStartup.
func (c *Cache) start(ctx context.Context) error {
	c.allSubName = c.subscriptionName("All")
	c.oneSubName = c.subscriptionName("One")

	// Discover existing peers before going on the bus, then count self in.
	c.discoverPeers(ctx)
	c.addMember(ctx, c.svc.ID())

	err := c.svc.Subscribe(c.allSubName, c.handleAll,
		sub.At("ANY", c.basePath+"/all"),
		sub.Description("Distributed cache broadcast handler."),
		sub.Web(),
		sub.NoQueue(),
		sub.Manual(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.svc.Subscribe(c.oneSubName, c.handleOne,
		sub.At("ANY", c.basePath+"/one"),
		sub.Description("Distributed cache owner handler."),
		sub.Web(),
		sub.DefaultQueue(),
		sub.Manual(),
	)
	if err != nil {
		_ = c.svc.Unsubscribe(c.allSubName)
		return errors.Trace(err)
	}
	// Activate the owner sub before the broadcast sub. A peer only ever answers a ping or join through
	// its broadcast handler, so making the owner sub live first guarantees that any replica another node
	// learns about (from a ping response, a join reply, or as a joiner) is already able to serve store
	// and load requests - closing the window where a key could route to an owner whose /one is not yet up.
	err = c.svc.ActivateSubscription(c.oneSubName)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.svc.ActivateSubscription(c.allSubName)
	if err != nil {
		return errors.Trace(err)
	}

	// Announce our arrival so peers add us to their membership set.
	c.broadcastJoin(ctx)

	// Launch the peer discovery loop, owned by the cache and ended in Close.
	var stopCtx context.Context
	stopCtx, c.stopCancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go c.pingLoop(stopCtx)
	return nil
}

// pingLoop rediscovers peers on every ping interval, or promptly when a request to an owner times
// out, until the cache is closed.
func (c *Cache) pingLoop(ctx context.Context) {
	defer c.wg.Done()
	for {
		t := time.NewTimer(c.PingInterval())
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			c.discoverPeers(ctx)
		case <-c.discoverNow:
			t.Stop()
			c.discoverPeers(ctx)
		}
	}
}

// triggerDiscover asks the discovery loop to re-derive the membership set immediately instead of
// waiting for the next interval. It is called when a request to an owner times out (a 404 ack
// timeout), which suggests the owner has left. The signal is coalesced, so a burst of timeouts to a
// departed owner queues at most one re-derivation.
func (c *Cache) triggerDiscover() {
	select {
	case c.discoverNow <- struct{}{}:
	default:
	}
}

// discoverPeers broadcasts a ping and recomputes the membership set and the generation hash
// from the IDs of the replicas that respond, including this replica itself.
func (c *Cache) discoverPeers(ctx context.Context) {
	u := fmt.Sprintf("%s/all?do=ping", c.basePath)
	var ids []string
	for r := range c.svc.Publish(ctx, pub.GET(u)) {
		res, err := r.Get()
		if err != nil {
			continue
		}
		id := frame.Of(res).FromID()
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	c.mux.Lock()
	changed := c.applyMembersLocked(ids)
	gen := c.generation
	c.mux.Unlock()

	if changed {
		c.svc.LogDebug(ctx, "Cache membership changed",
			"members", len(ids),
			"generation", gen,
		)
	}
}

// applyMembersLocked replaces the membership set with the given sorted IDs. If the generation
// changes, the current generation is shifted into the previous slot with a fresh change timestamp,
// so a coordinator can retry the previous owner during the transition window. Caller holds c.mux.
func (c *Cache) applyMembersLocked(next []string) (changed bool) {
	gen := generationOf(next)
	if gen == c.generation {
		c.members = next
		return false
	}
	c.prevMembers = c.members
	c.prevGeneration = c.generation
	c.genChangedAt = time.Now()
	c.members = next
	c.generation = gen
	return true
}

// generationOf hashes a sorted set of peer IDs into a generation identifier.
// Peers that agree on the membership set derive the same generation with no coordination.
func generationOf(sortedIDs []string) string {
	h := sha256.New()
	for _, id := range sortedIDs {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// addMember inserts a peer ID into the membership set and recomputes the generation if it was absent.
// It reports whether the set changed.
func (c *Cache) addMember(ctx context.Context, id string) (changed bool) {
	if id == "" {
		return false
	}
	c.mux.Lock()
	present := slices.Contains(c.members, id)
	if !present {
		next := append(slices.Clone(c.members), id)
		sort.Strings(next)
		c.applyMembersLocked(next)
	}
	gen, n := c.generation, len(c.members)
	c.mux.Unlock()
	if !present {
		c.svc.LogDebug(ctx, "Cache peer joined",
			"id", id,
			"members", n,
			"generation", gen,
		)
	}
	return !present
}

// removeMember drops a peer ID from the membership set and recomputes the generation if it was present.
func (c *Cache) removeMember(ctx context.Context, id string) {
	if id == "" {
		return
	}
	c.mux.Lock()
	next := make([]string, 0, len(c.members))
	present := false
	for _, m := range c.members {
		if m == id {
			present = true
			continue
		}
		next = append(next, m)
	}
	if present {
		c.applyMembersLocked(next)
	}
	gen, n := c.generation, len(c.members)
	c.mux.Unlock()
	if present {
		c.svc.LogDebug(ctx, "Cache peer left",
			"id", id,
			"members", n,
			"generation", gen,
		)
	}
}

// broadcastJoin announces this replica to its peers so they add it to their membership set, and absorbs
// each responder's ID into its own set. The exchange is two-way so that convergence does not depend on
// activation ordering: if this replica's join raced ahead of a peer's subscription activation (so the
// peer missed it), the peer's own join reply still teaches this replica about the peer, and the peer
// learns this replica through that reply's handler.
func (c *Cache) broadcastJoin(ctx context.Context) {
	u := fmt.Sprintf("%s/all?do=join", c.basePath)
	for r := range c.svc.Publish(ctx, pub.GET(u)) {
		res, err := r.Get()
		if err != nil {
			continue
		}
		c.addMember(ctx, frame.Of(res).FromID())
	}
}

// broadcastLeave announces this replica's departure so peers remove it from their membership set.
func (c *Cache) broadcastLeave(ctx context.Context) {
	if c.seams.IsFault(faultSkipLeave) {
		return
	}
	u := fmt.Sprintf("%s/all?do=leave", c.basePath)
	for r := range c.svc.Publish(ctx, pub.GET(u)) {
		_, _ = r.Get()
	}
}

// offload ships every locally-cached element this replica no longer owns to its new owner and drops
// it locally. On shutdown self has been removed from the membership set, so every key moves; on a
// join only the keys displaced to the new peer move. The pass is bounded by OffloadDuration.
//
// The stores are fired without waiting for each response (throughput), then all the responses are
// drained once at the end. Draining is what makes it safe: it waits roughly one round trip in total
// rather than one per key, and ensures the fire-and-forget response goroutines have settled before
// offload returns - otherwise they would outlive Close and race the connector's teardown.
func (c *Cache) offload(ctx context.Context) {
	if c.seams.IsFault(faultSkipOffload) {
		return
	}
	c.mux.RLock()
	members := slices.Clone(c.members)
	gen := c.generation
	c.mux.RUnlock()
	self := c.svc.ID()

	// Bound the whole pass, firing and draining, to OffloadDuration. The timeout on the context caps
	// the stores' response goroutines so the drain cannot outrun the budget.
	octx, cancel := context.WithTimeout(ctx, c.OffloadDuration())
	defer cancel()
	var pending []iter.Seq[*pub.Response]
	for key, value := range c.localCache.ToMap() {
		if octx.Err() != nil {
			break
		}
		owner := ownerOfIn(key, members)
		if owner == "" || owner == self {
			continue
		}
		c.seams.Checkpoint(octx, checkpointOffloadBeforeStore)
		u := fmt.Sprintf("%s/one?do=store&key=%s&gen=%s&soft=true", c.ownerURL(owner), url.QueryEscape(key), url.QueryEscape(gen))
		pending = append(pending, c.svc.Publish(octx, pub.Method("PUT"), pub.URL(u), pub.Body(value)))
		c.localCache.Delete(key)
	}
	// Drain every response so the fire-and-forget goroutines settle before returning. Each store was
	// published on octx, so its goroutine times out at the OffloadDuration deadline and the drain
	// cannot exceed the budget. It must drain all of them, not stop at the deadline: a skipped
	// response would leave a goroutine running that outlives offload and races the teardown.
	for _, responses := range pending {
		for range responses {
		}
	}
}

// subscriptionName produces a per-cache, Go-style listen name (e.g. "DcacheAll", "DcacheOne").
func (c *Cache) subscriptionName(suffix string) string {
	last := c.basePath
	if i := strings.LastIndex(last, "/"); i >= 0 {
		last = last[i+1:]
	}
	if last == "" {
		last = "Cache"
	}
	last = strings.ToUpper(last[:1]) + last[1:]
	return last + suffix
}

// handleAll dispatches a no-queue multicast to the underlying action named by the "do" query argument.
func (c *Cache) handleAll(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from other hosts
	if frame.Of(r).FromHost() != c.svc.Hostname() {
		return errors.New("foreign host '%s'", frame.Of(r).FromHost())
	}
	switch r.URL.Query().Get("do") {
	case "ping":
		return c.handlePing(w, r)
	case "join":
		return c.handleJoin(w, r)
	case "leave":
		return c.handleLeave(w, r)
	case "clear":
		return c.handleClear(w, r)
	case "deletePredicate":
		return c.handleDeletePredicate(w, r)
	case "weight":
		return c.handleWeight(w, r)
	case "len":
		return c.handleLen(w, r)
	default:
		return errors.New("invalid action '%s'", r.URL.Query().Get("do"))
	}
}

// handleWeight reports this replica's local cache weight in response to a weight broadcast.
func (c *Cache) handleWeight(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(strconv.Itoa(c.localCache.Weight())))
	return nil
}

// handleLen reports this replica's local cache element count in response to a len broadcast.
func (c *Cache) handleLen(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(strconv.Itoa(c.localCache.Len())))
	return nil
}

// handleClear empties this replica's local cache in response to a clear broadcast.
func (c *Cache) handleClear(w http.ResponseWriter, r *http.Request) error {
	c.localCache.Clear()
	return nil
}

// handleDeletePredicate deletes local keys matching a prefix or a substring, in response to a
// broadcast. It walks all keys, so it is O(N) per replica; reserve it for cache-invalidation events.
func (c *Cache) handleDeletePredicate(w http.ResponseWriter, r *http.Request) error {
	if prefix := r.URL.Query().Get("prefix"); prefix != "" {
		c.localCache.DeletePredicate(func(key string) bool {
			return strings.HasPrefix(key, prefix)
		})
		return nil
	}
	if contains := r.URL.Query().Get("contains"); contains != "" {
		c.localCache.DeletePredicate(func(key string) bool {
			return strings.Contains(key, contains)
		})
		return nil
	}
	return errors.New("missing prefix or contains")
}

// handlePing acknowledges a peer discovery broadcast with an empty 200 response.
func (c *Cache) handlePing(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

// handleJoin adds the announcing peer to the membership set. A join can move ownership of some of
// this replica's keys to the new peer, so it sheds the displaced keys via offload.
func (c *Cache) handleJoin(w http.ResponseWriter, r *http.Request) error {
	if c.addMember(r.Context(), frame.Of(r).FromID()) {
		c.offload(r.Context())
	}
	return nil
}

// handleLeave removes the departing peer from the membership set.
func (c *Cache) handleLeave(w http.ResponseWriter, r *http.Request) error {
	c.removeMember(r.Context(), frame.Of(r).FromID())
	return nil
}

// handleOne dispatches a default-queue unicast to the underlying action named by the "do" query argument.
func (c *Cache) handleOne(w http.ResponseWriter, r *http.Request) error {
	// Ignore messages from other hosts
	if frame.Of(r).FromHost() != c.svc.Hostname() {
		return errors.New("foreign host '%s'", frame.Of(r).FromHost())
	}
	switch r.URL.Query().Get("do") {
	case "store":
		return c.handleStore(w, r)
	case "load":
		return c.handleLoad(w, r)
	case "delete":
		return c.handleDelete(w, r)
	default:
		return errors.New("invalid action '%s'", r.URL.Query().Get("do"))
	}
}

// handleDelete removes a key from this replica's local cache. It is unconditional - deleting is
// idempotent and safe, so it is not gated on the generation.
func (c *Cache) handleDelete(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	c.localCache.Delete(key)
	return nil
}

// handleStore stores an element routed to this replica as the key's owner. The generation gate
// rejects the store with 417 when the caller's generation differs from this replica's current view,
// since a topology disagreement means this replica may not be the rightful owner.
func (c *Cache) handleStore(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	if !c.generationMatches(r.URL.Query().Get("gen")) {
		w.WriteHeader(http.StatusExpectationFailed)
		return nil
	}
	value, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Trace(err)
	}
	c.storeLocal(key, value, r.URL.Query().Get("soft") == "true")
	return nil
}

// handleLoad returns an element held by this replica as the key's owner. The generation gate rejects
// the load with 417 on a topology disagreement; a genuine absence is 404. Both drive the coordinator's
// previous-generation retry.
func (c *Cache) handleLoad(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if key == "" {
		return errors.New("missing key")
	}
	if !c.generationMatches(r.URL.Query().Get("gen")) {
		w.WriteHeader(http.StatusExpectationFailed)
		return nil
	}
	bump := r.URL.Query().Get("bump") != "false"
	ttl := c.MaxAge()
	s := r.URL.Query().Get("ttl")
	if s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return errors.Trace(err)
		}
		ttl = d
	}
	value, ok := c.localCache.Load(key, lru.Bump(bump), lru.MaxAge(ttl))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	_, err := w.Write(value)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Close ends the peer discovery loop, announces this replica's departure, unsubscribes the cache's
// handlers, offloads owned elements to peers, and clears the local cache.
func (c *Cache) Close(ctx context.Context) error {
	// End the peer discovery loop.
	if c.stopCancel != nil {
		c.stopCancel()
		c.wg.Wait()
		c.stopCancel = nil
	}

	// Announce departure while still on the bus, then drop self from the membership set.
	c.broadcastLeave(ctx)
	c.removeMember(ctx, c.svc.ID())

	// Unsubscribe before offloading so rescued elements route only to peers.
	var lastErr error
	if c.allSubName != "" {
		err := c.svc.Unsubscribe(c.allSubName)
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.allSubName = ""
	}
	if c.oneSubName != "" {
		err := c.svc.Unsubscribe(c.oneSubName)
		if err != nil {
			lastErr = errors.Trace(err)
		}
		c.oneSubName = ""
	}

	c.offload(ctx)
	c.localCache.Clear()
	return lastErr
}

// ownerOfIn returns the ID of the member that owns the key under rendezvous (HRW) hashing: the one
// whose hash of (key, memberID) is the largest. Returns "" when the set is empty.
func ownerOfIn(key string, members []string) string {
	var owner string
	var best []byte
	for _, id := range members {
		h := sha256.New()
		h.Write([]byte(key))
		h.Write([]byte{0})
		h.Write([]byte(id))
		sum := h.Sum(nil)
		if best == nil || bytes.Compare(sum, best) > 0 {
			best = sum
			owner = id
		}
	}
	return owner
}

// ownerOf returns the ID of the peer that owns the key under the current membership set.
func (c *Cache) ownerOf(key string) string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return ownerOfIn(key, c.members)
}

// prevOwnerOf returns the owner of the key under the previous membership set and that set's
// generation, but only within the transition window after a recent membership change. Outside the
// window, or with no recorded previous generation, ok is false.
func (c *Cache) prevOwnerOf(key string) (owner, gen string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.prevGeneration == "" || time.Since(c.genChangedAt) >= 2*c.offloadDuration {
		return "", "", false
	}
	return ownerOfIn(key, c.prevMembers), c.prevGeneration, true
}

// currentGeneration returns the generation of the current membership set.
func (c *Cache) currentGeneration() string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.generation
}

// generationMatches reports whether the given generation is acceptable: it equals this replica's
// current-view generation, or, within the overlap window, its previous generation. Accepting the
// previous generation is what lets a coordinator's previous-generation retry reach the old owner,
// which has itself advanced to the new generation.
func (c *Cache) generationMatches(gen string) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if gen == c.generation {
		return true
	}
	return c.prevGeneration != "" && gen == c.prevGeneration && time.Since(c.genChangedAt) < 2*c.offloadDuration
}

// outgoingGeneration returns the generation to stamp on an outgoing request. Under the staleGen
// fault it returns a deliberately mismatching value to drive the recipient's gate in tests.
func (c *Cache) outgoingGeneration() string {
	gen := c.currentGeneration()
	if c.seams.IsFault(faultStaleGen) {
		return gen + "-stale"
	}
	return gen
}

// ownerURL inserts the owner's direct-addressing segment as the first hostname segment of the base
// path. A peer ID already carries the "id-" prefix, so it is used as-is, after the URL scheme.
func (c *Cache) ownerURL(owner string) string {
	return strings.Replace(c.basePath, "https://", "https://"+owner+".", 1)
}

// storeLocal writes a value into the local cache. A soft store only fills an absent key, using the
// lru's atomic store-if-absent, so an offloaded value cannot clobber a newer value that an upstream
// write placed on the new owner while the old owner was still shedding.
func (c *Cache) storeLocal(key string, value []byte, soft bool) {
	if soft {
		c.localCache.LoadOrStore(key, value, lru.Weight(len(value)))
		return
	}
	c.localCache.Store(key, value, lru.Weight(len(value)))
}

// Store an element in the cache, routed to the peer that owns the key. The owner drops the store if
// its generation disagrees with this replica's, which is tolerated as a future miss.
func (c *Cache) Store(ctx context.Context, key string, value []byte, options ...StoreOption) error {
	if key == "" {
		return errors.New("missing key")
	}
	opts := cacheOptions{}
	for _, opt := range options {
		opt(&opts)
	}
	if opts.Compress {
		compressed, err := compress(value)
		if err != nil {
			return errors.Trace(err)
		}
		value = compressed
	}
	owner := c.ownerOf(key)
	if owner == "" || owner == c.svc.ID() {
		c.storeLocal(key, value, false)
		c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "store")
		return nil
	}
	u := fmt.Sprintf("%s/one?do=store&key=%s&gen=%s", c.ownerURL(owner), url.QueryEscape(key), url.QueryEscape(c.outgoingGeneration()))
	c.seams.Checkpoint(ctx, checkpointBeforeSend)
	res, err := c.svc.Request(ctx, pub.Method("PUT"), pub.URL(u), pub.Body(value))
	if err != nil {
		// A dead owner acks with a 404 timeout; the store is dropped (a future miss), and membership
		// is re-derived so the next store routes to a live owner.
		if errors.StatusCode(err) == http.StatusNotFound {
			c.triggerDiscover()
			return nil
		}
		return errors.Trace(err)
	}
	if res.StatusCode == http.StatusExpectationFailed {
		// The owner disagreed on the generation and dropped the store; a future load recomputes.
		c.svc.LogDebug(ctx, "Cache store rejected on generation mismatch", "key", key)
	}
	c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "store")
	return nil
}

// Load an element from the cache, routed to the peer that owns the key. On a miss within the
// transition window, the previous generation's owner is retried to find a key still on the old owner.
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
	owner := c.ownerOf(key)
	value, ok, err = c.loadFromOwner(ctx, owner, key, c.outgoingGeneration(), opts)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	served := owner
	if !ok {
		// Previous-generation retry: a key whose ownership just moved is a miss on the new owner but may
		// still sit on the old owner.
		prevOwner, prevGen, within := c.prevOwnerOf(key)
		if within && prevOwner != owner {
			value, ok, err = c.loadFromOwner(ctx, prevOwner, key, prevGen, opts)
			if err != nil {
				return nil, false, errors.Trace(err)
			}
			served = prevOwner
		}
	}
	if ok {
		value, err = decompress(value)
		if err != nil {
			return nil, false, errors.Trace(err)
		}
		hit := "remote"
		if served == "" || served == c.svc.ID() {
			hit = "local"
		}
		c.hits.Add(1)
		c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "load", "hit", hit)
		return value, true, nil
	}
	c.misses.Add(1)
	c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "load", "hit", "miss")
	return nil, false, nil
}

// loadFromOwner loads a key from a specific owner, serving locally when this replica is the owner and
// otherwise routing a unicast stamped with gen. A dead owner, a genuine 404, and a 417 gen-mismatch
// all read as a miss.
func (c *Cache) loadFromOwner(ctx context.Context, owner, key, gen string, opts cacheOptions) (value []byte, ok bool, err error) {
	if owner == "" || owner == c.svc.ID() {
		value, found := c.localCache.Load(key, lru.Bump(opts.Bump), lru.MaxAge(opts.MaxAge))
		if !found {
			return nil, false, nil
		}
		return value, true, nil
	}
	u := fmt.Sprintf("%s/one?do=load&key=%s&gen=%s&bump=%v&ttl=%s", c.ownerURL(owner), url.QueryEscape(key), url.QueryEscape(gen), opts.Bump, opts.MaxAge.String())
	res, err := c.svc.Request(ctx, pub.Method("GET"), pub.URL(u))
	if err != nil {
		// A dead owner acks with a 404 timeout; treat it as a miss and re-derive membership.
		if errors.StatusCode(err) == http.StatusNotFound {
			c.triggerDiscover()
			return nil, false, nil
		}
		return nil, false, errors.Trace(err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, false, nil
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	return data, true, nil
}

// Delete a key from the cache. It is removed from the current owner and, within the transition
// window, from the previous owner too, so a key still sitting on the old owner is not left behind
// (nor resurrected by that owner's pending offload, which has nothing left to shed).
func (c *Cache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("missing key")
	}
	owner := c.ownerOf(key)
	err := c.deleteFromOwner(ctx, owner, key)
	if err != nil {
		return errors.Trace(err)
	}
	prevOwner, _, within := c.prevOwnerOf(key)
	if within && prevOwner != owner {
		err = c.deleteFromOwner(ctx, prevOwner, key)
		if err != nil {
			return errors.Trace(err)
		}
	}
	c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "delete")
	return nil
}

// deleteFromOwner removes a key from a specific owner, deleting locally when this replica is the
// owner and otherwise routing a unicast. A dead owner cannot hold the key, so its 404 ack timeout is
// treated as done and triggers a membership re-derivation.
func (c *Cache) deleteFromOwner(ctx context.Context, owner, key string) error {
	if owner == "" || owner == c.svc.ID() {
		c.localCache.Delete(key)
		return nil
	}
	u := fmt.Sprintf("%s/one?do=delete&key=%s", c.ownerURL(owner), url.QueryEscape(key))
	_, err := c.svc.Request(ctx, pub.Method("DELETE"), pub.URL(u))
	if err != nil {
		if errors.StatusCode(err) == http.StatusNotFound {
			c.triggerDiscover()
			return nil
		}
		return errors.Trace(err)
	}
	return nil
}

// Clear empties the cache on every replica. The broadcast reaches this replica too, so it clears its
// own local cache through the same handler.
func (c *Cache) Clear(ctx context.Context) error {
	u := fmt.Sprintf("%s/all?do=clear", c.basePath)
	for r := range c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u)) {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// DeletePrefix deletes every key with the given prefix from all replicas. Because matching keys are
// spread across owners, it broadcasts and each replica walks its own keys (O(N)); reserve it for
// cache-invalidation events, not routine deletes.
func (c *Cache) DeletePrefix(ctx context.Context, keyPrefix string) error {
	if keyPrefix == "" {
		return errors.New("missing prefix")
	}
	u := fmt.Sprintf("%s/all?do=deletePredicate&prefix=%s", c.basePath, url.QueryEscape(keyPrefix))
	for r := range c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u)) {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "delete")
	return nil
}

// DeleteContains deletes every key containing the given substring from all replicas. Like
// DeletePrefix, it broadcasts and each replica walks its own keys (O(N)).
func (c *Cache) DeleteContains(ctx context.Context, keySubstring string) error {
	if keySubstring == "" {
		return errors.New("missing substring")
	}
	u := fmt.Sprintf("%s/all?do=deletePredicate&contains=%s", c.basePath, url.QueryEscape(keySubstring))
	for r := range c.svc.Publish(ctx, pub.Method("DELETE"), pub.URL(u)) {
		_, err := r.Get()
		if err != nil {
			return errors.Trace(err)
		}
	}
	c.svc.IncrementCounter(ctx, "microbus_cache_operations", 1, "op", "delete")
	return nil
}

// Weight is the total memory used by all the shards of the cache. Each key lives on a single owner,
// so the broadcast sums disjoint local weights into the cluster-wide total.
func (c *Cache) Weight(ctx context.Context) (int, error) {
	u := fmt.Sprintf("%s/all?do=weight", c.basePath)
	total := 0
	for r := range c.svc.Publish(ctx, pub.GET(u)) {
		res, err := r.Get()
		if err != nil {
			return 0, errors.Trace(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, errors.Trace(err)
		}
		wt, err := strconv.Atoi(utils.UnsafeBytesToString(body))
		if err != nil {
			return 0, errors.Trace(err)
		}
		total += wt
	}
	return total, nil
}

// Len is the total number of elements stored in all the shards of the cache. Each key lives on a
// single owner, so the broadcast sums disjoint local counts into the cluster-wide total.
func (c *Cache) Len(ctx context.Context) (int, error) {
	u := fmt.Sprintf("%s/all?do=len", c.basePath)
	total := 0
	for r := range c.svc.Publish(ctx, pub.GET(u)) {
		res, err := r.Get()
		if err != nil {
			return 0, errors.Trace(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, errors.Trace(err)
		}
		n, err := strconv.Atoi(utils.UnsafeBytesToString(body))
		if err != nil {
			return 0, errors.Trace(err)
		}
		total += n
	}
	return total, nil
}

// Hits returns the total number of cache hits. This number can technically overflow.
func (c *Cache) Hits() int {
	return int(c.hits.Load())
}

// Misses returns the total number of cache misses. This number can technically overflow.
func (c *Cache) Misses() int {
	return int(c.misses.Load())
}

// compress prefixes the value with the brotli magic word and brotli-compresses it. The prefix lets
// compressed and uncompressed values coexist in the same cache; decompress reads it back.
func compress(value []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(brotliMagicWord)
	br := brotli.NewWriterLevel(&buf, brotli.BestSpeed)
	_, err := br.Write(value)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = br.Close()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return buf.Bytes(), nil
}

// decompress reverses compress when the value carries the brotli magic-word prefix, and returns the
// value unchanged otherwise.
func decompress(value []byte) ([]byte, error) {
	if !bytes.HasPrefix(value, brotliMagicWord) {
		return value, nil
	}
	var buf bytes.Buffer
	br := brotli.NewReader(bytes.NewReader(value[len(brotliMagicWord):]))
	_, err := io.Copy(&buf, br)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return buf.Bytes(), nil
}

// marshalValue converts an untyped value into bytes.
// The value must be either []byte, string, or an object that can be marshaled to JSON.
func marshalValue(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return utils.UnsafeStringToBytes(v), nil
	case []byte:
		return v, nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return data, nil
	}
}

// unmarshalValue converts bytes into an untyped value.
// The value must be a pointer to []byte, string, or an object that can be unmarshaled from JSON.
func unmarshalValue(data []byte, value any) error {
	switch v := value.(type) {
	case *string:
		*v = string(data)
	case *[]byte:
		*v = data
	default:
		err := json.Unmarshal(data, value)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Set puts an element in the cache, marking it as the most recently used.
// The value must be either []byte, string, or an object that can be marshaled to JSON.
func (c *Cache) Set(ctx context.Context, key string, value any, options ...StoreOption) error {
	if key == "" {
		return errors.New("missing key")
	}
	data, err := marshalValue(value)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.Store(ctx, key, data, options...)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Get retrieves an element from the cache, marking it as the most recently used.
// The value must be a pointer to []byte, string, or an object that can be unmarshaled from JSON.
func (c *Cache) Get(ctx context.Context, key string, value any, options ...LoadOption) (found bool, err error) {
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
	err = unmarshalValue(data, value)
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// Peek retrieves an element from the cache, without marking it as the most recently used.
// The value must be a pointer to []byte, string, or an object that can be unmarshaled from JSON.
func (c *Cache) Peek(ctx context.Context, key string, value any, options ...LoadOption) (found bool, err error) {
	if key == "" {
		return false, errors.New("missing key")
	}
	options = append(options, NoBump())
	data, ok, err := c.Load(ctx, key, options...)
	if err != nil {
		return false, errors.Trace(err)
	}
	if !ok {
		return false, nil
	}
	err = unmarshalValue(data, value)
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// Has checks if an element is in the cache, without marking it as the most recently used.
func (c *Cache) Has(ctx context.Context, key string, options ...LoadOption) (found bool, err error) {
	if key == "" {
		return false, errors.New("missing key")
	}
	options = append(options, NoBump())
	_, ok, err := c.Load(ctx, key, options...)
	if err != nil {
		return false, errors.Trace(err)
	}
	return ok, nil
}

// LoadOrCompute returns the value for key from the cache. On miss, maker is called to produce the
// value, which is stored in the cache and returned. Concurrent callers in the same process for the
// same key share a single maker invocation, preventing cache stampede.
//
// If maker returns an error, the value is not cached, the error is returned to all waiters, and the
// next caller will retry. Stampede protection is per-process: with N peers, up to N concurrent maker
// invocations may occur on a cold key.
func (c *Cache) LoadOrCompute(ctx context.Context, key string, maker func(ctx context.Context) ([]byte, error), options ...StoreOption) (value []byte, err error) {
	if key == "" {
		return nil, errors.New("missing key")
	}
	if maker == nil {
		return nil, errors.New("missing maker")
	}
	// Fast path: cache hit without entering the singleflight group.
	value, ok, err := c.Load(ctx, key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if ok {
		return value, nil
	}
	// Slow path: dedup concurrent makers per key.
	v, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check after acquiring the singleflight slot in case the cache was populated between our
		// fast-path miss and entering the group.
		value, ok, err := c.Load(ctx, key)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if ok {
			return value, nil
		}
		value, err = maker(ctx)
		if err != nil {
			return nil, err // No trace
		}
		err = c.Store(ctx, key, value, options...)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return value, nil
	})
	if err != nil {
		return nil, err // No trace
	}
	return v.([]byte), nil
}

// GetOrCompute retrieves the value for key from the cache. On miss, maker is called to produce the
// value, which is stored in the cache and returned. Concurrent callers in the same process for the
// same key share a single maker invocation, preventing cache stampede.
//
// The value parameter must be a pointer to []byte, string, or an object that can be unmarshaled from
// JSON. Maker returns the typed value to be marshaled and stored.
//
// If maker returns an error, the value is not cached, the error is returned to all waiters, and the
// next caller will retry.
func (c *Cache) GetOrCompute(ctx context.Context, key string, value any, maker func(ctx context.Context) (any, error), options ...StoreOption) error {
	if maker == nil {
		return errors.New("missing maker")
	}
	data, err := c.LoadOrCompute(ctx, key, func(ctx context.Context) ([]byte, error) {
		v, err := maker(ctx)
		if err != nil {
			return nil, err // No trace
		}
		return marshalValue(v)
	}, options...)
	if err != nil {
		return err // No trace
	}
	err = unmarshalValue(data, value)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
