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

package dlru_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/dlru"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"
)

// eventually polls cond until it returns true or the deadline elapses.
func eventually(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDLRU_Membership(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	// A single replica sees only itself.
	alpha := connector.New("membership.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)
	err = alphaCache.SetPingInterval(time.Second)
	assert.NoError(err)

	members := alphaCache.Members()
	assert.Equal([]string{alpha.ID()}, members)
	gen1 := alphaCache.Generation()
	assert.NotEqual("", gen1)

	// A second replica joins. Both should see each other, on the same generation, and it should change.
	beta := connector.New("membership.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	err = betaCache.SetPingInterval(time.Second)
	assert.NoError(err)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)

	// beta learned alpha before subscribing, then counted itself in.
	betaMembers := betaCache.Members()
	assert.Equal(both, betaMembers)
	// alpha learned beta from its join broadcast.
	alphaSawBeta := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both)
	})
	assert.True(alphaSawBeta)

	gen2 := alphaCache.Generation()
	betaGen := betaCache.Generation()
	assert.Equal(gen2, betaGen) // both derive the same generation
	assert.NotEqual(gen1, gen2) // and it is not the single-replica generation

	// Simulate an ungraceful departure: beta closes but its leave announcement is lost (fault-injected).
	// alpha's periodic ping should then drop beta on its own.
	betaCache.Seams().Inject(dlru.FaultSkipLeave)
	err = betaCache.Close(ctx)
	assert.NoError(err)

	alphaDroppedBeta := eventually(3*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), []string{alpha.ID()})
	})
	assert.True(alphaDroppedBeta)
	finalGen := alphaCache.Generation()
	assert.Equal(gen1, finalGen) // generation returns to the single-replica value
}

func TestDLRU_StoreLoad(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("storeload.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("storeload.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	// Both replicas should agree on the two-member set before routing.
	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Store a spread of keys via alpha. HRW routes each to its owner, which may be alpha or beta.
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}

	// Each key is held by exactly one replica (single copy), and loadable from either.
	for _, k := range keys {
		_, okA := alphaCache.LocalCache().Load(k)
		_, okB := betaCache.LocalCache().Load(k)
		assert.True(okA != okB) // exactly one owner holds it

		valA, foundA, err := alphaCache.Load(ctx, k)
		assert.NoError(err)
		assert.True(foundA)
		assert.Equal("v-"+k, string(valA))

		valB, foundB, err := betaCache.Load(ctx, k)
		assert.NoError(err)
		assert.True(foundB)
		assert.Equal("v-"+k, string(valB))
	}

	// The total element count across both replicas equals the number of distinct keys.
	total := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(len(keys), total)

	// A missing key is a clean miss from either replica.
	_, found, err := betaCache.Load(ctx, "missing")
	assert.NoError(err)
	assert.False(found)
}

func TestDLRU_GenerationGate(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("gengate.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("gengate.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Find two keys owned by beta, so alpha routes them remotely and the owner's gate applies.
	var remote []string
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		if alphaCache.OwnerOf(k) == beta.ID() {
			remote = append(remote, k)
			if len(remote) == 2 {
				break
			}
		}
	}
	numRemote := len(remote)
	assert.Equal(2, numRemote)
	keyHeld, keyRejected := remote[0], remote[1]

	// Baseline: a remote store with a matching generation lands and loads back.
	err = alphaCache.Store(ctx, keyHeld, []byte("v"))
	assert.NoError(err)
	val, ok, err := alphaCache.Load(ctx, keyHeld)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("v", string(val))

	// A stale-generation store is refused by the owner (417) and never lands.
	alphaCache.Seams().Inject(dlru.FaultStaleGen)
	err = alphaCache.Store(ctx, keyRejected, []byte("v2"))
	assert.NoError(err)
	_, rejectedFound, err := alphaCache.Load(ctx, keyRejected)
	assert.NoError(err)
	assert.False(rejectedFound)

	// A stale-generation load is refused even though the owner holds the value; the prev-gen retry
	// falls back to this replica, which is not the owner, so it is a clean miss.
	alphaCache.Seams().Inject(dlru.FaultStaleGen)
	_, staleFound, err := alphaCache.Load(ctx, keyHeld)
	assert.NoError(err)
	assert.False(staleFound)

	// Without the fault the owner still holds and serves it.
	val, ok, err = alphaCache.Load(ctx, keyHeld)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("v", string(val))
}

func TestDLRU_OffloadOnShutdown(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("offshut.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)

	beta := connector.New("offshut.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Store only keys that alpha owns, so they all sit on alpha and must move when it leaves.
	var alphaKeys []string
	for i := 0; i < 60 && len(alphaKeys) < 5; i++ {
		k := "key" + strconv.Itoa(i)
		if alphaCache.OwnerOf(k) == alpha.ID() {
			alphaKeys = append(alphaKeys, k)
		}
	}
	numKeys := len(alphaKeys)
	assert.Equal(5, numKeys)
	for _, k := range alphaKeys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}
	alphaLen := alphaCache.LocalCache().Len()
	assert.Equal(numKeys, alphaLen)
	betaLen := betaCache.LocalCache().Len()
	assert.Equal(0, betaLen)

	// Closing alpha offloads its keys to beta, the sole survivor.
	err = alphaCache.Close(ctx)
	assert.NoError(err)

	betaSole := eventually(2*time.Second, func() bool {
		return slices.Equal(betaCache.Members(), []string{beta.ID()})
	})
	assert.True(betaSole)
	relocated := eventually(2*time.Second, func() bool {
		return betaCache.LocalCache().Len() == numKeys
	})
	assert.True(relocated)

	// Every offloaded key is now held and served by beta.
	for _, k := range alphaKeys {
		val, ok, err := betaCache.Load(ctx, k)
		assert.NoError(err)
		assert.True(ok)
		assert.Equal("v-"+k, string(val))
	}
}

func TestDLRU_RediscoverOn404(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("rediscover.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("rediscover.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// A key owned by beta, so alpha routes it there.
	var betaKey string
	for i := 0; i < 40 && betaKey == ""; i++ {
		k := "key" + strconv.Itoa(i)
		if alphaCache.OwnerOf(k) == beta.ID() {
			betaKey = k
		}
	}
	assert.NotEqual("", betaKey)

	// beta dies ungracefully: no leave broadcast and no offload, so alpha keeps routing betaKey to it.
	betaCache.Seams().Inject(dlru.FaultSkipLeave)
	betaCache.Seams().Inject(dlru.FaultSkipOffload)
	err = betaCache.Close(ctx)
	assert.NoError(err)
	err = beta.Shutdown(ctx)
	assert.NoError(err)

	// Nothing has told alpha that beta is gone; it still lists both.
	stillBoth := slices.Equal(alphaCache.Members(), both)
	assert.True(stillBoth)

	// Loading the beta-owned key hits a 404 to the dead owner, which triggers a re-discovery. The
	// default ping interval is a minute, so dropping beta within seconds proves the 404 trigger fired.
	_, _, err = alphaCache.Load(ctx, betaKey)
	assert.NoError(err)
	dropped := eventually(3*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), []string{alpha.ID()})
	})
	assert.True(dropped)
}

