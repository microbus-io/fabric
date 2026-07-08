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

package connector

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"
)

// TestConnector_DroppedAckUnicast arms the dropAck fault so a served request is never acknowledged.
// The unicast caller must fast-fail at the ack timeout with a 404 - well before the still-running
// handler completes - and leave no entry lingering in the reqs map.
func TestConnector_DroppedAckUnicast(t *testing.T) {
	// No parallel - Time sensitive
	assert := testarossa.For(t)

	ctx := t.Context()

	// Handler outlives the ack timeout, so a 404 within the ack window can only come from the dropped ack,
	// not from a response racing in ahead of it.
	con := New("dropped.ack.unicast.connector")
	con.Subscribe("Ack",
		func(w http.ResponseWriter, r *http.Request) error {
			time.Sleep(con.ackTimeout * 4)
			return nil
		},
		sub.At("GET", "ack"),
		sub.Web(),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	con.seams.Inject(faultDropAck, "Ack")

	t0 := time.Now()
	_, err = con.Request(ctx, pub.GET("https://dropped.ack.unicast.connector/ack"))
	dur := time.Since(t0)

	assert.Error(err)
	assert.Equal(http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.True(dur >= con.ackTimeout && dur < con.ackTimeout+time.Second, dur)
	// The await channel is torn down when the caller gives up.
	assert.Zero(con.reqs.Len())
}

// TestConnector_DroppedResponse arms the dropResponse fault so a served request is acknowledged
// but its response is never sent. The caller does not fast-fail at the ack timeout (an ack arrived);
// it waits out the full time budget and returns a 408, tearing down its await channel.
func TestConnector_DroppedResponse(t *testing.T) {
	// No parallel - Time sensitive
	assert := testarossa.For(t)

	ctx := t.Context()

	con := New("dropped.response.connector")
	con.Subscribe("Resp",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "resp"),
		sub.Web(),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	con.seams.Inject(faultDropResponse, "Resp")

	budget := 500 * time.Millisecond
	t0 := time.Now()
	_, err = con.Request(ctx,
		pub.GET("https://dropped.response.connector/resp"),
		pub.Timeout(budget),
	)
	dur := time.Since(t0)

	assert.Error(err)
	assert.Equal(http.StatusRequestTimeout, errors.Convert(err).StatusCode)
	// The caller waited the budget, not the shorter ack timeout.
	assert.True(dur >= budget && dur < budget+time.Second, dur)
	assert.Zero(con.reqs.Len())
}

// TestConnector_DroppedAckMulticast arms the dropAck fault so a served request is never acknowledged.
// A multicast caller, seeing neither an ack nor a response within the ack window, must return cleanly
// with zero responses and drop the subject from its known-responders cache.
func TestConnector_DroppedAckMulticast(t *testing.T) {
	// No parallel - Time sensitive
	assert := testarossa.For(t)

	ctx := t.Context()

	// A distinct client and server exercise the real bus path (a single self-looping connector would
	// short-circuit locally and bypass the ack/known-responders logic under test).
	// The handler sleeps only once armed, so in the faulted round no response can race in ahead of the
	// ack timer and register the responder.
	var slow atomic.Bool
	server := New("dropped.ack.multicast.server")
	server.Subscribe("Cast",
		func(w http.ResponseWriter, r *http.Request) error {
			if slow.Load() {
				time.Sleep(server.ackTimeout * 4)
			}
			w.Write([]byte("ok"))
			return nil
		},
		sub.At("GET", "cast"),
		sub.Web(),
	)
	client := New("dropped.ack.multicast.client")

	err := server.Startup(ctx)
	assert.NoError(err)
	defer server.Shutdown(ctx)
	err = client.Startup(ctx)
	assert.NoError(err)
	defer client.Shutdown(ctx)

	// A normal multicast populates the client's known-responders cache for the subject.
	count := 0
	for e := range client.Publish(ctx, pub.GET("https://dropped.ack.multicast.server/cast"), pub.Multicast()) {
		_, err := e.Get()
		assert.NoError(err)
		count++
	}
	assert.Equal(1, count)
	assert.True(client.knownResponders.Len() >= 1)

	// Drop the ack on the server and slow its handler so the faulted round sees zero acks, zero responses.
	slow.Store(true)
	server.seams.Inject(faultDropAck, "Cast")

	// Baseline includes the subject's entry plus incidental control-plane entries (e.g. on-new-subs).
	krBefore := client.knownResponders.Len()
	t0 := time.Now()
	count = 0
	for e := range client.Publish(ctx, pub.GET("https://dropped.ack.multicast.server/cast"), pub.Multicast()) {
		_, err := e.Get()
		assert.NoError(err)
		count++
	}
	dur := time.Since(t0)

	// Zero responders, and the subject's known-responders entry was cleared so the next multicast re-discovers.
	assert.Equal(0, count)
	assert.True(dur >= client.ackTimeout && dur < client.ackTimeout+time.Second, dur)
	assert.True(client.knownResponders.Len() < krBefore, client.knownResponders.Len())
	assert.Zero(client.reqs.Len())
}

// TestConnector_DuplicateResponse arms the duplicateResponse fault so each received response is
// re-injected many times over, well past the await channel's capacity. The caller must still get
// exactly one result, the excess copies must be absorbed by the overflow-goroutine path without
// wedging anything, and the request machinery must remain usable for a follow-up call.
func TestConnector_DuplicateResponse(t *testing.T) {
	// No parallel - Time sensitive
	assert := testarossa.For(t)

	ctx := t.Context()

	con := New("duplicate.response.connector")
	con.Subscribe("Echo",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("ok"))
			return nil
		},
		sub.At("GET", "echo"),
		sub.Web(),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Re-inject each response well past multicastChanCap so the overflow-goroutine path is taken.
	con.seams.InjectN(con.multicastChanCap*2, faultDuplicateResponse, con.hostname)

	res, err := con.Request(ctx, pub.GET("https://duplicate.response.connector/echo"))
	assert.NoError(err)
	if assert.NotNil(res) {
		assert.Equal(http.StatusOK, res.StatusCode)
	}

	// The await channel is torn down once the caller has its single result; the overflow goroutines
	// drain against the closed Done channel rather than blocking forever.
	drained := false
	for range 100 {
		if con.reqs.Len() == 0 {
			drained = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(drained)

	// The connector is not wedged: a follow-up request still round-trips.
	con.seams.Withdraw(faultDuplicateResponse, con.hostname)
	_, err = con.Request(ctx, pub.GET("https://duplicate.response.connector/echo"))
	assert.NoError(err)
}

// TestConnector_JWKSFetchErrRotation arms one JWKS fetch failure (the key-rotation-boundary case)
// and presents a valid token whose kid is not yet cached. The first verification fails 401, but the
// failed fetch must not poison the 1s cooldown: an immediate retry re-fetches, caches the key, and
// succeeds. Pre-N1-fix the second call would 401 for up to a second on the poisoned cooldown.
func TestConnector_JWKSFetchErrRotation(t *testing.T) {
	// No parallel - Time sensitive
	assert := testarossa.For(t)

	ctx := t.Context()

	type JWK struct {
		KTY string `json:"kty"`
		CRV string `json:"crv"`
		X   string `json:"x"`
		KID string `json:"kid"`
	}

	pub25519, priv25519, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	hash := sha256.Sum256(pub25519)
	kid := base64.RawURLEncoding.EncodeToString(hash[:8])
	signToken := func(claims jwt.MapClaims) (string, error) {
		claims["iss"] = "https://access.token.core"
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		token.Header["kid"] = kid
		return token.SignedString(priv25519)
	}

	issuer := New("access.token.core")
	issuer.Subscribe("JWKS",
		func(w http.ResponseWriter, r *http.Request) error {
			jwks := struct {
				Keys []JWK `json:"keys"`
			}{}
			jwks.Keys = append(jwks.Keys, JWK{
				KTY: "OKP",
				CRV: "Ed25519",
				X:   base64.RawURLEncoding.EncodeToString(pub25519),
				KID: kid,
			})
			w.Header().Set("Content-Type", "application/json")
			return json.NewEncoder(w).Encode(jwks)
		},
		sub.At("GET", ":888/jwks"),
		sub.Web(),
	)

	con := New("jwks.fetch.err.connector")
	con.Subscribe("Gated",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "gated"),
		sub.Web(),
		sub.RequiredClaims(`roles.student`),
	)

	err = issuer.Startup(ctx)
	assert.NoError(err)
	defer issuer.Shutdown(ctx)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	token, err := signToken(jwt.MapClaims{
		"sub":   "harry@hogwarts.edu",
		"roles": []string{"student"},
	})
	assert.NoError(err)

	// Fail exactly the first JWKS fetch, simulating the token service being briefly unreachable at
	// the rotation boundary when the kid is not yet cached.
	con.seams.Inject(faultJWKSFetchErr, "access.token.core")

	// First attempt: the fetch fails, so the key is never found and the token is rejected 401.
	_, err = con.Request(ctx, pub.GET("https://jwks.fetch.err.connector/gated"), pub.Token(token))
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))

	// Immediate retry (well within the 1s cooldown): a poisoned cooldown would short-circuit the
	// fetch and 401 again. With N1 fixed the cooldown was never set, so this re-fetches and accepts.
	_, err = con.Request(ctx, pub.GET("https://jwks.fetch.err.connector/gated"), pub.Token(token))
	assert.NoError(err)
}

