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
	"github.com/microbus-io/testarossa"
	"github.com/nats-io/nats.go"
)

func TestConnector_DirectorySubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var count int
	var appendix string
	con := New("directory.subscription.connector")
	con.Subscribe("GET", "directory/{appendix+}", func(w http.ResponseWriter, r *http.Request) error {
		count++
		_, appendix, _ = strings.Cut(r.URL.Path, "/directory/")
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/1.html")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "1.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/2.html")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "2.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/sub/3.html")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "sub/3.html", appendix)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "", appendix)

	testarossa.Equal(t, 4, count)

	// The path of the directory should not be captured
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory")
	testarossa.Error(t, err)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://hyphen-in-host_name.connector/path")
	testarossa.NoError(t, err)
	testarossa.True(t, entered)
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
		parts := strings.Split(r.URL.Path, "/")
		detected[r.URL.Path] = parts[2]
		return nil
	})
	con.Subscribe("GET", "/obj/{id}/beta", func(w http.ResponseWriter, r *http.Request) error {
		betaCount++
		parts := strings.Split(r.URL.Path, "/")
		detected[r.URL.Path] = parts[2]
		return nil
	})
	con.Subscribe("GET", "/obj/{id}", func(w http.ResponseWriter, r *http.Request) error {
		rootCount++
		parts := strings.Split(r.URL.Path, "/")
		detected[r.URL.Path] = parts[2]
		return nil
	})
	con.Subscribe("GET", "/obj", func(w http.ResponseWriter, r *http.Request) error {
		parentCount++
		detected[r.URL.Path] = ""
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1234/alpha")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2345/alpha")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 2, alphaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/1111/beta")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/2222/beta")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 2, betaCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj/8000")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, rootCount)
	_, err = con.GET(ctx, "https://path.arguments.in.subscription.connector/obj")
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, parentCount)

	testarossa.Equal(t, 6, len(detected))
	testarossa.Equal(t, "1234", detected["/obj/1234/alpha"])
	testarossa.Equal(t, "2345", detected["/obj/2345/alpha"])
	testarossa.Equal(t, "1111", detected["/obj/1111/beta"])
	testarossa.Equal(t, "2222", detected["/obj/2222/beta"])
	testarossa.Equal(t, "8000", detected["/obj/8000"])
	testarossa.Equal(t, "", detected["/obj"])
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/2222/gamma")
	testarossa.Error(t, err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x2x/gamma")
	testarossa.Error(t, err)
	_, err = con.GET(ctx, "https://mixed.asterisk.subscription.connector/obj/x*x/gamma")
	testarossa.NoError(t, err)
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
		testarossa.True(t, errors.Is(err, os.ErrNotExist))
		return err
	})
	con.Subscribe("GET", "stillalive", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://error.and.panic.connector/usererr")
	testarossa.Error(t, err)
	testarossa.Equal(t, "bad input", err.Error())
	testarossa.Equal(t, http.StatusBadRequest, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/err")
	testarossa.Error(t, err)
	testarossa.Equal(t, "it's bad", err.Error())
	testarossa.Equal(t, http.StatusInternalServerError, errors.Convert(err).StatusCode)

	_, err = con.GET(ctx, "https://error.and.panic.connector/panic")
	testarossa.Error(t, err)
	testarossa.Equal(t, "it's really bad", err.Error())

	_, err = con.GET(ctx, "https://error.and.panic.connector/oserr")
	testarossa.Error(t, err)
	testarossa.Equal(t, "file does not exist", err.Error())
	testarossa.False(t, errors.Is(err, os.ErrNotExist)) // Cannot reconstitute error type

	_, err = con.GET(ctx, "https://error.and.panic.connector/stillalive")
	testarossa.NoError(t, err)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://different.planes.connector/id")
		testarossa.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, "alpha", string(body))
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://different.planes.connector/id")
		testarossa.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, "beta", string(body))
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Subscribe after beta is started
	con.Subscribe("GET", "after", func(w http.ResponseWriter, r *http.Request) error {
		afterCalled = true
		return nil
	})

	// Send requests to both handlers
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/before")
	testarossa.NoError(t, err)
	_, err = con.GET(ctx, "https://subscribe.before.and.after.startup.connector/after")
	testarossa.NoError(t, err)

	testarossa.True(t, beforeCalled)
	testarossa.True(t, afterCalled)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	testarossa.NoError(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	testarossa.NoError(t, err)

	// Unsubscribe sub1
	err = con.Unsubscribe("GET", ":443/sub1")
	testarossa.NoError(t, err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	testarossa.Error(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	testarossa.NoError(t, err)

	// Deactivate all
	err = con.deactivateSubs()
	testarossa.NoError(t, err)

	// Send requests
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub1")
	testarossa.Error(t, err)
	_, err = con.GET(ctx, "https://unsubscribe.connector/sub2")
	testarossa.Error(t, err)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta1.Startup()
	testarossa.NoError(t, err)
	defer beta1.Shutdown()
	err = beta2.Startup()
	testarossa.NoError(t, err)
	defer beta2.Shutdown()

	// Send message
	responded := 0
	ch := alpha.Publish(ctx, pub.GET("https://alternative.host.connector/empty"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		testarossa.NoError(t, err)
		responded++
	}
	// Even though the microservices subscribe to the same alternative host, their queues should be different
	testarossa.Equal(t, 2, responded)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	testarossa.NoError(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	testarossa.NoError(t, err)

	err = con.Unsubscribe("GET", "/hello")
	testarossa.NoError(t, err)

	// Both subscriptions should be deactivated
	_, err = con.GET(ctx, "https://direct.addressing.connector/hello")
	testarossa.Error(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".direct.addressing.connector/hello")
	testarossa.Error(t, err)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Zero(t, con.pendingOps)

	// First call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	testarossa.Equal(t, int32(1), con.pendingOps)

	// Second call
	go func() {
		con.GET(con.Lifetime(), "https://sub.pending.ops.connector/op")
		end <- true
	}()
	<-start
	testarossa.Equal(t, int32(2), con.pendingOps)

	<-hold
	<-end
	testarossa.Equal(t, int32(1), con.pendingOps)
	<-hold
	<-end
	testarossa.Zero(t, con.pendingOps)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/single"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, get)
	testarossa.Zero(t, post)

	_, err = con.Request(ctx, pub.POST("https://subscription.methods.connector/single"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, get)
	testarossa.Equal(t, 1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/single"))
	testarossa.Error(t, err)
	testarossa.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	testarossa.Equal(t, 1, get)
	testarossa.Equal(t, 1, post)

	_, err = con.Request(ctx, pub.PATCH("https://subscription.methods.connector/star"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, get)
	testarossa.Equal(t, 1, post)
	testarossa.Equal(t, 1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.methods.connector/star"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, get)
	testarossa.Equal(t, 1, post)
	testarossa.Equal(t, 2, star)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:123/single"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, p123)
	testarossa.Zero(t, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:234/single"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, p123)
	testarossa.Equal(t, 1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/single"))
	testarossa.Error(t, err)
	testarossa.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	testarossa.Equal(t, 1, p123)
	testarossa.Equal(t, 1, p234)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:999/any"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, p123)
	testarossa.Equal(t, 1, p234)
	testarossa.Equal(t, 1, star)

	_, err = con.Request(ctx, pub.GET("https://subscription.ports.connector:10000/any"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, p123)
	testarossa.Equal(t, 1, p234)
	testarossa.Equal(t, 2, star)
}

func TestConnector_FrameConsistency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("frame.consistency.connector")
	con.Subscribe("GET", "/frame", func(w http.ResponseWriter, r *http.Request) error {
		f1 := frame.Of(r)
		f2 := frame.Of(r.Context())
		testarossa.Equal(t, &f1, &f2)
		f1.Set("ABC", "abc")
		testarossa.Equal(t, &f1, &f2)
		testarossa.Equal(t, "abc", f2.Get("ABC"))
		f2.Set("ABC", "")
		testarossa.Equal(t, &f1, &f2)
		testarossa.Equal(t, "", f1.Get("ABC"))
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.Request(ctx, pub.GET("https://frame.consistency.connector/frame"))
	testarossa.NoError(t, err)
}

func BenchmarkConnection_AckRequest(b *testing.B) {
	// Startup the microservices
	con := New("ack.request.connector")
	err := con.Startup()
	testarossa.NoError(b, err)
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
	// N=271141
	// 4412 ns/op (226654 ops/sec)
	// 5917 B/op
	// 26 allocs/op
}

func TestConnector_PathArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var foo string
	var bar string
	con := New("path.arguments.connector")
	con.Subscribe("ANY", "/foo/{foo}/bar/{bar}", func(w http.ResponseWriter, r *http.Request) error {
		parts := strings.Split(r.URL.Path, "/")
		foo = parts[2]
		bar = parts[4]
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Values provided in path should be delivered
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo/FOO1/bar/BAR1"))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "FOO1", foo)
		testarossa.Equal(t, "BAR1", bar)
	}
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo/{foo}/bar/{bar}"))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "{foo}", foo)
		testarossa.Equal(t, "{bar}", bar)
	}
	_, err = con.Request(ctx, pub.GET("https://path.arguments.connector/foo//bar/BAR2"))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "", foo)
		testarossa.Equal(t, "BAR2", bar)
	}
}

func TestConnector_InvalidPathArguments(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/1/x{mmm}x", "/2/{}x", "/3/x{}", "/4/x{+}", "/}{", "/{/x",
	} {
		con := New("invalid.path.arguments.connector")
		con.Subscribe("GET", path, func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
		err := con.Startup()
		if !testarossa.Error(t, err, path) {
			con.Shutdown()
		}
	}
}

func TestConnector_SubscriptionLocality(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.subscription.locality.connector")
	alpha.SetLocality("az1.dC2.weSt.Us")

	beta1 := New("beta.subscription.locality.connector")
	beta1.SetLocality("az2.dc2.WEST.us")

	beta2 := New("beta.subscription.locality.connector")
	beta2.SetLocality("az1.DC3.west.us")

	beta3 := New("beta.subscription.locality.connector")
	beta3.SetLocality("az1.dc2.east.US")

	beta4 := New("beta.subscription.locality.connector")
	beta4.SetLocality("az4.dc5.north.eu")

	beta5 := New("beta.subscription.locality.connector")
	beta5.SetLocality("az1.dc1.southwest.ap")

	beta6 := New("beta.subscription.locality.connector")
	beta6.SetLocality("az4.dc2.south.ap")

	// Startup the microservices
	for _, con := range []*Connector{alpha, beta1, beta2, beta3, beta4, beta5, beta6} {
		con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
		err := con.Startup()
		testarossa.NoError(t, err)
		defer con.Shutdown()
	}

	// Requests should converge to beta1 that is in the same DC as alpha
	for repeat := 0; repeat < 16; repeat++ {
		beta1Found := false
		for sticky := 0; sticky < 16; {
			localityBefore, _ := alpha.localResponder.Load("https://beta.subscription.locality.connector/ok")
			res, err := alpha.GET(ctx, "https://beta.subscription.locality.connector/ok")
			if !testarossa.NoError(t, err) {
				break
			}
			localityAfter, _ := alpha.localResponder.Load("https://beta.subscription.locality.connector/ok")
			testarossa.True(t, len(localityAfter) >= len(localityBefore))

			if beta1Found {
				// Once beta1 was found, all future requests should go there
				testarossa.Equal(t, beta1.id, frame.Of(res).FromID())
				sticky++
			}
			beta1Found = frame.Of(res).FromID() == beta1.id
		}
		alpha.localResponder.Clear() // Reset
	}

	// Shutting down beta1, requests should converge to beta2 that is in the same region as alpha
	beta1.Shutdown()

	for repeat := 0; repeat < 16; repeat++ {
		beta2Found := false
		for sticky := 0; sticky < 16; {
			res, err := alpha.GET(ctx, "https://beta.subscription.locality.connector/ok")
			if !testarossa.NoError(t, err) {
				break
			}
			testarossa.NotEqual(t, beta1.id, frame.Of(res).FromID()) // beta1 was shut down
			if beta2Found {
				// Once beta2 was found, all future requests should go there
				testarossa.Equal(t, beta2.id, frame.Of(res).FromID())
				sticky++
			}
			beta2Found = frame.Of(res).FromID() == beta2.id
		}
		alpha.localResponder.Clear() // Reset
	}
}