func TestDLRU_OffloadDoesNotClobber(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("noclobber.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	// alpha is closed explicitly mid-test to drive its offload.

	// alpha is the sole owner; store V1, on alpha.
	var keys []string
	for i := 0; i < 12; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("V1"))
		assert.NoError(err)
	}

	// Skip alpha's shed on join, so V1 stays on alpha, the old owner.
	alphaCache.Seams().Inject(dlru.FaultSkipOffload)

	beta := connector.New("noclobber.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// A key that beta now owns, so an upstream write of V2 lands on beta while alpha still holds V1.
	var moved string
	for _, k := range keys {
		if betaCache.OwnerOf(k) == beta.ID() {
			moved = k
			break
		}
	}
	assert.NotEqual("", moved)
	err = betaCache.Store(ctx, moved, []byte("V2"))
	assert.NoError(err)

	// alpha leaves and offloads its stale V1 to beta. The soft store must not overwrite V2.
	err = alphaCache.Close(ctx)
	assert.NoError(err)

	betaSole := eventually(2*time.Second, func() bool {
		return slices.Equal(betaCache.Members(), []string{beta.ID()})
	})
	assert.True(betaSole)

	val, ok, err := betaCache.Load(ctx, moved)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("V2", string(val)) // the upstream write survived the offload
}

func TestDLRU_Delete(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("del.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("del.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	var keys []string
	for i := 0; i < 10; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}
	total := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(len(keys), total)

	// Deleting each key removes it from its owner, and it loads as a miss from either replica.
	for _, k := range keys {
		err = alphaCache.Delete(ctx, k)
		assert.NoError(err)
	}
	remaining := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(0, remaining)
	for _, k := range keys {
		_, found, err := betaCache.Load(ctx, k)
		assert.NoError(err)
		assert.False(found)
	}
}