// returnedEarly reports whether a request goroutine has already returned - used to assert that a
// request is still in flight while frozen at a checkpoint.
func returnedEarly(done <-chan error) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// TestConnector_CheckpointReqRegistered breaks at reqRegistered on the caller and asserts the await
// channel is registered but the request has not yet been published - the caller is frozen between
// registration and send. Resuming lets it complete and drains the reqs map. Exercises the Break /
// Wait / Resume half of the seam on the client path.
func TestConnector_CheckpointReqRegistered(t *testing.T) {
	// No parallel - drives a breakpoint
	assert := testarossa.For(t)

	ctx := t.Context()

	server := New("checkpoint.reqreg.server")
	server.Subscribe("Echo",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("ok"))
			return nil
		},
		sub.At("GET", "echo"),
		sub.Web(),
	)
	client := New("checkpoint.reqreg.client")

	err := server.Startup(ctx)
	assert.NoError(err)
	defer server.Shutdown(ctx)
	err = client.Startup(ctx)
	assert.NoError(err)
	defer client.Shutdown(ctx)

	// A warmup request flushes any residual startup control traffic so the breakpoint below catches
	// only the measured request.
	_, err = client.Request(ctx, pub.GET("https://checkpoint.reqreg.server/echo"))
	assert.NoError(err)

	client.seams.Break(checkpointReqRegistered)
	done := make(chan error, 1)
	go func() {
		_, e := client.Request(ctx, pub.GET("https://checkpoint.reqreg.server/echo"))
		done <- e
	}()
	client.seams.Wait(checkpointReqRegistered)

	// Registered, but nothing sent yet, so the request is still in flight.
	assert.Equal(1, client.reqs.Len())
	assert.False(returnedEarly(done))

	client.seams.Resume(checkpointReqRegistered)
	err = <-done
	assert.NoError(err)
	assert.Zero(client.reqs.Len())
}

