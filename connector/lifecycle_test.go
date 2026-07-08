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
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"
)

func TestConnector_StartupShutdown(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	var startupCalled, shutdownCalled bool

	con := New("startup.shutdown.connector")
	con.SetOnStartup(func(ctx context.Context) error {
		startupCalled = true
		return nil
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	assert.False(startupCalled)
	assert.False(shutdownCalled)
	assert.False(con.IsStarted())

	err := con.Startup(ctx)
	assert.NoError(err)
	assert.True(startupCalled)
	assert.False(shutdownCalled)
	assert.True(con.IsStarted())

	err = con.Shutdown(ctx)
	assert.NoError(err)
	assert.True(startupCalled)
	assert.True(shutdownCalled)
	assert.False(con.IsStarted())
}

func TestConnector_StartupError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	var startupCalled, shutdownCalled bool

	con := New("startup.error.connector")
	con.SetOnStartup(func(ctx context.Context) error {
		startupCalled = true
		return errors.New("oops")
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	assert.False(startupCalled)
	assert.False(shutdownCalled)
	assert.False(con.IsStarted())

	err := con.Startup(ctx)
	assert.Error(err)
	assert.True(startupCalled)
	assert.True(shutdownCalled)
	assert.False(con.IsStarted())

	err = con.Shutdown(ctx)
	assert.Error(err)
	assert.True(startupCalled)
	assert.True(shutdownCalled)
	assert.False(con.IsStarted())
}

func TestConnector_StartupPanic(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("startup.panic.connector")
	con.SetOnStartup(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup(ctx)
	assert.Error(err)
	assert.Equal("really bad", err.Error())
}

func TestConnector_ShutdownPanic(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("shutdown.panic.connector")
	con.SetOnShutdown(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup(ctx)
	assert.NoError(err)
	err = con.Shutdown(ctx)
	assert.Error(err)
	assert.Equal("really bad", err.Error())
}

func TestConnector_StartupTimeout(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("startup.timeout.connector")

	done := make(chan bool)
	con.SetOnStartup(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	})

	go func() {
		err := con.Startup(ctx)
		assert.Error(err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	assert.False(con.IsStarted())
}

func TestConnector_ShutdownTimeout(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("shutdown.timeout.connector")

	done := make(chan bool)
	con.SetOnShutdown(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	})

	err := con.Startup(ctx)
	assert.NoError(err)
	assert.True(con.IsStarted())

	go func() {
		err := con.Shutdown(ctx)
		assert.Error(err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	assert.False(con.IsStarted())
}

func TestConnector_InitError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("init.error.connector")
	err := con.DefineConfig("Hundred", cfg.DefaultValue("101"), cfg.Validation("int [1,100]"))
	assert.Error(err)
	err = con.Startup(ctx)
	assert.Error(err)

	con = New("init.error.connector")
	err = con.DefineConfig("Hundred", cfg.DefaultValue("1"), cfg.Validation("int [1,100]"))
	assert.NoError(err)
	err = con.SetConfig("Hundred", "101")
	assert.Error(err)
	err = con.Startup(ctx)
	assert.Error(err)

	con = New("init.error.connector")
	err = con.Subscribe("BadPath",
		func(w http.ResponseWriter, r *http.Request) error {
			return nil
		},
		sub.At("GET", ":BAD/path"),
		sub.Web(),
	)
	assert.Error(err)
	err = con.Startup(ctx)
	assert.Error(err)

	con = New("init.error.connector")
	err = con.StartTicker("ticktock", -time.Minute, func(ctx context.Context) error {
		return nil
	})
	assert.Error(err)
	err = con.Startup(ctx)
	assert.Error(err)
}

func TestConnector_Restart(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	var startupCalled atomic.Int32
	var shutdownCalled atomic.Int32
	var endpointCalled atomic.Int32
	var tickerCalled atomic.Int32

	// Set up a configurator
	plane := utils.RandomIdentifier(12)
	configurator := New("configurator.core")
	configurator.SetDeployment(LAB) // Tickers and configs are disabled in TESTING
	configurator.SetPlane(plane)

	err := configurator.Startup(ctx)
	assert.NoError(err)
	defer configurator.Shutdown(ctx)

	// Set up the connector
	con := New("restart.connector")
	con.SetDeployment(LAB) // Tickers and configs are disabled in TESTING
	con.SetPlane(plane)
	con.SetOnStartup(func(ctx context.Context) error {
		startupCalled.Add(1)
		return nil
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled.Add(1)
		return nil
	})
	con.Subscribe("Endpoint",
		func(w http.ResponseWriter, r *http.Request) error {
			endpointCalled.Add(1)
			return nil
		},
		sub.At("GET", "/endpoint"),
		sub.Web(),
	)
	con.StartTicker("tick", time.Millisecond*500, func(ctx context.Context) error {
		tickerCalled.Add(1)
		return nil
	})
	con.DefineConfig("config", cfg.DefaultValue("default"))

	assert.Equal("default", con.configs["config"].Value)

	// Startup
	configurator.Subscribe("ValuesOverride",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte(`{"values":{"config":"overridden"}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)
	unsub := func() error { return configurator.Unsubscribe("ValuesOverride") }

	err = con.Startup(ctx)
	assert.NoError(err)
	assert.Equal(int32(1), startupCalled.Load())
	assert.Zero(shutdownCalled.Load())
	_, err = con.Request(ctx, pub.GET("https://restart.connector/endpoint"))
	assert.NoError(err)
	assert.Equal(int32(1), endpointCalled.Load())
	time.Sleep(time.Second)
	assert.True(tickerCalled.Load() > 0)
	assert.Equal("overridden", con.Config("config"))

	// Shutdown
	subCount := con.subs.Len()
	err = con.Shutdown(ctx)
	assert.NoError(err)
	assert.Equal(int32(1), shutdownCalled.Load())
	assert.Equal(subCount-2, con.subs.Len()) // 2 dlru subs unregistered by Cache.Close

	startupCalled.Store(0)
	shutdownCalled.Store(0)
	endpointCalled.Store(0)
	tickerCalled.Store(0)

	// Restart
	unsub()
	configurator.Subscribe("ValuesEmpty",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte(`{}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err = con.Startup(ctx)
	assert.NoError(err)
	assert.Equal(int32(1), startupCalled.Load())
	assert.Zero(shutdownCalled.Load())
	assert.Equal(subCount, con.subs.Len())

	_, err = con.Request(ctx, pub.GET("https://restart.connector/endpoint"))
	assert.NoError(err)
	assert.Equal(int32(1), endpointCalled.Load())
	time.Sleep(time.Second)
	assert.True(tickerCalled.Load() > 0)
	assert.Equal("default", con.Config("config"))

	// Shutdown
	err = con.Shutdown(ctx)
	assert.NoError(err)
	assert.Equal(int32(1), shutdownCalled.Load())
	assert.Equal(subCount-2, con.subs.Len()) // 2 dlru subs unregistered by Cache.Close
}

func TestConnector_GoGracefulShutdown(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)
	ctx := t.Context()

	con := New("go.graceful.shutdown.connector")
	err := con.Startup(ctx)
	assert.NoError(err)

	done500 := false
	con.Go(ctx, func(ctx context.Context) (err error) {
		time.Sleep(500 * time.Millisecond)
		done500 = true
		return nil
	})
	done300 := false
	con.Go(ctx, func(ctx context.Context) (err error) {
		time.Sleep(400 * time.Millisecond)
		done300 = true
		return nil
	})
	started := time.Now()
	err = con.Shutdown(ctx)
	assert.NoError(err)
	dur := time.Since(started)
	assert.True(dur >= 500*time.Millisecond)
	assert.True(done500)
	assert.True(done300)
}

func TestConnector_Parallel(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("parallel.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	j1 := false
	j2 := false
	j3 := false
	started := time.Now()
	err = con.Parallel(ctx,
		func(ctx context.Context) (err error) {
			time.Sleep(100 * time.Millisecond)
			j1 = true
			return nil
		},
		func(ctx context.Context) (err error) {
			time.Sleep(200 * time.Millisecond)
			j2 = true
			return nil
		},
		func(ctx context.Context) (err error) {
			time.Sleep(300 * time.Millisecond)
			j3 = true
			return nil
		},
	)
	dur := time.Since(started)
	assert.True(dur >= 300*time.Millisecond)
	assert.NoError(err)
	assert.True(j1)
	assert.True(j2)
	assert.True(j3)
}

func TestConnector_ParallelCancelOnError(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	con := New("parallel.cancel.on.error.connector")
	err := con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	canceled := false
	started := time.Now()
	err = con.Parallel(ctx,
		func(ctx context.Context) (err error) {
			return errors.New("oops")
		},
		func(ctx context.Context) (err error) {
			select {
			case <-ctx.Done():
				canceled = true
			case <-time.After(4 * time.Second):
			}
			return nil
		},
	)
	dur := time.Since(started)
	assert.Error(err, "oops")
	assert.True(canceled)
	assert.True(dur < 4*time.Second)
}