func TestDLRU_DeleteReachesOldOwner(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("delold.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	var keys []string
	for i := 0; i < 12; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}

	// Skip the shed so keys stay on alpha, the old owner, when beta joins.
	alphaCache.Seams().Inject(dlru.FaultSkipOffload)

	beta := connector.New("delold.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// A key that beta now owns but that the skipped shed left on alpha.
	var moved string
	for _, k := range keys {
		if betaCache.OwnerOf(k) == beta.ID() {
			moved = k
			break
		}
	}
	assert.NotEqual("", moved)
	_, onAlpha := alphaCache.LocalCache().Load(moved)
	assert.True(onAlpha)

	// Deleting via beta, the new owner, must also reach alpha, the previous owner, within the window.
	err = betaCache.Delete(ctx, moved)
	assert.NoError(err)
	_, stillOnAlpha := alphaCache.LocalCache().Load(moved)
	assert.False(stillOnAlpha)
}

func TestDLRU_Clear(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("clear.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("clear.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	var keys []string
	for i := 0; i < 10; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}
	total := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(len(keys), total)

	// Clear empties every replica's local cache.
	err = alphaCache.Clear(ctx)
	assert.NoError(err)
	cleared := eventually(2*time.Second, func() bool {
		return alphaCache.LocalCache().Len() == 0 && betaCache.LocalCache().Len() == 0
	})
	assert.True(cleared)
}

func TestDLRU_DeletePredicate(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("delpred.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("delpred.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// A mix of keys, spread across both owners by HRW.
	var userKeys, orderKeys []string
	for i := 0; i < 8; i++ {
		userKeys = append(userKeys, "user:"+strconv.Itoa(i))
		orderKeys = append(orderKeys, "order:"+strconv.Itoa(i))
	}
	for _, k := range append(append([]string{}, userKeys...), orderKeys...) {
		err = alphaCache.Store(ctx, k, []byte("v"))
		assert.NoError(err)
	}
	total := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(len(userKeys)+len(orderKeys), total)

	// DeletePrefix removes the user:* family from every replica; order:* survives.
	err = alphaCache.DeletePrefix(ctx, "user:")
	assert.NoError(err)
	afterPrefix := eventually(2*time.Second, func() bool {
		return alphaCache.LocalCache().Len()+betaCache.LocalCache().Len() == len(orderKeys)
	})
	assert.True(afterPrefix)
	for _, k := range userKeys {
		_, found, err := betaCache.Load(ctx, k)
		assert.NoError(err)
		assert.False(found)
	}

	// DeleteContains removes what is left; a substring shared by all order keys clears them.
	err = alphaCache.DeleteContains(ctx, "order")
	assert.NoError(err)
	afterContains := eventually(2*time.Second, func() bool {
		return alphaCache.LocalCache().Len()+betaCache.LocalCache().Len() == 0
	})
	assert.True(afterContains)
}

func TestDLRU_PrevGenRetry(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("prevgen.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	// alpha is the sole owner; store keys, all on alpha.
	var keys []string
	for i := 0; i < 12; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}

	// Skip the shed so keys stay on alpha, the old owner, when beta joins.
	alphaCache.Seams().Inject(dlru.FaultSkipOffload)

	beta := connector.New("prevgen.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Find a key whose ownership moved to beta but which the skipped shed left on alpha.
	var moved string
	for _, k := range keys {
		if betaCache.OwnerOf(k) == beta.ID() {
			moved = k
			break
		}
	}
	assert.NotEqual("", moved)
	_, onAlpha := alphaCache.LocalCache().Load(moved)
	assert.True(onAlpha) // still on the old owner
	_, onBeta := betaCache.LocalCache().Load(moved)
	assert.False(onBeta) // shed was skipped, so the new owner does not hold it

	// Loading from beta, the new owner, misses locally, then the previous-generation retry finds it
	// on alpha, the old owner, whose gate accepts the previous generation within the overlap window.
	val, ok, err := betaCache.Load(ctx, moved)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("v-"+moved, string(val))
}

func TestDLRU_OffloadOnJoin(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("offjoin.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	// alpha is the sole owner, so every stored key lands on alpha.
	var keys []string
	for i := 0; i < 10; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("v-"+k))
		assert.NoError(err)
	}
	alphaLen := alphaCache.LocalCache().Len()
	assert.Equal(len(keys), alphaLen)

	// beta joins; alpha sheds the keys whose ownership moved to beta.
	beta := connector.New("offjoin.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// After the shed, the copies are conserved and each key sits on its HRW owner.
	settled := eventually(2*time.Second, func() bool {
		return alphaCache.LocalCache().Len()+betaCache.LocalCache().Len() == len(keys)
	})
	assert.True(settled)
	for _, k := range keys {
		owner := alphaCache.OwnerOf(k)
		_, onAlpha := alphaCache.LocalCache().Load(k)
		_, onBeta := betaCache.LocalCache().Load(k)
		if owner == beta.ID() {
			assert.True(onBeta)   // shed to beta
			assert.False(onAlpha) // removed from alpha
		} else {
			assert.True(onAlpha) // kept on alpha
			assert.False(onBeta)
		}
		val, ok, err := betaCache.Load(ctx, k)
		assert.NoError(err)
		assert.True(ok)
		assert.Equal("v-"+k, string(val))
	}
}

func TestDLRU_Accessors(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("accessors.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("accessors.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Set/Get round-trip a JSON-marshaled value across the owner boundary.
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	for i, k := range []string{"p1", "p2", "p3", "p4", "p5", "p6"} {
		err = alphaCache.Set(ctx, k, point{X: i, Y: i * 2})
		assert.NoError(err)
	}
	var got point
	found, err := betaCache.Get(ctx, "p3", &got)
	assert.NoError(err)
	assert.True(found)
	assert.Equal(point{X: 2, Y: 4}, got)

	// A missing key is a clean not-found.
	found, err = betaCache.Get(ctx, "absent", &got)
	assert.NoError(err)
	assert.False(found)

	// Weight and Len aggregate the disjoint per-replica shards into the cluster totals.
	n, err := alphaCache.Len(ctx)
	assert.NoError(err)
	assert.Equal(6, n)
	localTotal := alphaCache.LocalCache().Len() + betaCache.LocalCache().Len()
	assert.Equal(6, localTotal)

	wt, err := alphaCache.Weight(ctx)
	assert.NoError(err)
	assert.True(wt > 0)
	assert.Equal(alphaCache.LocalCache().Weight()+betaCache.LocalCache().Weight(), wt)

	// Hits and Misses accumulate across the accessors used above. A hit and a miss are guaranteed.
	assert.True(alphaCache.Hits() > 0 || betaCache.Hits() > 0)
	assert.True(betaCache.Misses() > 0)
}

func TestDLRU_LoadOrCompute(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("loadorcompute.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	// First call computes and stores; a second call hits the cache without recomputing.
	calls := 0
	maker := func(ctx context.Context) ([]byte, error) {
		calls++
		return []byte("computed"), nil
	}
	v, err := alphaCache.LoadOrCompute(ctx, "k", maker)
	assert.NoError(err)
	assert.Equal("computed", string(v))
	assert.Equal(1, calls)

	v, err = alphaCache.LoadOrCompute(ctx, "k", maker)
	assert.NoError(err)
	assert.Equal("computed", string(v))
	assert.Equal(1, calls)

	// A maker error is not cached and propagates to the caller.
	boom := errors.New("boom")
	_, err = alphaCache.LoadOrCompute(ctx, "err", func(ctx context.Context) ([]byte, error) {
		return nil, boom
	})
	assert.Error(err)

	// GetOrCompute marshals the typed value and round-trips it.
	type box struct {
		N int `json:"n"`
	}
	var out box
	err = alphaCache.GetOrCompute(ctx, "box", &out, func(ctx context.Context) (any, error) {
		return box{N: 42}, nil
	})
	assert.NoError(err)
	assert.Equal(42, out.N)
}

func TestDLRU_Options(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("options.cache.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("options.cache.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// Find a key owned by beta so the option travels over the wire, not just the local path.
	remoteKey := ""
	for i := 0; i < 100; i++ {
		k := "remote-" + strconv.Itoa(i)
		if alphaCache.OwnerOf(k) == beta.ID() {
			remoteKey = k
			break
		}
	}
	assert.NotEqual("", remoteKey)

	// Compress: the stored bytes at the owner are not the plaintext, but Load decompresses back to it.
	payload := []byte(strings.Repeat("compress me ", 64))
	err = alphaCache.Store(ctx, remoteKey, payload, dlru.Compress(true))
	assert.NoError(err)
	stored, ok := betaCache.LocalCache().Load(remoteKey)
	assert.True(ok)
	assert.NotEqual(string(payload), string(stored))
	got, ok, err := alphaCache.Load(ctx, remoteKey)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal(string(payload), string(got))

	// MaxAge: a tiny max-age on load discards a value not bumped within the window, yielding a miss
	// even though the owner still holds it.
	ageKey := ""
	for i := 0; i < 100; i++ {
		k := "age-" + strconv.Itoa(i)
		if alphaCache.OwnerOf(k) == beta.ID() {
			ageKey = k
			break
		}
	}
	assert.NotEqual("", ageKey)
	err = alphaCache.Store(ctx, ageKey, []byte("v"))
	assert.NoError(err)
	slept := alpha.Sleep(ctx, 10*time.Millisecond)
	assert.NoError(slept)
	_, ok, err = alphaCache.Load(ctx, ageKey, dlru.MaxAge(time.Millisecond))
	assert.NoError(err)
	assert.False(ok)

	// Has and Peek observe presence without requiring the value to be bumped.
	err = alphaCache.Set(ctx, "present", "yes")
	assert.NoError(err)
	has, err := betaCache.Has(ctx, "present")
	assert.NoError(err)
	assert.True(has)
	has, err = betaCache.Has(ctx, "absent")
	assert.NoError(err)
	assert.False(has)
	var peeked string
	found, err := betaCache.Peek(ctx, "present", &peeked)
	assert.NoError(err)
	assert.True(found)
	assert.Equal("yes", peeked)
}

// startConn boots a connector for the given host, failing the test on error.
func startConn(t testing.TB, ctx context.Context, host string) *connector.Connector {
	con := connector.New(host)
	err := con.Startup(ctx)
	if err != nil {
		t.Fatalf("startup %s: %v", host, err)
	}
	return con
}

// newCache builds a Cache on the connector at a fixed path, with short intervals suited to tests.
func newCache(t testing.TB, ctx context.Context, con *connector.Connector) *dlru.Cache {
	c, err := dlru.NewCache(ctx, con, ":444/testcache")
	if err != nil {
		t.Fatalf("newcache: %v", err)
	}
	err = c.SetPingInterval(250 * time.Millisecond)
	if err != nil {
		t.Fatalf("setpinginterval: %v", err)
	}
	err = c.SetOffloadDuration(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("setoffloadduration: %v", err)
	}
	return c
}

// converge waits until every cache sees the full membership set.
func converge(t testing.TB, caches ...*dlru.Cache) {
	ok := eventually(3*time.Second, func() bool {
		for _, c := range caches {
			if len(c.Members()) != len(caches) {
				return false
			}
		}
		return true
	})
	if !ok {
		t.Fatalf("caches did not converge to %d members", len(caches))
	}
}

// keyOwnedBy returns a key whose HRW owner is the given peer ID.
func keyOwnedBy(c *dlru.Cache, prefix, owner string) string {
	for i := 0; ; i++ {
		k := prefix + strconv.Itoa(i)
		if c.OwnerOf(k) == owner {
			return k
		}
	}
}

// TestDLRU_RandomActions drives a random sequence of store/load/delete across three replicas and
// checks every load against a reference model. Ported from TestDLRU_RandomActions, extended to actually
// exercise delete (the original's delete branch was unreachable), and to route through Cache's single
// owner rather than the write-local model.
func TestDLRU_RandomActions(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	host := "random.actions.cache.dlru"
	alpha := startConn(t, ctx, host)
	defer alpha.Shutdown(ctx)
	beta := startConn(t, ctx, host)
	defer beta.Shutdown(ctx)
	gamma := startConn(t, ctx, host)
	defer gamma.Shutdown(ctx)

	alphaCache := newCache(t, ctx, alpha)
	defer alphaCache.Close(ctx)
	betaCache := newCache(t, ctx, beta)
	defer betaCache.Close(ctx)
	gammaCache := newCache(t, ctx, gamma)
	defer gammaCache.Close(ctx)
	converge(t, alphaCache, betaCache, gammaCache)

	caches := []*dlru.Cache{alphaCache, betaCache, gammaCache}
	state := map[string][]byte{}
	for range 5000 {
		cache := caches[rand.IntN(len(caches))]
		key := strconv.Itoa(rand.IntN(20))
		switch rand.IntN(4) {
		case 0, 1: // Load
			bump := rand.IntN(2) == 1
			val1, ok1, err := cache.Load(ctx, key, dlru.Bump(bump))
			assert.NoError(err)
			val2, ok2 := state[key]
			assert.Equal(ok2, ok1)
			assert.Equal(val2, val1)

		case 2: // Store
			val := []byte(utils.RandomIdentifier(15))
			err := cache.Store(ctx, key, val)
			assert.NoError(err)
			state[key] = val

		case 3: // Delete
			err := cache.Delete(ctx, key)
			assert.NoError(err)
			delete(state, key)
		}
	}
}

// TestDLRU_InvalidRequests ports TestDLRU_InvalidRequests.
func TestDLRU_InvalidRequests(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	con := startConn(t, ctx, "invalid.requests.cache.dlru")
	defer con.Shutdown(ctx)
	cache := newCache(t, ctx, con)
	defer cache.Close(ctx)

	_, _, err := cache.Load(ctx, "")
	assert.Equal("missing key", err.Error())
	err = cache.Store(ctx, "", nil)
	assert.Equal("missing key", err.Error())
	err = cache.Delete(ctx, "")
	assert.Equal("missing key", err.Error())
}

// TestDLRU_OptionGetters ports TestDLRU_Options (the max-age/max-memory getters).
func TestDLRU_OptionGetters(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	con := startConn(t, ctx, "options.getters.cache.dlru")
	defer con.Shutdown(ctx)
	cache := newCache(t, ctx, con)
	defer cache.Close(ctx)

	err := cache.SetMaxAge(5 * time.Hour)
	assert.NoError(err)
	err = cache.SetMaxMemoryMB(8)
	assert.NoError(err)
	assert.Equal(5*time.Hour, cache.MaxAge())
	assert.Equal(8*1024*1024, cache.MaxMemory())
}

// TestDLRU_ComputeStampede ports the singleflight-focused subtests of TestDLRU_LoadOrCompute and
// TestDLRU_GetOrCompute that the basic Cache compute test does not cover.
func TestDLRU_ComputeStampede(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	con := startConn(t, ctx, "compute.stampede.cache.dlru")
	defer con.Shutdown(ctx)
	cache := newCache(t, ctx, con)
	defer cache.Close(ctx)

	t.Run("singleflight_dedups_concurrent_makers", func(t *testing.T) {
		assert := testarossa.For(t)
		const goroutines = 100
		var calls atomic.Int64
		release := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(goroutines)
		results := make([][]byte, goroutines)
		errs := make([]error, goroutines)
		for i := range goroutines {
			go func() {
				defer wg.Done()
				<-release
				results[i], errs[i] = cache.LoadOrCompute(ctx, "stampede-key", func(ctx context.Context) ([]byte, error) {
					calls.Add(1)
					time.Sleep(50 * time.Millisecond) // widen the singleflight window
					return []byte("once"), nil
				})
			}()
		}
		close(release)
		wg.Wait()
		assert.Equal(int64(1), calls.Load())
		for i := range goroutines {
			assert.NoError(errs[i])
			assert.Equal([]byte("once"), results[i])
		}
	})

	t.Run("validates_inputs", func(t *testing.T) {
		assert := testarossa.For(t)
		_, err := cache.LoadOrCompute(ctx, "", func(ctx context.Context) ([]byte, error) {
			return nil, nil
		})
		assert.Error(err)
		_, err = cache.LoadOrCompute(ctx, "key", nil)
		assert.Error(err)
	})

	t.Run("forwards_store_options", func(t *testing.T) {
		assert := testarossa.For(t)
		payload := []byte(utils.RandomIdentifier(4 << 10)) // 4KiB
		value, err := cache.LoadOrCompute(ctx, "compressed-key", func(ctx context.Context) ([]byte, error) {
			return payload, nil
		}, dlru.Compress(true))
		assert.NoError(err)
		assert.Equal(payload, value)

		got, ok, err := cache.Load(ctx, "compressed-key")
		assert.NoError(err)
		assert.True(ok)
		assert.Equal(payload, got)
	})
}

// TestDLRU_Soak sustains concurrent store/load/delete traffic against a stable core of replicas
// while a churner replica joins and leaves continuously. Run under -race, it stresses the membership,
// generation-gate, offload, and 404-rediscover paths for data races and deadlocks, and checks that
// goroutines do not grow unbounded after the churn stops. It is not a correctness oracle - the reference
// model check lives in FuzzDLRU_, which runs against a quiescent cluster.
func TestDLRU_Soak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in -short mode")
	}
	ctx := context.Background()
	host := "soak.cache.dlru"

	// Stable core of two replicas.
	coreCons := make([]*connector.Connector, 0, 2)
	coreCaches := make([]*dlru.Cache, 0, 2)
	for range 2 {
		con := startConn(t, ctx, host)
		coreCons = append(coreCons, con)
		coreCaches = append(coreCaches, newCache(t, ctx, con))
	}
	defer func() {
		for _, c := range coreCaches {
			c.Close(ctx)
		}
		for _, con := range coreCons {
			con.Shutdown(ctx)
		}
	}()
	converge(t, coreCaches...)

	runtime.GC()
	baseline := runtime.NumGoroutine()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var failOnce sync.Once
	var failed atomic.Bool
	fail := func(where string, err error) {
		failOnce.Do(func() {
			t.Errorf("soak %s failed: %v", where, err)
			failed.Store(true)
		})
	}
	stop := func() bool { return runCtx.Err() != nil || failed.Load() }

	var wg sync.WaitGroup

	// Operation workers.
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop() {
				cache := coreCaches[rand.IntN(len(coreCaches))]
				key := fmt.Sprintf("k%d", rand.IntN(128))
				switch op := rand.IntN(10); {
				case op < 4:
					err := cache.Store(ctx, key, []byte(utils.RandomIdentifier(16)))
					if err != nil {
						fail("store", err)
						return
					}
				case op < 7:
					_, _, err := cache.Load(ctx, key)
					if err != nil {
						fail("load", err)
						return
					}
				case op < 9:
					err := cache.Delete(ctx, key)
					if err != nil {
						fail("delete", err)
						return
					}
				default:
					err := cache.Set(ctx, key, "v")
					if err != nil {
						fail("set", err)
						return
					}
				}
			}
		}()
	}

	// Churn worker: bring a third replica in and out, forcing membership and generation changes plus
	// offload on every departure.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop() {
			con := connector.New(host)
			err := con.Startup(ctx)
			if err != nil {
				fail("churn startup", err)
				return
			}
			churner := newCache(t, ctx, con)
			time.Sleep(150 * time.Millisecond)
			err = churner.Close(ctx)
			if err != nil {
				fail("churn close", err)
				con.Shutdown(ctx)
				return
			}
			err = con.Shutdown(ctx)
			if err != nil {
				fail("churn shutdown", err)
				return
			}
		}
	}()

	wg.Wait()

	// Let the core settle back to two members and any offload goroutines drain, then check the cluster is
	// still functional and goroutines have not leaked.
	converge(t, coreCaches...)
	err := coreCaches[0].Store(ctx, "final", []byte("ok"))
	if err != nil {
		t.Fatalf("post-soak store: %v", err)
	}
	val, ok, err := coreCaches[1].Load(ctx, "final")
	if err != nil {
		t.Fatalf("post-soak load: %v", err)
	}
	if !ok || string(val) != "ok" {
		t.Fatalf("post-soak load: ok=%v val=%q", ok, val)
	}

	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	final := runtime.NumGoroutine()
	t.Logf("goroutines: baseline=%d final=%d", baseline, final)
	// Generous bound: the churner is fully closed, so goroutines should return near the baseline. A gross
	// leak (undrained offload responses, leaked ping loops) would blow well past this.
	if final > baseline+40 {
		t.Errorf("goroutine growth suggests a leak: baseline=%d final=%d", baseline, final)
	}
}

