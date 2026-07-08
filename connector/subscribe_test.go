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
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/transport"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"
)

func TestConnector_DirectorySubscription(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	var count int
	var appendix string
	con := New("directory.subscription.connector")
	con.Subscribe("Directory",
		func(w http.ResponseWriter, r *http.Request) error {
			count++
			_, appendix, _ = strings.Cut(r.URL.Path, "/directory/")
			return nil
		},
		sub.At("GET", "directory/{appendix...}"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages to various locations under the directory
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/1.html")
	assert.NoError(err)
	assert.Equal("1.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/2.html")
	assert.NoError(err)
	assert.Equal("2.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/sub/3.html")
	assert.NoError(err)
	assert.Equal("sub/3.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/")
	assert.NoError(err)
	assert.Equal("", appendix)

	assert.Equal(4, count)

	// The path of the directory should not be captured
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory")
	assert.Error(err)
}

func TestConnector_HyphenInHostname(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice. Hyphens are valid in service identity hostnames;
	// underscores are reserved for the flat-host encoding (period-replacement)
	// and are no longer allowed in raw hostnames.
	entered := false
	con := New("hyphen-in-host-name.connector")
	con.Subscribe("Path",
		func(w http.ResponseWriter, r *http.Request) error {
			entered = true
			return nil
		},
		sub.At("GET", "path"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	_, err = con.GET(ctx, "https://hyphen-in-host-name.connector/path")
	assert.NoError(err)
	assert.True(entered)
}

func TestConnector_PathArgumentsInSubscription(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	alphaCount := 0
	betaCount := 0
	rootCount := 0
	parentCount := 0
	detected := map[string]string{}
	con := New("path.arguments.in.subscription.connector")
	con.Subscribe("Alpha",
		func(w http.ResponseWriter, r *http.Request) error {
			alphaCount++
			parts := strings.Split(r.URL.Path, "/")
			detected[r.URL.Path] = parts[2]
			return nil
		},
		sub.At("GET", "/obj/{id}/alpha"),
		sub.Web(),
	)
	con.Subscribe("Beta",
		func(w http.ResponseWriter, r *http.Request) error {
			betaCount++
			parts := strings.Split(r.URL.Path, "/")
			detected[r.URL.Path] = parts[2]
			return nil
		},
		sub.At("GET", "/obj/{id}/beta"),
		sub.Web(),
	)
	con.Subscribe("ObjID",
		func(w http.ResponseWriter, r *http.Request) error {
			rootCount++
			parts := strings.Split(r.URL.Path, "/")
			detected[r.URL.Path] = parts[2]
			return nil
		},
		sub.At("GET", "/obj/{id}"),
		sub.Web(),
	)
	con.Subscribe("Obj",
		func(w http.ResponseWriter, r *http.Request) error {
			parentCount++
			detected[r.URL.Path] = ""
			return nil
		},
		sub.At("GET", "/obj"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1234/alpha")
	assert.NoError(err)
	assert.Equal(1, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2345/alpha")
	assert.NoError(err)
	assert.Equal(2, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1111/beta")
	assert.NoError(err)
	assert.Equal(1, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2222/beta")
	assert.NoError(err)
	assert.Equal(2, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/8000")
	assert.NoError(err)
	assert.Equal(1, rootCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj")
	assert.NoError(err)
	assert.Equal(1, parentCount)

	assert.Len(detected, 6)
	assert.Equal("1234", detected["/obj/1234/alpha"])
	assert.Equal("2345", detected["/obj/2345/alpha"])
	assert.Equal("1111", detected["/obj/1111/beta"])
	assert.Equal("2222", detected["/obj/2222/beta"])
	assert.Equal("8000", detected["/obj/8000"])
	assert.Equal("", detected["/obj"])
}

func TestConnector_MixedAsteriskSubscription(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	detected := map[string]bool{}
	con := New("mixed.asterisk.subscription.connector")
	con.Subscribe("Gamma",
		func(w http.ResponseWriter, r *http.Request) error {
			detected[r.URL.Path] = true
			return nil
		},
		sub.At("GET", "/obj/x*x/gamma"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/2222/gamma")
	assert.Error(err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x2x/gamma")
	assert.Error(err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x*x/gamma")
	assert.NoError(err)
}

func TestConnector_ErrorAndPanic(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	con := New("error.and.panic.connector")
	con.Subscribe("UserErr",
		func(w http.ResponseWriter, r *http.Request) error {
			return errors.New("bad input", http.StatusBadRequest)
		},
		sub.At("GET", "usererr"),
		sub.Web(),
	)
	con.Subscribe("Err",
		func(w http.ResponseWriter, r *http.Request) error {
			return errors.New("it's bad")
		},
		sub.At("GET", "err"),
		sub.Web(),
	)
	con.Subscribe("Panic",
		func(w http.ResponseWriter, r *http.Request) error {
			panic("it's really bad")
		},
		sub.At("GET", "panic"),
		sub.Web(),
	)
	con.Subscribe("OsErr",
		func(w http.ResponseWriter, r *http.Request) error {
			err := errors.Trace(os.ErrNotExist)
			assert.True(errors.Is(err, os.ErrNotExist))
			return err
		},
		sub.At("GET", "oserr"),
		sub.Web(),
	)
	con.Subscribe("StillAlive",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "stillalive"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages
	_, err = con.GET(ctx, "https://error.and.panic.connector/usererr")
	assert.Error(err)
	assert.Equal("bad input", err.Error())
	assert.Equal(http.StatusBadRequest, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/err")
	assert.Error(err)
	assert.Equal("it's bad", err.Error())
	assert.Equal(http.StatusInternalServerError, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/panic")
	assert.Error(err)
	assert.Equal("it's really bad", err.Error())

	_, err = con.GET(ctx, "https://error.and.panic.connector/oserr")
	assert.Error(err)
	assert.Equal("file does not exist", err.Error())
	assert.False(errors.Is(err, os.ErrNotExist)) // Cannot reconstitute error type

	_, err = con.GET(ctx, "https://error.and.panic.connector/stillalive")
	assert.NoError(err)
}

func TestConnector_DifferentPlanes(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservices
	alpha := New("different.planes.connector")
	alpha.SetPlane("alpha")
	alpha.Subscribe("ID",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("alpha"))
			return nil
		},
		sub.At("GET", "id"),
		sub.Web(),
	)

	beta := New("different.planes.connector")
	beta.SetPlane("beta")
	beta.Subscribe("ID",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("beta"))
			return nil
		},
		sub.At("GET", "id"),
		sub.Web(),
	)

	// Startup the microservices
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)

	// Alpha should never see beta
	for range 32 {
		response, err := alpha.GET(ctx, "https://different.planes.connector/id")
		if assert.NoError(err) {
			body, err := io.ReadAll(response.Body)
			if assert.NoError(err) {
				assert.Equal("alpha", string(body))
			}
		}
	}

	// Beta should never see alpha
	for range 32 {
		response, err := beta.GET(ctx, "https://different.planes.connector/id")
		if assert.NoError(err) {
			body, err := io.ReadAll(response.Body)
			if assert.NoError(err) {
				assert.Equal("beta", string(body))
			}
		}
	}
}

func TestConnector_SubscribeBeforeAndAfterStartup(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservices
	var beforeCalled, afterCalled bool
	con := New("subscribe.before.and.after.startup.connector")

	// Subscribe before beta is started
	con.Subscribe("Before",
		func(w http.ResponseWriter, r *http.Request) error {
			beforeCalled = true
			return nil
		},
		sub.At("GET", "before"),
		sub.Web(),
	)

	// Startup the microservice
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Subscribe after beta is started
	con.Subscribe("After",
		func(w http.ResponseWriter, r *http.Request) error {
			afterCalled = true
			return nil
		},
		sub.At("GET", "after"),
		sub.Web(),
	)

	// Send requests to both handlers
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/before")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/after")
	assert.NoError(err)

	assert.True(beforeCalled)
	assert.True(afterCalled)
}

func TestConnector_Unsubscribe(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	con := New("unsubscribe.connector")

	// Subscribe
	con.Subscribe("Sub1",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "sub1"),
		sub.Web(),
	)
	unsub1 := func() error { return con.Unsubscribe("Sub1") }
	con.Subscribe("Sub2",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "sub2"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.NoError(err)

	// Unsubscribe sub1
	err = unsub1()
	assert.NoError(err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.Error(err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.NoError(err)

	// Deactivate all
	err = con.deactivateSubs()
	assert.NoError(err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.Error(err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.Error(err)
}

func TestConnector_AnotherHost(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservices
	alpha := New("alpha.another.host.connector")
	alpha.Subscribe("Empty",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "https://alternative.host.connector/empty"),
		sub.Web(),
	)

	beta1 := New("beta.another.host.connector")
	beta1.Subscribe("Empty",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "https://alternative.host.connector/empty"),
		sub.Web(),
	)

	beta2 := New("beta.another.host.connector")
	beta2.Subscribe("Empty",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", "https://alternative.host.connector/empty"),
		sub.Web(),
	)

	// Startup the microservices
	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	err = beta1.Startup(ctx)
	assert.NoError(err)
	defer beta1.Shutdown(ctx)
	err = beta2.Startup(ctx)
	assert.NoError(err)
	defer beta2.Shutdown(ctx)

	// Send message
	responded := 0
	ch := alpha.Publish(ctx, pub.GET("https://alternative.host.connector/empty"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		assert.NoError(err)
		responded++
	}
	// Even though the microservices subscribe to the same alternative host, their queues should be different
	assert.Equal(2, responded)
}

func TestConnector_DirectAddressing(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	con := New("direct.addressing.connector")
	con.Subscribe("Hello",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("Hello"))
			return nil
		},
		sub.At("GET", "/hello"),
		sub.Web(),
	)
	unsub := func() error { return con.Unsubscribe("Hello") }

	// Startup the microservice
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	assert.NoError(err)

	err = unsub()
	assert.NoError(err)

	// Both subscriptions should be deactivated
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	assert.Error(err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	assert.Error(err)
}

func TestConnector_SubPendingOps(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("sub.pending.ops.connector")

	start := make(chan bool)
	hold := make(chan bool)
	end := make(chan bool)
	con.Subscribe("Op",
		func(w http.ResponseWriter, r *http.Request) error {
			start <- true
			hold <- true
			return nil
		},
		sub.At("GET", "/op"),
		sub.Web(),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Zero(con.pendingOps.Load())

	// First call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	assert.Equal(int32(1), con.pendingOps.Load())

	// Second call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	assert.Equal(int32(2), con.pendingOps.Load())

	// The handler goroutine's deferred c.pendingOps.Add(-1) runs after the response is
	// published on the transport, so under heavy parallel test load the caller's GET can
	// return (and unblock end) before the decrement is observed. Poll briefly.
	waitForPending := func(want int32) {
		deadline := time.Now().Add(2 * time.Second)
		for con.pendingOps.Load() != want && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
	}
	<-hold
	<-end
	waitForPending(1)
	assert.Equal(int32(1), con.pendingOps.Load())
	<-hold
	<-end
	waitForPending(0)
	assert.Zero(con.pendingOps.Load())
}

func TestConnector_SubscriptionMethods(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	var get int
	var post int
	var star int
	con := New("subscription.methods.connector")
	con.Subscribe("SingleGet",
		func(w http.ResponseWriter, r *http.Request) error {
			get++
			return nil
		},
		sub.At("GET", "single"),
		sub.Web(),
	)
	con.Subscribe("SinglePost",
		func(w http.ResponseWriter, r *http.Request) error {
			post++
			return nil
		},
		sub.At("POST", "single"),
		sub.Web(),
	)
	con.Subscribe("Star",
		func(w http.ResponseWriter, r *http.Request) error {
			star++
			return nil
		},
		sub.At("ANY", "star"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/single"))
	assert.NoError(err)
	assert.Equal(1, get)
	assert.Zero(post)

	_, err = con.Request(ctx, pub.POST("https://subscription.methods.connector/single"))
	assert.NoError(err)
	assert.Equal(1, get)
	assert.Equal(1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/single"))
	assert.Error(err)
	assert.Equal(http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.Equal(1, get)
	assert.Equal(1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/star"))
	assert.NoError(err)
	assert.Equal(1, get)
	assert.Equal(1, post)
	assert.Equal(1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/star"))
	assert.NoError(err)
	assert.Equal(1, get)
	assert.Equal(1, post)
	assert.Equal(2, star)
}

func TestConnector_SubscriptionPorts(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	var p123 int
	var p234 int
	var star int
	con := New("subscription.ports.connector")
	con.Subscribe("Single123",
		func(w http.ResponseWriter, r *http.Request) error {
			p123++
			return nil
		},
		sub.At("GET", ":123/single"),
		sub.Web(),
	)
	con.Subscribe("Single234",
		func(w http.ResponseWriter, r *http.Request) error {
			p234++
			return nil
		},
		sub.At("GET", ":234/single"),
		sub.Web(),
	)
	con.Subscribe("Any",
		func(w http.ResponseWriter, r *http.Request) error {
			star++
			return nil
		},
		sub.At("GET", ":0/any"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:123/single"))
	assert.NoError(err)
	assert.Equal(1, p123)
	assert.Zero(p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:234/single"))
	assert.NoError(err)
	assert.Equal(1, p123)
	assert.Equal(1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/single"))
	assert.Error(err)
	assert.Equal(http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.Equal(1, p123)
	assert.Equal(1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/any"))
	assert.NoError(err)
	assert.Equal(1, p123)
	assert.Equal(1, p234)
	assert.Equal(1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:10000/any"))
	assert.NoError(err)
	assert.Equal(1, p123)
	assert.Equal(1, p234)
	assert.Equal(2, star)
}

func TestConnector_FrameConsistency(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	con := New("frame.consistency.connector")
	con.Subscribe("Frame",
		func(w http.ResponseWriter, r *http.Request) error {
			f1 := frame.Of(r)
			f2 := frame.Of(r.Context())
			assert.Equal(&f1, &f2)
			f1.Set("ABC", "abc")
			assert.Equal(&f1, &f2)
			assert.Equal("abc", f2.Get("ABC"))
			f2.Set("ABC", "")
			assert.Equal(&f1, &f2)
			assert.Equal("", f1.Get("ABC"))
			return nil
		},
		sub.At("GET", "/frame"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://frame.consistency.connector/frame"))
	assert.NoError(err)
}

// TestConnector_VerifiedSourceOverwrite verifies that the receiver-side
// connector overwrites the Microbus-From-Host header with the verified source
// from the subject's <from_host_flat> segment, regardless of any value the
// publisher set on the header. The receiver also exposes the verified source
// via VerifiedSource(ctx).
func TestConnector_VerifiedSourceOverwrite(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	var observedFromHost string
	receiver := New("receiver.verified.source.connector")
	receiver.Subscribe("Echo",
		func(w http.ResponseWriter, r *http.Request) error {
			observedFromHost = frame.Of(r).FromHost()
			return nil
		},
		sub.At("GET", "echo"),
		sub.Web(),
	)

	sender := New("sender.verified.source.connector")
	err := sender.Startup(ctx)
	assert.NoError(err)
	defer sender.Shutdown(ctx)
	err = receiver.Startup(ctx)
	assert.NoError(err)
	defer receiver.Shutdown(ctx)

	// Honest send - From-Host header is set by the framework to the sender's
	// hostname. Receiver should observe that same hostname.
	_, err = sender.Request(ctx, pub.GET("https://receiver.verified.source.connector/echo"))
	assert.NoError(err)
	assert.Equal("sender.verified.source.connector", observedFromHost)

	// Spoof attempt - set a bogus From-Host header on the outbound request.
	// In production this would also fail at the NATS PUB ACL boundary, but
	// even on the short-circuit transport (no NATS in tests) the receiver
	// must overwrite the header with the verified source from the subject,
	// which the sender's connector built from its own hostname.
	_, err = sender.Request(ctx,
		pub.GET("https://receiver.verified.source.connector/echo"),
		pub.Header(frame.HeaderFromHost, "attacker.local"),
	)
	assert.NoError(err)
	assert.Equal("sender.verified.source.connector", observedFromHost,
		"receiver must overwrite spoofed From-Host with verified source from the subject")
}

// TestConnector_SubscriptionTimeBudget verifies that sub.TimeBudget shortens
// the inbound handler's context deadline to min(caller budget, declared budget),
// never lengthens it, and is inert when not declared.
func TestConnector_SubscriptionTimeBudget(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	var declaredBudget, undeclaredBudget, callerBudget, longBudget time.Duration
	var declaredRemaining time.Duration

	con := New("time.budget.connector")
	con.Subscribe("Declared",
		func(w http.ResponseWriter, r *http.Request) error {
			declaredBudget = frame.Of(r).TimeBudget()
			if dl, ok := r.Context().Deadline(); ok {
				declaredRemaining = time.Until(dl)
			}
			return nil
		},
		sub.At("GET", "/declared"),
		sub.Web(),
		sub.TimeBudget(2*time.Second),
	)
	con.Subscribe("Undeclared",
		func(w http.ResponseWriter, r *http.Request) error {
			undeclaredBudget = frame.Of(r).TimeBudget()
			return nil
		},
		sub.At("GET", "/undeclared"),
		sub.Web(),
	)
	con.Subscribe("Caller",
		func(w http.ResponseWriter, r *http.Request) error {
			callerBudget = frame.Of(r).TimeBudget()
			return nil
		},
		sub.At("GET", "/caller"),
		sub.Web(),
		sub.TimeBudget(2*time.Second),
	)
	con.Subscribe("Long",
		func(w http.ResponseWriter, r *http.Request) error {
			longBudget = frame.Of(r).TimeBudget()
			return nil
		},
		sub.At("GET", "/long"),
		sub.Web(),
		sub.TimeBudget(40*time.Second),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// No caller budget: the connector default (20s) is clamped down to the
	// declared 2s, and the handler's context deadline reflects it.
	_, err = con.Request(ctx, pub.GET("https://time.budget.connector/declared"))
	assert.NoError(err)
	assert.Equal(2*time.Second, declaredBudget, "declared budget should clamp the connector default")
	assert.True(declaredRemaining > 0 && declaredRemaining <= 2*time.Second,
		"handler deadline should be within the declared budget, got %v", declaredRemaining)
	assert.True(declaredRemaining < con.defaultTimeBudget,
		"declared budget must shorten below the connector default %v, got %v", con.defaultTimeBudget, declaredRemaining)

	// No declared budget: the connector default stays in force, unclamped.
	_, err = con.Request(ctx, pub.GET("https://time.budget.connector/undeclared"))
	assert.NoError(err)
	assert.Equal(con.defaultTimeBudget, undeclaredBudget, "absent sub.TimeBudget leaves the default in force")

	// Caller budget smaller than the declared budget wins (min semantics);
	// the declared budget never lengthens what the caller asked for.
	_, err = con.Request(ctx,
		pub.GET("https://time.budget.connector/caller"),
		pub.Timeout(800*time.Millisecond),
	)
	assert.NoError(err)
	assert.Equal(800*time.Millisecond, callerBudget, "smaller caller budget must win over the declared budget")

	// Declared budget longer than the connector default cannot lengthen it -
	// sub.TimeBudget only ever shortens.
	_, err = con.Request(ctx, pub.GET("https://time.budget.connector/long"))
	assert.NoError(err)
	assert.Equal(con.defaultTimeBudget, longBudget, "a declared budget above the default must not lengthen it")
}

// TestConnector_SubscriptionTimeBudgetReject verifies that when sub.TimeBudget
// shortens the budget below a network round-trip, the subscriber rejects the
// request by sending a 408 error response - so the caller fails fast - rather
// than acking and then silently dropping it, which strands the caller until its
// own (possibly much larger) pub.Timeout. Reproduces the foreman/timebudgetflow
// hang where a 50ms declared budget under a 100ms NATS round-trip wedged a worker
// for the full TimeBudget ceiling.
// Not parallel - asserts an upper bound on elapsed time.
func TestConnector_SubscriptionTimeBudgetReject(t *testing.T) {
	assert := testarossa.For(t)

	ctx := t.Context()

	handlerCalled := make(chan struct{}, 1)
	con := New("time.budget.reject.connector")
	con.Subscribe("Tight",
		func(w http.ResponseWriter, r *http.Request) error {
			select {
			case handlerCalled <- struct{}{}:
			default:
			}
			return nil
		},
		sub.At("GET", "/tight"),
		sub.Web(),
		sub.TimeBudget(50*time.Millisecond),
	)

	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Force the declared 50ms budget below a network round-trip, deterministically
	// and independent of the transport, so handleRequest takes the reject path.
	con.networkRoundtrip = 100 * time.Millisecond

	// The caller's own timeout is generous; only the subscriber knows the declared
	// budget is too small. Pre-fix the caller blocked here until pub.Timeout fired.
	callerTimeout := 4 * time.Second
	start := time.Now()
	_, err = con.Request(ctx,
		pub.GET("https://time.budget.reject.connector/tight"),
		pub.Timeout(callerTimeout),
	)
	elapsed := time.Since(start)

	assert.Error(err)
	assert.Equal(http.StatusRequestTimeout, errors.Convert(err).StatusCode)
	assert.True(elapsed < 2*time.Second,
		"caller must fail fast on a rejected budget, not block until pub.Timeout (%v); took %v",
		callerTimeout, elapsed)
	handlerRan := false
	select {
	case <-handlerCalled:
		handlerRan = true
	default:
	}
	assert.False(handlerRan, "handler must not run when the budget is too small to dispatch")
}

func BenchmarkConnection_AckRequest(b *testing.B) {
	ctx := context.Background()
	// Startup the microservices
	con := New("ack.request.connector")
	err := con.Startup(ctx)
	testarossa.NoError(b, err)
	defer con.Shutdown(ctx)

	req, _ := http.NewRequest("POST", "https://nowhere/", strings.NewReader(utils.RandomIdentifier(16*1024)))
	f := frame.Of(req)
	f.SetFromHost("someone")
	f.SetFromID("me")
	f.SetMessageID("123456")

	var buf bytes.Buffer
	req.Write(&buf)
	msgData := buf.Bytes()

	b.ResetTimer()
	for b.Loop() {
		con.ackRequest(&transport.Msg{
			Data: msgData,
		}, &sub.Subscription{})
	}

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/connector
	// cpu: Apple M1 Pro
	// BenchmarkConnection_AckRequest-10    	  202579	      5860 ns/op	    2508 B/op	      46 allocs/op
}

func TestConnector_PathArguments(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservice
	var foo string
	var bar string
	con := New("path.arguments.connector")
	con.Subscribe("FooBar",
		func(w http.ResponseWriter, r *http.Request) error {
			parts := strings.Split(r.URL.Path, "/")
			foo = parts[2]
			bar = parts[4]
			return nil
		},
		sub.At("ANY", "/foo/{foo}/bar/{bar}"),
		sub.Web(),
	)

	// Startup the microservices
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Values provided in path should be delivered
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo/FOO1/bar/BAR1"))
	if assert.NoError(err) {
		assert.Equal("FOO1", foo)
		assert.Equal("BAR1", bar)
	}
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo/{foo}/bar/{bar}"))
	if assert.NoError(err) {
		assert.Equal("", foo)
		assert.Equal("", bar)
	}
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo//bar/BAR2"))
	if assert.NoError(err) {
		assert.Equal("", foo)
		assert.Equal("BAR2", bar)
	}
}

func TestConnector_InvalidPathArguments(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	for _, path := range []string{
		"/1/x{mmm}x", "/2/{}x", "/3/x{}", "/4/x{...}", "/}{", "/{/x",
	} {
		con := New("invalid.path.arguments.connector")
		con.Subscribe("Bad",
			func(w http.ResponseWriter, r *http.Request) error {
				return nil
			},
			sub.At("GET", path),
			sub.Web(),
		)
		err := con.Startup(ctx)
		if !assert.Error(err, "%", path) {
			con.Shutdown(ctx)
		}
	}
}

func TestConnector_SubscriptionLocality(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservices
	alpha := New("alpha.subscription.locality.connector")
	alpha.SetLocality("az1.dc2.west.us")

	beta1 := New("beta.subscription.locality.connector")
	beta1.SetLocality("az2.dc2.west.us")

	beta2 := New("beta.subscription.locality.connector")
	beta2.SetLocality("az1.dc3.west.us")

	beta3 := New("beta.subscription.locality.connector")
	beta3.SetLocality("az1.dc2.east.us")

	beta4 := New("beta.subscription.locality.connector")
	beta4.SetLocality("az4.dc5.north.eu")

	beta5 := New("beta.subscription.locality.connector")
	beta5.SetLocality("az1.dc1.southwest.ap")

	beta6 := New("beta.subscription.locality.connector")
	beta6.SetLocality("az4.dc2.south.ap")

	// Startup the microservices
	for _, con := range []*Connector{alpha, beta1, beta2, beta3, beta4, beta5, beta6} {
		con.Subscribe("Ok",
			func(w http.ResponseWriter, r *http.Request) error {
				return nil
			},
			sub.At("GET", "ok"),
			sub.Web(),
		)
		err := con.Startup(ctx)
		assert.NoError(err)
		defer con.Shutdown(ctx)
	}

	// Requests should converge to beta1 that is in the same DC as alpha
	for repeat := 0; repeat < 16; repeat++ {
		beta1Found := false
		for sticky := 0; sticky < 16; {
			localityBefore, _ := alpha.localResponder.Load("https://beta.subscription.locality.connector/ok")
			res, err := alpha.GET(ctx, "https://beta.subscription.locality.connector/ok")
			if !assert.NoError(err) {
				break
			}
			localityAfter, _ := alpha.localResponder.Load("https://beta.subscription.locality.connector/ok")
			assert.True(len(localityAfter) >= len(localityBefore))

			if beta1Found {
				// Once beta1 was found, all future requests should go there
				assert.Equal(beta1.id, frame.Of(res).FromID())
				sticky++
			}
			beta1Found = frame.Of(res).FromID() == beta1.id
		}
		alpha.localResponder.Clear() // Reset
	}

	// Shutting down beta1, requests should converge to beta2 that is in the same region as alpha
	beta1.Shutdown(ctx)

	for repeat := 0; repeat < 16; repeat++ {
		beta2Found := false
		for sticky := 0; sticky < 16; {
			res, err := alpha.GET(ctx, "https://beta.subscription.locality.connector/ok")
			if !assert.NoError(err) {
				break
			}
			assert.NotEqual(beta1.id, frame.Of(res).FromID()) // beta1 was shut down
			if beta2Found {
				// Once beta2 was found, all future requests should go there
				assert.Equal(beta2.id, frame.Of(res).FromID())
				sticky++
			}
			beta2Found = frame.Of(res).FromID() == beta2.id
		}
		alpha.localResponder.Clear() // Reset
	}
}

func TestConnector_SubscriptionNoLocalityWithID(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	// Create the microservices
	alpha := New("alpha.subscription.no.locality.with.id.connector")
	alpha.SetLocality("az1.dc2.west.us")

	beta := New("beta.subscription.no.locality.with.id.connector")
	beta.SetLocality("az1.dc2.west.us")
	beta.Subscribe("ByID",
		func(w http.ResponseWriter, r *http.Request) error {
			// When targeting a microservice by its ID, locality-aware optimization should not kick in
			assert.Equal(beta.id+".beta.subscription.no.locality.with.id.connector:443", r.Host)
			return nil
		},
		sub.At("GET", "byid"),
		sub.Web(),
	)
	first := true
	beta.Subscribe("ByHost",
		func(w http.ResponseWriter, r *http.Request) error {
			// When targeting by host, locality-aware optimization should kick in after the first request
			if first {
				assert.Equal("beta.subscription.no.locality.with.id.connector:443", r.Host)
				first = false
			} else {
				assert.Equal("loc-us-west-dc2-az1.beta.subscription.no.locality.with.id.connector:443", r.Host)
			}
			return nil
		},
		sub.At("GET", "byhost"),
		sub.Web(),
	)

	err := alpha.Startup(ctx)
	assert.NoError(err)
	defer alpha.Shutdown(ctx)
	err = beta.Startup(ctx)
	assert.NoError(err)
	defer beta.Shutdown(ctx)

	for repeat := 0; repeat < 16; repeat++ {
		_, err := alpha.GET(ctx, "https://"+beta.ID()+".beta.subscription.no.locality.with.id.connector/byid")
		if !assert.NoError(err) {
			break
		}
	}
	for repeat := 0; repeat < 16; repeat++ {
		_, err := alpha.GET(ctx, "https://beta.subscription.no.locality.with.id.connector/byhost")
		if !assert.NoError(err) {
			break
		}
	}
}

func TestConnector_Actor(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type JWK struct {
		KTY string `json:"kty"`
		CRV string `json:"crv"`
		X   string `json:"x"`
		KID string `json:"kid"`
	}

	// Generate a key pair for the mock token issuer
	pub25519, priv25519, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	hash := sha256.Sum256(pub25519)
	kid := base64.RawURLEncoding.EncodeToString(hash[:8])
	signToken := func(claims jwt.MapClaims) (string, error) {
		claims["iss"] = "https://access.token.core"
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		token.Header["kid"] = kid
		signed, err := token.SignedString(priv25519)
		if err != nil {
			return "", err
		}
		return signed, nil
	}

	// Create a mock token issuer that serves JWKS
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

	// Create the microservice under test
	entered := 0
	con := New("con.actor.connector")
	con.Subscribe("Student",
		func(w http.ResponseWriter, r *http.Request) error {
			entered++
			return nil
		},
		sub.At("GET", "student"),
		sub.Web(),
		sub.RequiredClaims(`iss=="https://access.token.core" && (roles.student || roles.professor)`),
	)
	con.Subscribe("Professor",
		func(w http.ResponseWriter, r *http.Request) error {
			entered++
			return nil
		},
		sub.At("GET", "professor"),
		sub.Web(),
		sub.RequiredClaims(`iss=="https://access.token.core" && roles.professor`),
	)

	// Startup both connectors
	ctx := t.Context()
	err = issuer.Startup(ctx)
	assert.NoError(err)
	defer issuer.Shutdown(ctx)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Without a token
	_, err = con.GET(ctx, "https://con.actor.connector/student")
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
	assert.Equal(0, entered)
	_, err = con.GET(ctx, "https://con.actor.connector/professor")
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
	assert.Equal(0, entered)

	// Create token for wizard role
	harry, err := signToken(jwt.MapClaims{
		"sub":   "harry@hogwarts.edu",
		"roles": []string{"wizard", "student"},
	})
	assert.NoError(err)
	_, err = con.Request(ctx, pub.GET("https://con.actor.connector/student"), pub.Token(harry))
	assert.NoError(err)
	assert.Equal(1, entered)
	_, err = con.Request(ctx, pub.GET("https://con.actor.connector/professor"), pub.Token(harry))
	assert.Error(err)
	assert.Equal(http.StatusForbidden, errors.StatusCode(err))
	assert.Equal(1, entered)

	// Create token for professor role
	dumbledore, err := signToken(jwt.MapClaims{
		"sub":   "dumbledore@hogwarts.edu",
		"roles": []string{"wizard", "professor", "headmaster"},
	})
	assert.NoError(err)
	_, err = con.Request(ctx, pub.GET("https://con.actor.connector/student"), pub.Token(dumbledore))
	assert.NoError(err)
	assert.Equal(2, entered)
	_, err = con.Request(ctx, pub.GET("https://con.actor.connector/professor"), pub.Token(dumbledore))
	assert.NoError(err)
	assert.Equal(3, entered)
}

// TestConnector_ActorVerifiedWithoutRequiredClaims verifies that a present actor token is
// signature-verified even on an endpoint that declares no requiredClaims. A missing token is
// still allowed through (the endpoint is public), but a token that fails verification is
// rejected with 401 rather than reaching the handler with unverified claims.
func TestConnector_ActorVerifiedWithoutRequiredClaims(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type JWK struct {
		KTY string `json:"kty"`
		CRV string `json:"crv"`
		X   string `json:"x"`
		KID string `json:"kid"`
	}

	// Generate the issuer key pair and an unrelated attacker key pair
	pub25519, priv25519, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	_, attackerPriv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	hash := sha256.Sum256(pub25519)
	kid := base64.RawURLEncoding.EncodeToString(hash[:8])
	signWith := func(key ed25519.PrivateKey, claims jwt.MapClaims) (string, error) {
		claims["iss"] = "https://access.token.core"
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		token.Header["kid"] = kid
		return token.SignedString(key)
	}

	// Mock token issuer serving JWKS for the issuer key only
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

	// Endpoint with no requiredClaims that reads the actor claims
	entered := 0
	sawAdmin := false
	con := New("con.actor.noclaims.connector")
	con.Subscribe("Open",
		func(w http.ResponseWriter, r *http.Request) error {
			entered++
			sawAdmin, _ = frame.Of(r).IfActor("roles.admin")
			return nil
		},
		sub.At("GET", "open"),
		sub.Web(),
	)

	ctx := t.Context()
	err = issuer.Startup(ctx)
	assert.NoError(err)
	defer issuer.Shutdown(ctx)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// No token: the endpoint is public and serves the request
	_, err = con.GET(ctx, "https://con.actor.noclaims.connector/open")
	assert.NoError(err)
	assert.Equal(1, entered)
	assert.False(sawAdmin)

	// A properly signed token is verified and its claims are readable in the handler
	valid, err := signWith(priv25519, jwt.MapClaims{"sub": "root@example.com", "roles": []string{"admin"}})
	assert.NoError(err)
	_, err = con.Request(ctx, pub.GET("https://con.actor.noclaims.connector/open"), pub.Token(valid))
	assert.NoError(err)
	assert.Equal(2, entered)
	assert.True(sawAdmin)

	// A token signed by an attacker key (same kid) fails signature verification and is rejected,
	// even though the endpoint declares no requiredClaims
	forged, err := signWith(attackerPriv, jwt.MapClaims{"sub": "mallory@example.com", "roles": []string{"admin"}})
	assert.NoError(err)
	_, err = con.Request(ctx, pub.GET("https://con.actor.noclaims.connector/open"), pub.Token(forged))
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
	assert.Equal(2, entered)
}

// TestConnector_ActorPinnedIssuer verifies that the JWKS-pinning gate rejects tokens
// whose iss claim points at a hostname that is not on the framework's pinned-issuer list,
// even when the token would otherwise be syntactically valid and signed correctly. The
// rejection must happen before any JWKS fetch is attempted, so an attacker that stands
// up a fake JWKS endpoint at their hostname cannot influence verification.
func TestConnector_ActorPinnedIssuer(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

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

	// Stand up an attacker-controlled "issuer" at a hostname that is NOT pinned. The
	// attacker serves a JWKS at :888/jwks like a legitimate issuer would. If the
	// verifier ever fetched JWKS from this hostname, the forged token would verify.
	jwksFetched := false
	attacker := New("attacker.local")
	attacker.Subscribe("JWKS",
		func(w http.ResponseWriter, r *http.Request) error {
			jwksFetched = true
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

	// Service under test gates an endpoint behind a roles claim. With pinning, no token
	// signed by the attacker can satisfy this gate regardless of the claims it carries.
	entered := 0
	con := New("con.pinned.issuer.connector")
	con.Subscribe("Admin",
		func(w http.ResponseWriter, r *http.Request) error {
			entered++
			return nil
		},
		sub.At("GET", "admin"),
		sub.Web(),
		sub.RequiredClaims(`roles.admin`),
	)

	ctx := t.Context()
	err = attacker.Startup(ctx)
	assert.NoError(err)
	defer attacker.Shutdown(ctx)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Forge a token claiming iss=attacker.local with an admin role. The signature is
	// valid against the attacker's published JWKS, so a naive verifier that follows
	// iss would accept it. The pinned verifier must reject it.
	forgedClaims := jwt.MapClaims{
		"iss":   "https://attacker.local",
		"sub":   "evil@attacker.local",
		"roles": []string{"admin"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, forgedClaims)
	token.Header["kid"] = kid
	forgedToken, err := token.SignedString(priv25519)
	assert.NoError(err)

	_, err = con.Request(ctx, pub.GET("https://con.pinned.issuer.connector/admin"), pub.Token(forgedToken))
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
	assert.Equal(0, entered)
	assert.False(jwksFetched, "verifier must not fetch JWKS from a non-pinned issuer hostname")

	// A token claiming iss=https://access.token.core (a pinned hostname) but signed by
	// the attacker's key is also rejected - the iss check passes, but the verifier
	// fetches JWKS from the real access.token.core (which has no JWKS subscription in
	// this test) and the signature verification fails.
	wrongIssuerClaims := jwt.MapClaims{
		"iss":   "https://access.token.core",
		"sub":   "evil@attacker.local",
		"roles": []string{"admin"},
	}
	token2 := jwt.NewWithClaims(jwt.SigningMethodEdDSA, wrongIssuerClaims)
	token2.Header["kid"] = kid
	mismatchedToken, err := token2.SignedString(priv25519)
	assert.NoError(err)

	_, err = con.Request(ctx, pub.GET("https://con.pinned.issuer.connector/admin"), pub.Token(mismatchedToken))
	assert.Error(err)
	assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
	assert.Equal(0, entered)
}

func TestConnector_SubscribeDefaults(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.defaults.connector")
	noopHandler := func(w http.ResponseWriter, r *http.Request) error { return nil }

	cases := []struct {
		name    string
		feature sub.Option
		port    string
		route   string
	}{
		{"FunctionEndpoint", sub.Function(struct{}{}, struct{}{}), "443", "/function-endpoint"},
		{"WebEndpoint", sub.Web(), "443", "/web-endpoint"},
		{"InboundEventEndpoint", sub.InboundEvent(struct{}{}, struct{}{}), "417", "/inbound-event-endpoint"},
		{"TaskEndpoint", sub.Task(struct{}{}, struct{}{}), "428", "/task-endpoint"},
		{"WorkflowEndpoint", sub.Workflow(struct{}{}, struct{}{}), "428", "/workflow-endpoint"},
	}
	for _, tc := range cases {
		err := con.Subscribe(tc.name, noopHandler, tc.feature)
		assert.NoError(err)
	}

	stored := con.subs.Snapshot()
	for _, tc := range cases {
		s, ok := stored[tc.name]
		if !assert.True(ok) {
			continue
		}
		assert.Equal(tc.name, s.Name)
		assert.Equal("ANY", s.Method)
		assert.Equal(tc.port, s.Port)
		assert.Equal(tc.route, s.Path)
		assert.Equal("subscribe.defaults.connector", s.Host)
		assert.Equal("subscribe.defaults.connector", s.Queue)
	}
}

func TestConnector_SubscribeOverrides(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.overrides.connector")
	noopHandler := func(w http.ResponseWriter, r *http.Request) error { return nil }

	err := con.Subscribe("MyEndpoint", noopHandler,
		sub.At("POST", ":1080/custom-route"),
		sub.Description("does X"),
		sub.Function(struct{}{}, struct{}{}),
		sub.NoQueue(),
	)
	assert.NoError(err)

	s, ok := con.subs.Load("MyEndpoint")
	if !assert.True(ok) {
		return
	}
	assert.Equal("MyEndpoint", s.Name)
	assert.Equal("POST", s.Method)
	assert.Equal("1080", s.Port)
	assert.Equal("/custom-route", s.Path)
	assert.Equal("does X", s.Description)
	assert.Equal(sub.TypeFunction, s.Type)
	assert.Equal("", s.Queue)
}

func TestConnector_SubscribeInvalidName(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.invalid.name.connector")
	noopHandler := func(w http.ResponseWriter, r *http.Request) error { return nil }

	for _, bad := range []string{"", "lower", "_Underscore", "Has-Hyphen", "Has Space", "9StartsWithDigit"} {
		err := con.Subscribe(bad, noopHandler, sub.Web())
		assert.Error(err)
	}

	// Anon_<random> from legacy Subscribe must not collide with valid Listen names because
	// the underscore makes it not match IsUpperCaseIdentifier.
	err := con.Subscribe("Anon_X", noopHandler, sub.Web())
	assert.Error(err)
}

func TestConnector_SubscribeDuplicateName(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.duplicate.connector")
	noopHandler := func(w http.ResponseWriter, r *http.Request) error { return nil }

	err := con.Subscribe("MyEndpoint", noopHandler, sub.Web())
	assert.NoError(err)

	err = con.Subscribe("MyEndpoint", noopHandler, sub.Web())
	assert.Error(err)
}

func TestConnector_SubscribeFeatureCount(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.feature.count.connector")
	noopHandler := func(w http.ResponseWriter, r *http.Request) error { return nil }

	// Zero feature options.
	err := con.Subscribe("NoFeature", noopHandler)
	assert.Error(err)

	// Two feature options.
	err = con.Subscribe("TwoFeatures", noopHandler, sub.Web(), sub.Function(struct{}{}, struct{}{}))
	assert.Error(err)
}

func TestConnector_SubscribeNilHandler(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("subscribe.nil.handler.connector")
	err := con.Subscribe("MyEndpoint", nil, sub.Web())
	assert.Error(err)
}

func TestConnector_UnsubscribeUnknown(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("unsubscribe.unknown.connector")
	err := con.Unsubscribe("DoesNotExist")
	assert.Error(err)
}

func TestConnector_SubscribeUnsubscribeRoundtrip(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	count := 0
	con := New("subscribe.roundtrip.connector")
	err := con.Subscribe("Greet",
		func(w http.ResponseWriter, r *http.Request) error {
			count++
			return nil
		},
		sub.Web(),
	)
	assert.NoError(err)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	_, err = con.GET(ctx, "https://subscribe.roundtrip.connector/greet")
	assert.NoError(err)
	assert.Equal(1, count)

	err = con.Unsubscribe("Greet")
	assert.NoError(err)

	_, err = con.GET(ctx, "https://subscribe.roundtrip.connector/greet")
	assert.Error(err)
}

func TestConnector_ManualSubscription(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	calls := 0
	con := New("manual.subscription.connector")
	con.SetDeployment(LOCAL) // TESTING auto-activates manual subs; opt out to exercise the gating
	err := con.Subscribe("ManualEndpoint",
		func(w http.ResponseWriter, r *http.Request) error {
			calls++
			return nil
		},
		sub.At("GET", "manual-endpoint"),
		sub.Web(),
		sub.Manual(),
	)
	assert.NoError(err)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Before ActivateSubscription, callers see a 404 ack-timeout because the sub
	// is registered but not on the bus.
	_, err = con.GET(ctx, "https://manual.subscription.connector/manual-endpoint")
	assert.Error(err)
	assert.Equal(0, calls)

	// Activate and verify traffic now reaches the handler.
	err = con.ActivateSubscription("ManualEndpoint")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://manual.subscription.connector/manual-endpoint")
	assert.NoError(err)
	assert.Equal(1, calls)

	// Second call on the same sub is a no-op.
	err = con.ActivateSubscription("ManualEndpoint")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://manual.subscription.connector/manual-endpoint")
	assert.NoError(err)
	assert.Equal(2, calls)
}

func TestConnector_ManualSubscription_RegisteredPostStartup(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()

	calls := 0
	con := New("manual.subscription.post.startup.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Register a manual sub after startup; it should NOT activate immediately.
	err = con.Subscribe("LateManual",
		func(w http.ResponseWriter, r *http.Request) error {
			calls++
			return nil
		},
		sub.At("GET", "late-manual"),
		sub.Web(),
		sub.Manual(),
	)
	assert.NoError(err)

	_, err = con.GET(ctx, "https://manual.subscription.post.startup.connector/late-manual")
	assert.Error(err)
	assert.Equal(0, calls)

	err = con.ActivateSubscription("LateManual")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://manual.subscription.post.startup.connector/late-manual")
	assert.NoError(err)
	assert.Equal(1, calls)
}

func TestConnector_ActivateSubscription_UnknownName(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()
	con := New("activate.unknown.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	err = con.ActivateSubscription("NoSuchSub")
	assert.Error(err)
	err = con.DeactivateSubscription("NoSuchSub")
	assert.Error(err)
}

func TestConnector_ActivateSubscription_IteratesByTag(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()
	con := New("activate.by.tag.connector")
	con.SetDeployment(LOCAL) // TESTING auto-activates manual subs; opt out to exercise the gating
	callsA, callsB, callsC := 0, 0, 0
	con.Subscribe("A",
		func(w http.ResponseWriter, r *http.Request) error { callsA++; return nil },
		sub.At("GET", "a"), sub.Web(), sub.Manual(), sub.Tag("python"),
	)
	con.Subscribe("B",
		func(w http.ResponseWriter, r *http.Request) error { callsB++; return nil },
		sub.At("GET", "b"), sub.Web(), sub.Manual(), sub.Tag("python"),
	)
	con.Subscribe("C",
		func(w http.ResponseWriter, r *http.Request) error { callsC++; return nil },
		sub.At("GET", "c"), sub.Web(), sub.Manual(),
	)
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	// Activate only python-tagged subs by iterating Subscriptions().
	activated := 0
	for _, s := range con.Subscriptions() {
		for _, t := range s.Tags {
			if t == "python" {
				assert.NoError(con.ActivateSubscription(s.Name))
				activated++
				break
			}
		}
	}
	assert.Equal(2, activated)

	_, err = con.GET(ctx, "https://activate.by.tag.connector/a")
	assert.NoError(err)
	_, err = con.GET(ctx, "https://activate.by.tag.connector/b")
	assert.NoError(err)
	// C is still manual and off-bus.
	_, err = con.GET(ctx, "https://activate.by.tag.connector/c")
	assert.Error(err)
	assert.Equal(1, callsA)
	assert.Equal(1, callsB)
	assert.Equal(0, callsC)

	// Deactivate one by name; the other stays active.
	assert.NoError(con.DeactivateSubscription("A"))
	_, err = con.GET(ctx, "https://activate.by.tag.connector/a")
	assert.Error(err)
	_, err = con.GET(ctx, "https://activate.by.tag.connector/b")
	assert.NoError(err)
}

func TestConnector_Subscriptions_Snapshot(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	ctx := t.Context()
	con := New("subscriptions.snapshot.connector")
	con.SetDeployment(LOCAL) // TESTING auto-activates manual subs; opt out to exercise the gating
	err := con.Subscribe("Tagged",
		func(w http.ResponseWriter, r *http.Request) error { return nil },
		sub.At("GET", "tagged"), sub.Web(), sub.Manual(), sub.Tag("a", "b"),
	)
	assert.NoError(err)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	var info SubscriptionInfo
	for _, s := range con.Subscriptions() {
		if s.Name == "Tagged" {
			info = s
			break
		}
	}
	assert.Equal("Tagged", info.Name)
	assert.Equal(true, info.Manual)
	assert.Equal(false, info.Active)
	assert.Equal([]string{"a", "b"}, info.Tags)

	// Mutating the snapshot doesn't affect the live sub.
	info.Tags[0] = "mutated"
	for _, s := range con.Subscriptions() {
		if s.Name == "Tagged" {
			assert.Equal([]string{"a", "b"}, s.Tags)
			break
		}
	}
}

// TestConnector_UnsignedTokenDeploymentGate pins the alg=none carve-out: an unsigned actor token
// (as minted by pub.Actor) is accepted only in the TESTING deployment, where its claims are still
// evaluated against requiredClaims, and is rejected outright everywhere else. This guards against a
// future change silently widening the carve-out to LAB/PROD.
func TestConnector_UnsignedTokenDeploymentGate(t *testing.T) {
	t.Parallel()

	newGated := func(hostname, deployment string) (*Connector, *int) {
		entered := 0
		con := New(hostname)
		con.SetDeployment(deployment)
		con.Subscribe("Admin",
			func(w http.ResponseWriter, r *http.Request) error {
				entered++
				return nil
			},
			sub.At("GET", "admin"),
			sub.Web(),
			sub.RequiredClaims(`roles.admin`),
		)
		return con, &entered
	}

	t.Run("accepted_in_testing", func(t *testing.T) {
		assert := testarossa.For(t)
		ctx := t.Context()
		con, entered := newGated("unsigned.testing.connector", TESTING)
		err := con.Startup(ctx)
		assert.NoError(err)
		defer con.Shutdown(ctx)

		// An unsigned token with the required claim is accepted and reaches the handler.
		_, err = con.Request(ctx, pub.GET("https://unsigned.testing.connector/admin"),
			pub.Actor(jwt.MapClaims{"roles": []string{"admin"}}))
		assert.NoError(err)
		assert.Equal(1, *entered)

		// An unsigned token that fails the claim is rejected 403 - the claims are still evaluated.
		_, err = con.Request(ctx, pub.GET("https://unsigned.testing.connector/admin"),
			pub.Actor(jwt.MapClaims{"roles": []string{"viewer"}}))
		assert.Error(err)
		assert.Equal(http.StatusForbidden, errors.StatusCode(err))
		assert.Equal(1, *entered)
	})

	t.Run("rejected_outside_testing", func(t *testing.T) {
		assert := testarossa.For(t)
		ctx := t.Context()
		con, entered := newGated("unsigned.lab.connector", LAB)
		err := con.Startup(ctx)
		assert.NoError(err)
		defer con.Shutdown(ctx)

		// The same unsigned token, claim and all, is rejected 401 outside TESTING - the signature
		// (absent) is never trusted, so claims are not even reached.
		_, err = con.Request(ctx, pub.GET("https://unsigned.lab.connector/admin"),
			pub.Actor(jwt.MapClaims{"roles": []string{"admin"}}))
		assert.Error(err)
		assert.Equal(http.StatusUnauthorized, errors.StatusCode(err))
		assert.Equal(0, *entered)
	})
}
