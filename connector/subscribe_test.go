/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/stretchr/testify/assert"
)

func TestConnector_DirectorySubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var count int32
	con := New("directory.subscription.connector")
	con.SetDeployment(TESTINGAPP)
	con.Subscribe("directory/", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages to various locations under the directory
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/1.html")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/2.html")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://directory.subscription.connector/directory/sub/3.html")
	assert.NoError(t, err)

	assert.Equal(t, int32(4), count)
}

func TestConnector_HyphenInHostName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	entered := false
	con := New("hyphen-in-host_name.connector")
	con.Subscribe("path", func(w http.ResponseWriter, r *http.Request) error {
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

func TestConnector_AsteriskSubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	alphaCount := 0
	betaCount := 0
	rootCount := 0
	parentCount := 0
	detected := map[string]bool{}
	con := New("asterisk.subscription.connector")
	con.SetDeployment(TESTINGAPP)
	con.Subscribe("/obj/*/alpha", func(w http.ResponseWriter, r *http.Request) error {
		alphaCount++
		detected[r.URL.Path] = true
		return nil
	})
	con.Subscribe("/obj/*/beta", func(w http.ResponseWriter, r *http.Request) error {
		betaCount++
		detected[r.URL.Path] = true
		return nil
	})
	con.Subscribe("/obj/*", func(w http.ResponseWriter, r *http.Request) error {
		rootCount++
		detected[r.URL.Path] = true
		return nil
	})
	con.Subscribe("/obj", func(w http.ResponseWriter, r *http.Request) error {
		parentCount++
		detected[r.URL.Path] = true
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj/1234/alpha")
	assert.NoError(t, err)
	assert.Equal(t, 1, alphaCount)
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj/2345/alpha")
	assert.NoError(t, err)
	assert.Equal(t, 2, alphaCount)
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj/1111/beta")
	assert.NoError(t, err)
	assert.Equal(t, 1, betaCount)
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj/2222/beta")
	assert.NoError(t, err)
	assert.Equal(t, 2, betaCount)
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj/8000")
	assert.NoError(t, err)
	assert.Equal(t, 1, rootCount)
	_, err = con.GET(ctx, "https://asterisk.subscription.connector/obj")
	assert.NoError(t, err)
	assert.Equal(t, 1, parentCount)

	assert.Len(t, detected, 6)
	assert.True(t, detected["/obj/1234/alpha"])
	assert.True(t, detected["/obj/2345/alpha"])
	assert.True(t, detected["/obj/1111/beta"])
	assert.True(t, detected["/obj/2222/beta"])
	assert.True(t, detected["/obj/8000"])
	assert.True(t, detected["/obj"])
}

func TestConnector_MixedAsteriskSubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	detected := map[string]bool{}
	con := New("mixed.asterisk.subscription.connector")
	con.SetDeployment(TESTINGAPP)
	con.Subscribe("/obj/x*x/gamma", func(w http.ResponseWriter, r *http.Request) error {
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
	con.SetDeployment(TESTINGAPP)
	con.Subscribe("usererr", func(w http.ResponseWriter, r *http.Request) error {
		return errors.Newc(http.StatusBadRequest, "bad input")
	})
	con.Subscribe("err", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("it's bad")
	})
	con.Subscribe("panic", func(w http.ResponseWriter, r *http.Request) error {
		panic("it's really bad")
	})
	con.Subscribe("oserr", func(w http.ResponseWriter, r *http.Request) error {
		err := errors.Trace(os.ErrNotExist)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		return err
	})
	con.Subscribe("stillalive", func(w http.ResponseWriter, r *http.Request) error {
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
	alpha.SetDeployment(TESTINGAPP)
	alpha.SetPlane("alpha")
	alpha.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})

	beta := New("different.planes.connector")
	beta.SetDeployment(TESTINGAPP)
	beta.SetPlane("beta")
	beta.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
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
	con.SetDeployment(TESTINGAPP)

	// Subscribe before beta is started
	con.Subscribe("before", func(w http.ResponseWriter, r *http.Request) error {
		beforeCalled = true
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Subscribe after beta is started
	con.Subscribe("after", func(w http.ResponseWriter, r *http.Request) error {
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
	con.SetDeployment(TESTINGAPP)

	// Subscribe
	con.Subscribe("sub1", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	con.Subscribe("sub2", func(w http.ResponseWriter, r *http.Request) error {
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
	err = con.Unsubscribe(":443/sub1")
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
	alpha.SetDeployment(TESTINGAPP)
	alpha.Subscribe("https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	beta1 := New("beta.another.host.connector")
	beta1.SetDeployment(TESTINGAPP)
	beta1.Subscribe("https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	beta2 := New("beta.another.host.connector")
	beta2.SetDeployment(TESTINGAPP)
	beta2.Subscribe("https://alternative.host.connector/empty", func(w http.ResponseWriter, r *http.Request) error {
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
	con.SetDeployment(TESTINGAPP)
	con.Subscribe("/hello", func(w http.ResponseWriter, r *http.Request) error {
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

	err = con.Unsubscribe("/hello")
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
	con.SetDeployment(TESTINGAPP)

	start := make(chan bool)
	hold := make(chan bool)
	end := make(chan bool)
	con.Subscribe("/op", func(w http.ResponseWriter, r *http.Request) error {
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