// Shared cluster for the fuzz target, built once and reused across inputs.
var (
	fuzzOnce  sync.Once
	fuzzMu    sync.Mutex
	fuzzAlpha *dlru.Cache
	fuzzBeta  *dlru.Cache
	fuzzCtx   = context.Background()
)

// FuzzDLRU_ drives randomized operation sequences over a small key space against a stable two-replica
// cluster, checking the full key space against a reference model after every operation. Because the cluster
// is quiescent (no membership churn), Cache's single-owner routing is immediately consistent, so the model
// is exact: it also asserts the single-copy invariant (each present key is held by exactly one replica).
func FuzzDLRU_(f *testing.F) {
	f.Add([]byte{0, 0, 5})
	f.Add([]byte{0, 0, 1, 1, 3, 2, 2, 1, 7})
	f.Add([]byte{0, 0, 9, 0, 3, 9, 5, 0, 0, 6, 1, 0})
	f.Add([]byte{4, 0, 0, 0, 1, 1, 2, 2, 2})

	keys := []string{"a0", "a1", "a2", "b0", "b1", "b2", "c0", "c1", "c2"}
	prefixes := []string{"a", "b", "c"}
	digits := []string{"0", "1", "2"}

	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzMu.Lock()
		defer fuzzMu.Unlock()

		fuzzOnce.Do(func() {
			host := "fuzz.cache.dlru"
			a := connector.New(host)
			err := a.Startup(fuzzCtx)
			if err != nil {
				t.Fatalf("fuzz alpha startup: %v", err)
			}
			b := connector.New(host)
			err = b.Startup(fuzzCtx)
			if err != nil {
				t.Fatalf("fuzz beta startup: %v", err)
			}
			fuzzAlpha = newCache(t, fuzzCtx, a)
			fuzzBeta = newCache(t, fuzzCtx, b)
			converge(t, fuzzAlpha, fuzzBeta)
		})

		// Isolate each input.
		err := fuzzAlpha.Clear(fuzzCtx)
		if err != nil {
			t.Fatalf("clear: %v", err)
		}
		model := map[string][]byte{}

		i := 0
		for i+1 < len(data) {
			op, sel := data[i], data[i+1]
			i += 2
			var vb byte
			if i < len(data) {
				vb = data[i]
				i++
			}
			key := keys[int(sel)%len(keys)]
			val := bytes.Repeat([]byte{vb}, 1+int(vb)%8)
			cache := fuzzAlpha
			if sel&1 == 1 {
				cache = fuzzBeta
			}

			switch op % 7 {
			case 0: // store
				err = cache.Store(fuzzCtx, key, val)
				model[key] = val
			case 1: // store compressed
				err = cache.Store(fuzzCtx, key, val, dlru.Compress(true))
				model[key] = val
			case 2: // typed set
				err = cache.Set(fuzzCtx, key, string(val))
				model[key] = val
			case 3: // delete
				err = cache.Delete(fuzzCtx, key)
				delete(model, key)
			case 4: // clear
				err = cache.Clear(fuzzCtx)
				model = map[string][]byte{}
			case 5: // delete prefix
				p := prefixes[int(sel)%len(prefixes)]
				err = cache.DeletePrefix(fuzzCtx, p)
				for k := range model {
					if strings.HasPrefix(k, p) {
						delete(model, k)
					}
				}
			case 6: // delete contains
				d := digits[int(vb)%len(digits)]
				err = cache.DeleteContains(fuzzCtx, d)
				for k := range model {
					if strings.Contains(k, d) {
						delete(model, k)
					}
				}
			}
			if err != nil {
				t.Fatalf("op %d on key %s: %v", op%7, key, err)
			}

			// Verify the full key space from both replicas against the model, plus single-copy placement.
			for _, k := range keys {
				want, present := model[k]
				for _, cc := range []*dlru.Cache{fuzzAlpha, fuzzBeta} {
					got, ok, err := cc.Load(fuzzCtx, k)
					if err != nil {
						t.Fatalf("load %s: %v", k, err)
					}
					if ok != present {
						t.Fatalf("key %s presence: got %v want %v", k, ok, present)
					}
					if present && !bytes.Equal(got, want) {
						t.Fatalf("key %s value: got %q want %q", k, got, want)
					}
				}
				copies := 0
				if _, ok := fuzzAlpha.LocalCache().Load(k); ok {
					copies++
				}
				if _, ok := fuzzBeta.LocalCache().Load(k); ok {
					copies++
				}
				exp := 0
				if present {
					exp = 1
				}
				if copies != exp {
					t.Fatalf("key %s copies: got %d want %d", k, copies, exp)
				}
			}
		}
	})
}

