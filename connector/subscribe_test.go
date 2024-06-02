/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestConnector_DirectorySubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var count int
	var appendix string
	con := New("directory.subscription.connector")
	con.Subscribe("GET", "directory/", func(w http.ResponseWriter, r *http.Request) error {
		count++
		appendix = r.URL.Query().Get("appendix")
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/")
	assert.NoError(t, err)
	assert.Equal(t, "", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/1.html")
	assert.NoError(t, err)
	assert.Equal(t, "1.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/2.html")
	assert.NoError(t, err)
	assert.Equal(t, "2.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/sub/3.html")
	assert.NoError(t, err)
	assert.Equal(t, "sub/3.html", appendix)

	assert.Equal(t, 4, count)
}

func TestConnector_HyphenInHostname(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	entered := false
	con := New("hyphen-in-host_name.connector")
	con.Subscribe("GET", "path", func(w http.ResponseWriter, r *http.Request) error {
		entered = true
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://hyphen-in-host_name.connector/path")
	assert.NoError(t, err)
	assert.True(t, entered)
}

func TestConnector_PathArgumentsInSubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	alphaCount := 0
	betaCount := 0
	rootCount := 0
	parentCount := 0
	detected := map[string]string{}
	con := New("path.arguments.in.subscription.connector")
	con.Subscribe("GET", "/obj/{id}/alpha", func(w http.ResponseWriter, r *http.Request) error {
		alphaCount++
		detected[r.URL.Path] = r.URL.Query().Get("id")
		return nil
	})
	con.Subscribe("GET", "/obj/{id}/beta", func(w http.ResponseWriter, r *http.Request) error {
		betaCount++
		detected[r.URL.Path] = r.URL.Query().Get("id")
		return nil
	})
	con.Subscribe("GET", "/obj/{id}", func(w http.ResponseWriter, r *http.Request) error {
		rootCount++
		detected[r.URL.Path] = r.URL.Query().Get("id")
		return nil
	})
	con.Subscribe("GET", "/obj", func(w http.ResponseWriter, r *http.Request) error {
		parentCount++
		detected[r.URL.Path] = r.URL.Query().Get("id")
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1234/alpha")
	assert.NoError(t, err)
	assert.Equal(t, 1, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2345/alpha")
	assert.NoError(t, err)
	assert.Equal(t, 2, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1111/beta")
	assert.NoError(t, err)
	assert.Equal(t, 1, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2222/beta")
	assert.NoError(t, err)
	assert.Equal(t, 2, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/8000")
	assert.NoError(t, err)
	assert.Equal(t, 1, rootCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj")
	assert.NoError(t, err)
	assert.Equal(t, 1, parentCount)

	assert.Len(t, detected, 6)
	assert.Equal(t, "1234", detected["/obj/1234/alpha"])
	assert.Equal(t, "2345", detected["/obj/2345/alpha"])
	assert.Equal(t, "1111", detected["/obj/1111/beta"])
	assert.Equal(t, "2222", detected["/obj/2222/beta"])
	assert.Equal(t, "8000", detected["/obj/8000"])
	assert.Equal(t, "", detected["/obj"])
}

func TestConnector_MixedAsteriskSubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	detected := map[string]bool{}
	con := New("mixed.asterisk.subscription.connector")
	con.Subscribe("GET", "/obj/x*x/gamma", func(w http.ResponseWriter, r *http.Request) error {
		detected[r.URL.Path] = true
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/2222/gamma")
	assert.Error(t, err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x2x/gamma")
	assert.Error(t, err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x*x/gamma")
	assert.NoError(t, err)
}

func TestConnector_ErrorAndPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("error.and.panic.connector")
	con.Subscribe("GET", "usererr", func(w http.ResponseWriter, r *http.Request) error {
		return errors.Newc(http.StatusBadRequest, "bad input")
	})
	con.Subscribe("GET", "err", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("it's bad")
	})
	con.Subscribe("GET", "panic", func(w http.ResponseWriter, r *http.Request) error {
		panic("it's really bad")
	})
	con.Subscribe("GET", "oserr", func(w http.ResponseWriter, r *http.Request) error {
		err := errors.Trace(os.ErrNotExist)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		return err
	})
	con.Subscribe("GET", "stillalive", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://error.and.panic.connector/usererr")
	assert.Error(t, err)
	assert.Equal(t, "bad input", err.Error())
	assert.Equal(t, http.StatusBadRequest, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/err")
	assert.Error(t, err)
	assert.Equal(t, "it's bad", err.Error())
	assert.Equal(t, http.StatusInternalServerError, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/panic")
	assert.Error(t, err)
	assert.Equal(t, "it's really bad", err.Error())

	_, err = con.GET(ctx, "https://error.and.panic.connector/oserr")
	assert.Error(t, err)
	assert.Equal(t, "file does not exist", err.Error())
	assert.False(t, errors.Is(err, os.ErrNotExist)) // Cannot reconstitute error type

	_, err = con.GET(ctx, "https://error.and.panic.connector/stillalive")
	assert.NoError(t, err)
}

func TestConnector_DifferentPlanes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("different.planes.connector")
	alpha.SetPlane("alpha")
	alpha.Subscribe("GET", "id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})

	beta := New("different.planes.connector")
	beta.SetPlane("beta")
	beta.Subscribe("GET", "id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("beta"))
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://different.planes.connector/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("alpha"), body)
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://different.planes.connector/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("beta"), body)
	}
}

func TestConnector_SubscribeBeforeAndAfterStartup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	var beforeCalled, afterCalled bool
	con := New("subscribe.before.and.after.startup.connector")

	// Subscribe before beta is started
	con.Subscribe("GET", "before", func(w http.ResponseWriter, r *http.Request) error {
		beforeCalled = true
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Subscribe after beta is started
	con.Subscribe("GET", "after", func(w http.ResponseWriter, r *http.Request) error {
		afterCalled = true
		return nil
	})

	// Send requests to both handlers
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/before")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/after")
	assert.NoError(t, err)

	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestConnector_Unsubscribe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("unsubscribe.connector")

	// Subscribe
	con.Subscribe("GET", "sub1", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	con.Subscribe("GET", "sub2", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.NoError(t, err)

	// Unsubscribe sub1
	err = con.Unsubscribe("GET", ":443/sub1")
	assert.NoError(t, err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.Error(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.NoError(t, err)

	// Deactivate all
	err = con.deactivateSubs()
	assert.NoError(t, err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	assert.Error(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	assert.Error(t, err)
}

func TestConnector_AnotherHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.another.host.connector")
	alpha.Subscribe("GET", "https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	beta1 := New("beta.another.host.connector")
	beta1.Subscribe("GET", "https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	beta2 := New("beta.another.host.connector")
	beta2.Subscribe("GET", "https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta1.Startup()
	assert.NoError(t, err)
	defer beta1.Shutdown()
	err = beta2.Startup()
	assert.NoError(t, err)
	defer beta2.Shutdown()

	// Send message
	responded := 0
	ch := alpha.Publish(ctx, pub.GET("https://alternative.host.connector/empty"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		assert.NoError(t, err)
		responded++
	}
	// Even though the microservices subscribe to the same alternative host, their queues should be different
	assert.Equal(t, 2, responded)
}

func TestConnector_DirectAddressing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("direct.addressing.connector")
	con.Subscribe("GET", "/hello", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("Hello"))
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	assert.NoError(t, err)

	err = con.Unsubscribe("GET", "/hello")
	assert.NoError(t, err)

	// Both subscriptions should be deactivated
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	assert.Error(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	assert.Error(t, err)
}

func TestConnector_SubPendingOps(t *testing.T) {
	t.Parallel()

	con := New("sub.pending.ops.connector")

	start := make(chan bool)
	hold := make(chan bool)
	end := make(chan bool)
	con.Subscribe("GET", "/op", func(w http.ResponseWriter, r *http.Request) error {
		start <- true
		hold <- true
		return nil
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Zero(t, con.pendingOps)

	// First call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	assert.Equal(t, int32(1), con.pendingOps)

	// Second call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	assert.Equal(t, int32(2), con.pendingOps)

	<-hold
	<-end
	assert.Equal(t, int32(1), con.pendingOps)
	<-hold
	<-end
	assert.Zero(t, con.pendingOps)
}

func TestConnector_SubscriptionMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var get int
	var post int
	var star int
	con := New("subscription.methods.connector")
	con.Subscribe("GET", "single", func(w http.ResponseWriter, r *http.Request) error {
		get++
		return nil
	})
	con.Subscribe("POST", "single", func(w http.ResponseWriter, r *http.Request) error {
		post++
		return nil
	})
	con.Subscribe("ANY", "star", func(w http.ResponseWriter, r *http.Request) error {
		star++
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/single"))
	assert.NoError(t, err)
	assert.Equal(t, 1, get)
	assert.Equal(t, 0, post)

	_, err = con.Request(ctx, pub.POST("https://subscription.methods.connector/single"))
	assert.NoError(t, err)
	assert.Equal(t, 1, get)
	assert.Equal(t, 1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/single"))
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.Equal(t, 1, get)
	assert.Equal(t, 1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/star"))
	assert.NoError(t, err)
	assert.Equal(t, 1, get)
	assert.Equal(t, 1, post)
	assert.Equal(t, 1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/star"))
	assert.NoError(t, err)
	assert.Equal(t, 1, get)
	assert.Equal(t, 1, post)
	assert.Equal(t, 2, star)
}

func TestConnector_SubscriptionPorts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var p123 int
	var p234 int
	var star int
	con := New("subscription.ports.connector")
	con.Subscribe("GET", ":123/single", func(w http.ResponseWriter, r *http.Request) error {
		p123++
		return nil
	})
	con.Subscribe("GET", ":234/single", func(w http.ResponseWriter, r *http.Request) error {
		p234++
		return nil
	})
	con.Subscribe("GET", ":0/any", func(w http.ResponseWriter, r *http.Request) error {
		star++
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:123/single"))
	assert.NoError(t, err)
	assert.Equal(t, 1, p123)
	assert.Equal(t, 0, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:234/single"))
	assert.NoError(t, err)
	assert.Equal(t, 1, p123)
	assert.Equal(t, 1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/single"))
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.Equal(t, 1, p123)
	assert.Equal(t, 1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/any"))
	assert.NoError(t, err)
	assert.Equal(t, 1, p123)
	assert.Equal(t, 1, p234)
	assert.Equal(t, 1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:10000/any"))
	assert.NoError(t, err)
	assert.Equal(t, 1, p123)
	assert.Equal(t, 1, p234)
	assert.Equal(t, 2, star)
}

func TestConnector_FrameConsistency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("frame.consistency.connector")
	con.Subscribe("GET", "/frame", func(w http.ResponseWriter, r *http.Request) error {
		f1 := frame.Of(r)
		f2 := frame.Of(r.Context())
		assert.Equal(t, f1, f2)
		f1.Set("ABC", "abc")
		assert.Equal(t, f1, f2)
		assert.Equal(t, "abc", f2.Get("ABC"))
		f2.Set("ABC", "")
		assert.Equal(t, f1, f2)
		assert.Equal(t, "", f1.Get("ABC"))
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://frame.consistency.connector/frame"))
	assert.NoError(t, err)
}

func BenchmarkConnection_AckRequest(b *testing.B) {
	// Startup the microservices
	con := New("ack.request.connector")
	err := con.Startup()
	assert.NoError(b, err)
	defer con.Shutdown()

	req, _ := http.NewRequest("POST", "https://nowhere/", strings.NewReader(rand.AlphaNum64(16*1024)))
	f := frame.Of(req)
	f.SetFromHost("someone")
	f.SetFromID("me")
	f.SetMessageID("123456")

	var buf bytes.Buffer
	req.Write(&buf)
	msgData := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		con.ackRequest(&nats.Msg{
			Data: msgData,
		}, &sub.Subscription{})
	}

	// On 2021 MacBook Pro M1 16":
	// N=256477
	// 4782 ns/op (209117 ops/sec)
	// 6045 B/op
	// 26 allocs/op
}

func TestConnector_PathArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var foo string
	var bar string
	con := New("path.arguments.connector")
	con.Subscribe("GET", "/foo/{foo}/bar/{bar}", func(w http.ResponseWriter, r *http.Request) error {
		foo = r.URL.Query().Get("foo")
		bar = r.URL.Query().Get("bar")
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo/FOO/bar/BAR?foo=x&bar=x"))
	assert.NoError(t, err)
	assert.Equal(t, "FOO", foo)
	assert.Equal(t, "BAR", bar)
}