// TestConnector_CheckpointAfterAck breaks at afterAck on the server and asserts the ack has been sent
// while the handler has not yet run - the request is frozen between acknowledgement and dispatch.
// Resuming runs the handler and completes the call. Pins the ack-before-handler ordering.
func TestConnector_CheckpointAfterAck(t *testing.T) {
	// No parallel - drives a breakpoint
	assert := testarossa.For(t)

	ctx := t.Context()

	var handlerEntered atomic.Bool
	server := New("checkpoint.afterack.server")
	server.Subscribe("Echo",
		func(w http.ResponseWriter, r *http.Request) error {
			handlerEntered.Store(true)
			w.Write([]byte("ok"))
			return nil
		},
		sub.At("GET", "echo"),
		sub.Web(),
	)
	client := New("checkpoint.afterack.client")

	err := server.Startup(ctx)
	assert.NoError(err)
	defer server.Shutdown(ctx)
	err = client.Startup(ctx)
	assert.NoError(err)
	defer client.Shutdown(ctx)

	// Warmup flushes residual control traffic so the breakpoint catches only the measured request.
	_, err = client.Request(ctx, pub.GET("https://checkpoint.afterack.server/echo"))
	assert.NoError(err)
	handlerEntered.Store(false)

	server.seams.Break(checkpointAfterAck)
	done := make(chan error, 1)
	go func() {
		_, e := client.Request(ctx, pub.GET("https://checkpoint.afterack.server/echo"))
		done <- e
	}()
	server.seams.Wait(checkpointAfterAck)

	// The ack has gone out, but the handler goroutine has not been spawned yet.
	assert.False(handlerEntered.Load())
	assert.False(returnedEarly(done))

	server.seams.Resume(checkpointAfterAck)
	err = <-done
	assert.NoError(err)
	assert.True(handlerEntered.Load())
}

// TestConnector_CheckpointBeforeResponseSend breaks at beforeResponseSend on the server and asserts
// the handler ran to completion while its response has not yet been published - the caller is still
// pending. Resuming delivers the response and completes the call. Pins the handler-then-send ordering.
func TestConnector_CheckpointBeforeResponseSend(t *testing.T) {
	// No parallel - drives a breakpoint
	assert := testarossa.For(t)

	ctx := t.Context()

	var handlerReturned atomic.Bool
	server := New("checkpoint.beforeresp.server")
	server.Subscribe("Echo",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("ok"))
			handlerReturned.Store(true)
			return nil
		},
		sub.At("GET", "echo"),
		sub.Web(),
	)
	client := New("checkpoint.beforeresp.client")

	err := server.Startup(ctx)
	assert.NoError(err)
	defer server.Shutdown(ctx)
	err = client.Startup(ctx)
	assert.NoError(err)
	defer client.Shutdown(ctx)

	// Warmup flushes residual control traffic so the breakpoint catches only the measured request.
	_, err = client.Request(ctx, pub.GET("https://checkpoint.beforeresp.server/echo"))
	assert.NoError(err)
	handlerReturned.Store(false)

	server.seams.Break(checkpointBeforeResponseSend)
	done := make(chan error, 1)
	go func() {
		_, e := client.Request(ctx, pub.GET("https://checkpoint.beforeresp.server/echo"))
		done <- e
	}()
	server.seams.Wait(checkpointBeforeResponseSend)

	// The handler finished, but its response has not been published, so the caller is still pending.
	assert.True(handlerReturned.Load())
	assert.Equal(1, client.reqs.Len())
	assert.False(returnedEarly(done))

	server.seams.Resume(checkpointBeforeResponseSend)
	err = <-done
	assert.NoError(err)
	assert.Zero(client.reqs.Len())
}