// BenchmarkDLRU_StoreLocal stores a key this replica owns (no round trip).
func BenchmarkDLRU_StoreLocal(b *testing.B) {
	ctx := context.Background()
	alpha := startConn(b, ctx, "bench.store.cache.dlru")
	defer alpha.Shutdown(ctx)
	beta := startConn(b, ctx, "bench.store.cache.dlru")
	defer beta.Shutdown(ctx)
	alphaCache := newCache(b, ctx, alpha)
	defer alphaCache.Close(ctx)
	betaCache := newCache(b, ctx, beta)
	defer betaCache.Close(ctx)
	converge(b, alphaCache, betaCache)

	key := keyOwnedBy(alphaCache, "local-", alpha.ID())
	b.ResetTimer()
	for b.Loop() {
		err := alphaCache.Store(ctx, key, []byte("Bar"))
		testarossa.NoError(b, err)
	}
	b.StopTimer()
}

// BenchmarkDLRU_StoreRemote stores a key owned by the peer (one unicast round trip).
func BenchmarkDLRU_StoreRemote(b *testing.B) {
	ctx := context.Background()
	alpha := startConn(b, ctx, "bench.store.cache.dlru")
	defer alpha.Shutdown(ctx)
	beta := startConn(b, ctx, "bench.store.cache.dlru")
	defer beta.Shutdown(ctx)
	alphaCache := newCache(b, ctx, alpha)
	defer alphaCache.Close(ctx)
	betaCache := newCache(b, ctx, beta)
	defer betaCache.Close(ctx)
	converge(b, alphaCache, betaCache)

	key := keyOwnedBy(alphaCache, "remote-", beta.ID())
	b.ResetTimer()
	for b.Loop() {
		err := alphaCache.Store(ctx, key, []byte("Bar"))
		testarossa.NoError(b, err)
	}
	b.StopTimer()
}

