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
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestConnector_StartupShutdown(t *testing.T) {
	t.Parallel()

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

	testarossa.False(t, startupCalled)
	testarossa.False(t, shutdownCalled)
	testarossa.False(t, con.IsStarted())

	err := con.Startup()
	testarossa.NoError(t, err)
	testarossa.True(t, startupCalled)
	testarossa.False(t, shutdownCalled)
	testarossa.True(t, con.IsStarted())

	err = con.Shutdown()
	testarossa.NoError(t, err)
	testarossa.True(t, startupCalled)
	testarossa.True(t, shutdownCalled)
	testarossa.False(t, con.IsStarted())
}

func TestConnector_StartupError(t *testing.T) {
	t.Parallel()

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

	testarossa.False(t, startupCalled)
	testarossa.False(t, shutdownCalled)
	testarossa.False(t, con.IsStarted())

	err := con.Startup()
	testarossa.Error(t, err)
	testarossa.True(t, startupCalled)
	testarossa.True(t, shutdownCalled)
	testarossa.False(t, con.IsStarted())

	err = con.Shutdown()
	testarossa.Error(t, err)
	testarossa.True(t, startupCalled)
	testarossa.True(t, shutdownCalled)
	testarossa.False(t, con.IsStarted())
}

func TestConnector_StartupPanic(t *testing.T) {
	t.Parallel()

	con := New("startup.panic.connector")
	con.SetOnStartup(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup()
	testarossa.Error(t, err)
	testarossa.Equal(t, "really bad", err.Error())
}

func TestConnector_ShutdownPanic(t *testing.T) {
	t.Parallel()

	con := New("shutdown.panic.connector")
	con.SetOnShutdown(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup()
	testarossa.NoError(t, err)
	err = con.Shutdown()
	testarossa.Error(t, err)
	testarossa.Equal(t, "really bad", err.Error())
}

func TestConnector_StartupTimeout(t *testing.T) {
	t.Parallel()

	con := New("startup.timeout.connector")

	done := make(chan bool)
	con.SetOnStartup(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	})

	go func() {
		err := con.Startup()
		testarossa.Error(t, err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	testarossa.False(t, con.IsStarted())
}

func TestConnector_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	con := New("shutdown.timeout.connector")

	done := make(chan bool)
	con.SetOnShutdown(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	})

	err := con.Startup()
	testarossa.NoError(t, err)
	testarossa.True(t, con.IsStarted())

	go func() {
		err := con.Shutdown()
		testarossa.Error(t, err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	testarossa.False(t, con.IsStarted())
}

func TestConnector_InitError(t *testing.T) {
	t.Parallel()

	con := New("init.error.connector")
	err := con.DefineConfig("Hundred", cfg.DefaultValue("101"), cfg.Validation("int [1,100]"))
	testarossa.Error(t, err)
	err = con.Startup()
	testarossa.Error(t, err)

	con = New("init.error.connector")
	err = con.DefineConfig("Hundred", cfg.DefaultValue("1"), cfg.Validation("int [1,100]"))
	testarossa.NoError(t, err)
	err = con.SetConfig("Hundred", "101")
	testarossa.Error(t, err)
	err = con.Startup()
	testarossa.Error(t, err)

	con = New("init.error.connector")
	err = con.Subscribe("GET", ":BAD/path", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	testarossa.Error(t, err)
	err = con.Startup()
	testarossa.Error(t, err)

	con = New("init.error.connector")
	err = con.StartTicker("ticktock", -time.Minute, func(ctx context.Context) error {
		return nil
	})
	testarossa.Error(t, err)
	err = con.Startup()
	testarossa.Error(t, err)
}

func TestConnector_Restart(t *testing.T) {
	t.Parallel()

	startupCalled := 0
	shutdownCalled := 0
	endpointCalled := 0
	tickerCalled := 0

	// Set up a configurator
	plane := rand.AlphaNum64(12)
	configurator := New("configurator.core")
	configurator.SetDeployment(LAB) // Tickers and configs are disabled in TESTING
	configurator.SetPlane(plane)

	err := configurator.Startup()
	testarossa.NoError(t, err)
	defer configurator.Shutdown()

	// Set up the connector
	con := New("restart.connector")
	con.SetDeployment(LAB) // Tickers and configs are disabled in TESTING
	con.SetPlane(plane)
	con.SetOnStartup(func(ctx context.Context) error {
		startupCalled++
		return nil
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled++
		return nil
	})
	con.Subscribe("GET", "/endpoint", func(w http.ResponseWriter, r *http.Request) error {
		endpointCalled++
		return nil
	})
	con.StartTicker("tick", time.Millisecond*500, func(ctx context.Context) error {
		tickerCalled++
		return nil
	})
	con.DefineConfig("config", cfg.DefaultValue("default"))

	testarossa.Equal(t, "default", con.configs["config"].Value)

	// Startup
	configurator.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(`{"values":{"config":"overriden"}}`))
		return nil
	})
	err = con.Startup()
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, startupCalled)
	testarossa.Zero(t, shutdownCalled)
	_, err = con.Request(con.lifetimeCtx, pub.GET("https://restart.connector/endpoint"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, endpointCalled)
	time.Sleep(time.Second)
	testarossa.True(t, tickerCalled > 0)
	testarossa.Equal(t, "overriden", con.Config("config"))

	// Shutdown
	err = con.Shutdown()
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, shutdownCalled)

	// Restart
	configurator.Unsubscribe("POST", "/values")
	configurator.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(`{}`))
		return nil
	})
	startupCalled = 0
	shutdownCalled = 0
	endpointCalled = 0
	tickerCalled = 0

	err = con.Startup()
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, startupCalled)
	testarossa.Zero(t, shutdownCalled)
	_, err = con.Request(con.lifetimeCtx, pub.GET("https://restart.connector/endpoint"))
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, endpointCalled)
	time.Sleep(time.Second)
	testarossa.True(t, tickerCalled > 0)
	testarossa.Equal(t, "default", con.Config("config"))

	// Shutdown
	err = con.Shutdown()
	testarossa.NoError(t, err)
	testarossa.Equal(t, 1, shutdownCalled)
}

func TestConnector_GoGracefulShutdown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	con := New("go.graceful.shutdown.connector")
	err := con.Startup()
	testarossa.NoError(t, err)

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
	err = con.Shutdown()
	testarossa.NoError(t, err)
	dur := time.Since(started)
	testarossa.True(t, dur >= 500*time.Millisecond)
	testarossa.True(t, done500)
	testarossa.True(t, done300)
}

func TestConnector_Parallel(t *testing.T) {
	t.Parallel()

	con := New("parallel.connector")
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	j1 := false
	j2 := false
	j3 := false
	started := time.Now()
	err = con.Parallel(
		func() (err error) {
			time.Sleep(100 * time.Millisecond)
			j1 = true
			return nil
		},
		func() (err error) {
			time.Sleep(200 * time.Millisecond)
			j2 = true
			return nil
		},
		func() (err error) {
			time.Sleep(300 * time.Millisecond)
			j3 = true
			return nil
		},
	)
	dur := time.Since(started)
	testarossa.True(t, dur >= 300*time.Millisecond)
	testarossa.NoError(t, err)
	testarossa.True(t, j1)
	testarossa.True(t, j2)
	testarossa.True(t, j3)
}