// BenchmarkDLRU_LoadLocal loads a key this replica owns (served from local memory).
func BenchmarkDLRU_LoadLocal(b *testing.B) {
	ctx := context.Background()
	alpha := startConn(b, ctx, "bench.load.cache.dlru")
	defer alpha.Shutdown(ctx)
	beta := startConn(b, ctx, "bench.load.cache.dlru")
	defer beta.Shutdown(ctx)
	alphaCache := newCache(b, ctx, alpha)
	defer alphaCache.Close(ctx)
	betaCache := newCache(b, ctx, beta)
	defer betaCache.Close(ctx)
	converge(b, alphaCache, betaCache)

	key := keyOwnedBy(alphaCache, "local-", alpha.ID())
	err := alphaCache.Store(ctx, key, []byte("Bar"))
	testarossa.NoError(b, err)
	b.ResetTimer()
	for b.Loop() {
		_, ok, err := alphaCache.Load(ctx, key)
		testarossa.NoError(b, err)
		testarossa.True(b, ok)
	}
	b.StopTimer()
}

// BenchmarkDLRU_LoadRemote loads a key owned by the peer (one unicast round trip).
func BenchmarkDLRU_LoadRemote(b *testing.B) {
	ctx := context.Background()
	alpha := startConn(b, ctx, "bench.load.cache.dlru")
	defer alpha.Shutdown(ctx)
	beta := startConn(b, ctx, "bench.load.cache.dlru")
	defer beta.Shutdown(ctx)
	alphaCache := newCache(b, ctx, alpha)
	defer alphaCache.Close(ctx)
	betaCache := newCache(b, ctx, beta)
	defer betaCache.Close(ctx)
	converge(b, alphaCache, betaCache)

	key := keyOwnedBy(alphaCache, "remote-", beta.ID())
	err := alphaCache.Store(ctx, key, []byte("Bar"))
	testarossa.NoError(b, err)
	b.ResetTimer()
	for b.Loop() {
		_, ok, err := alphaCache.Load(ctx, key)
		testarossa.NoError(b, err)
		testarossa.True(b, ok)
	}
	b.StopTimer()
}

// TestDLRU_ConcurrentStartupConvergence pins the startup-convergence contract: when replicas start
// concurrently, every replica must know the full membership set and cross-replica read-after-write must
// succeed immediately, with no settle time. It deliberately leaves the ping interval at its default (one
// minute) so periodic re-discovery cannot mask a join that raced subscription activation - the startup
// join exchange alone must converge the cluster. Regresses the bug where a join broadcast arriving before
// a peer's subscription was active left that peer unaware of the joiner (and routing keys to the wrong
// owner) until the next ping.
func TestDLRU_ConcurrentStartupConvergence(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	const replicas = 3
	for trial := range 8 {
		host := fmt.Sprintf("converge%d.cache.dlru", trial)
		cons := make([]*connector.Connector, replicas)
		for i := range cons {
			cons[i] = startConn(t, ctx, host)
		}

		// Create the caches concurrently so join broadcasts race subscription activation. Default ping
		// interval - no re-discovery to paper over a missed join.
		caches := make([]*dlru.Cache, replicas)
		errs := make([]error, replicas)
		var wg sync.WaitGroup
		for i := range caches {
			wg.Add(1)
			go func() {
				defer wg.Done()
				caches[i], errs[i] = dlru.NewCache(ctx, cons[i], ":444/testcache")
			}()
		}
		wg.Wait()
		for i := range errs {
			assert.NoError(errs[i])
		}

		// Every replica must see the full set as soon as construction returns.
		full := make([]string, replicas)
		for i := range cons {
			full[i] = cons[i].ID()
		}
		slices.Sort(full)
		for i, c := range caches {
			assert.Equal(full, c.Members(), "trial %d replica %d did not converge", trial, i)
		}

		// Cross-replica read-after-write must succeed with no settle time: store via one replica, load via
		// the next. A membership disagreement would route these to different owners and miss.
		for k := 0; k < 4*replicas; k++ {
			key := fmt.Sprintf("k%d", k)
			val := []byte(fmt.Sprintf("v%d", k))
			err := caches[k%replicas].Store(ctx, key, val)
			assert.NoError(err)
			got, ok, err := caches[(k+1)%replicas].Load(ctx, key)
			assert.NoError(err)
			assert.True(ok, "trial %d key %s missing after cross-replica store", trial, key)
			assert.Equal(val, got)
		}

		for _, c := range caches {
			c.Close(ctx)
		}
		for _, con := range cons {
			con.Shutdown(ctx)
		}
	}
}

// TestDLRU_ShedInFlight freezes a join-triggered shed at the exact moment before a displaced key is
// soft-stored to its new owner and, in that window, exercises the two guarantees that make an in-flight
// shed safe: a Load for the moving key still resolves (previous-generation retry falls back to the old
// owner, which is still serving and still holds the value), and an upstream write to the new owner is not
// clobbered by the offloaded value (soft store is store-if-absent). A checkpoint makes the window
// deterministic instead of relying on timing.
func TestDLRU_ShedInFlight(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("shedinflight.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	// alpha is the sole owner: store a spread of keys as V1, all landing on alpha.
	var keys []string
	for i := 0; i < 16; i++ {
		keys = append(keys, "key"+strconv.Itoa(i))
	}
	for _, k := range keys {
		err = alphaCache.Store(ctx, k, []byte("V1"))
		assert.NoError(err)
	}

	// Freeze alpha's join-triggered shed just before the first displaced key is soft-stored. handleJoin
	// has already added beta and computed the offload set, but nothing has been deleted, so every
	// displaced key is still on alpha.
	alphaCache.Seams().Break(dlru.CheckpointOffloadBeforeStore)

	beta := connector.New("shedinflight.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	var betaCache *dlru.Cache
	done := make(chan struct{})
	go func() {
		defer close(done)
		// broadcastJoin blocks on alpha's frozen handleJoin response until the shed is resumed.
		betaCache, _ = dlru.NewCache(ctx, beta, ":444/test")
	}()

	// Wait until alpha is frozen mid-shed.
	alphaCache.Seams().Wait(dlru.CheckpointOffloadBeforeStore)

	// alpha has added beta by now. Pick a key beta owns that is still physically on alpha.
	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	assert.Equal(both, alphaCache.Members())
	var moved string
	for _, k := range keys {
		if alphaCache.OwnerOf(k) == beta.ID() {
			moved = k
			break
		}
	}
	assert.NotEqual("", moved)

	// Liveness during shed: a Load for the moving key still resolves. The new owner (beta) misses, and the
	// previous-generation retry falls back to alpha, which is still serving and still holds V1.
	val, ok, err := alphaCache.Load(ctx, moved)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("V1", string(val))

	// No clobber during shed: an upstream write of V2, routed to the new owner beta while the shed is
	// frozen, must survive the offload's soft store.
	err = alphaCache.Store(ctx, moved, []byte("V2"))
	assert.NoError(err)

	// Release the shed and let beta finish joining.
	alphaCache.Seams().Resume(dlru.CheckpointOffloadBeforeStore)
	<-done
	assert.NotNil(betaCache)
	defer betaCache.Close(ctx)

	// The upstream V2 won; the soft-stored V1 did not clobber it.
	got, ok, err := betaCache.Load(ctx, moved)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("V2", string(got))
}

// TestDLRU_StoreGateDuringTopologyChange freezes a Store after it has chosen its owner and stamped the
// current generation onto the request, then changes the topology (a third replica joins, bumping the
// generation) before releasing it. The store carries the now-previous generation to its owner, which
// still accepts it within the overlap window (previous-generation acceptance), so the value lands rather
// than being dropped. This exercises the generation gate against a real concurrent membership change,
// deterministically, rather than the forged staleGen fault.
func TestDLRU_StoreGateDuringTopologyChange(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := context.Background()

	alpha := connector.New("gatetopo.dlru")
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	alphaCache, err := dlru.NewCache(ctx, alpha, ":444/test")
	assert.NoError(err)
	defer alphaCache.Close(ctx)

	beta := connector.New("gatetopo.dlru")
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)
	betaCache, err := dlru.NewCache(ctx, beta, ":444/test")
	assert.NoError(err)
	defer betaCache.Close(ctx)

	both := []string{alpha.ID(), beta.ID()}
	slices.Sort(both)
	converged := eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), both) && slices.Equal(betaCache.Members(), both)
	})
	assert.True(converged)

	// A key beta owns, so alpha routes the store remotely and stamps the current generation.
	var key string
	for i := 0; ; i++ {
		k := "k" + strconv.Itoa(i)
		if alphaCache.OwnerOf(k) == beta.ID() {
			key = k
			break
		}
	}

	// Freeze the store after owner and generation are stamped, before the request is sent.
	alphaCache.Seams().Break(dlru.CheckpointBeforeSend)
	var storeErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		storeErr = alphaCache.Store(ctx, key, []byte("V"))
	}()
	alphaCache.Seams().Wait(dlru.CheckpointBeforeSend)

	// Change the topology underneath the in-flight store: a third replica joins, bumping the generation
	// on alpha and beta (their message handlers process the join independently of the frozen store).
	gamma := connector.New("gatetopo.dlru")
	err = gamma.Startup(ctx)
	assert.NoError(err)
	defer gamma.Shutdown(ctx)
	gammaCache, err := dlru.NewCache(ctx, gamma, ":444/test")
	assert.NoError(err)
	defer gammaCache.Close(ctx)

	allThree := []string{alpha.ID(), beta.ID(), gamma.ID()}
	slices.Sort(allThree)
	converged = eventually(2*time.Second, func() bool {
		return slices.Equal(alphaCache.Members(), allThree) && slices.Equal(betaCache.Members(), allThree)
	})
	assert.True(converged)

	// Release the store. It carries the previous generation to beta, which still accepts it within the
	// overlap window.
	alphaCache.Seams().Resume(dlru.CheckpointBeforeSend)
	<-done
	assert.NoError(storeErr)

	// The value landed and is retrievable (from beta directly, or via a previous-generation retry if
	// ownership moved to gamma under the new generation).
	val, ok, err := betaCache.Load(ctx, key)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("V", string(val))
}
