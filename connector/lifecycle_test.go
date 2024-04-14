/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
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

	assert.False(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.False(t, con.IsStarted())

	err := con.Startup()
	assert.NoError(t, err)
	assert.True(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.True(t, con.IsStarted())

	err = con.Shutdown()
	assert.NoError(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, con.IsStarted())
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

	assert.False(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.False(t, con.IsStarted())

	err := con.Startup()
	assert.Error(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, con.IsStarted())

	err = con.Shutdown()
	assert.Error(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, con.IsStarted())
}

func TestConnector_StartupPanic(t *testing.T) {
	t.Parallel()

	con := New("startup.panic.connector")
	con.SetOnStartup(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup()
	assert.Error(t, err)
	assert.Equal(t, "really bad", err.Error())
}

func TestConnector_ShutdownPanic(t *testing.T) {
	t.Parallel()

	con := New("shutdown.panic.connector")
	con.SetOnShutdown(func(ctx context.Context) error {
		panic("really bad")
	})
	err := con.Startup()
	assert.NoError(t, err)
	err = con.Shutdown()
	assert.Error(t, err)
	assert.Equal(t, "really bad", err.Error())
}

func TestConnector_StartupTimeout(t *testing.T) {
	t.Parallel()

	con := New("startup.timeout.connector")

	done := make(chan bool)
	con.SetOnStartup(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, cb.TimeBudget(500*time.Millisecond))

	go func() {
		err := con.Startup()
		assert.Error(t, err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	assert.False(t, con.IsStarted())
}

func TestConnector_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	con := New("shutdown.timeout.connector")

	done := make(chan bool)
	con.SetOnShutdown(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, cb.TimeBudget(500*time.Millisecond))

	err := con.Startup()
	assert.NoError(t, err)
	assert.True(t, con.IsStarted())

	go func() {
		err := con.Shutdown()
		assert.Error(t, err)
		done <- true
	}()
	time.Sleep(600 * time.Millisecond)
	<-done
	assert.False(t, con.IsStarted())
}

func TestConnector_InitError(t *testing.T) {
	t.Parallel()

	con := New("init.error.connector")
	err := con.DefineConfig("Hundred", cfg.DefaultValue("101"), cfg.Validation("int [1,100]"))
	assert.Error(t, err)
	err = con.Startup()
	assert.Error(t, err)

	con = New("init.error.connector")
	err = con.DefineConfig("Hundred", cfg.DefaultValue("1"), cfg.Validation("int [1,100]"))
	assert.NoError(t, err)
	err = con.SetConfig("Hundred", "101")
	assert.Error(t, err)
	err = con.Startup()
	assert.Error(t, err)

	con = New("init.error.connector")
	err = con.Subscribe(":99999/path", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	assert.Error(t, err)
	err = con.Startup()
	assert.Error(t, err)

	con = New("init.error.connector")
	err = con.StartTicker("ticktock", -time.Minute, func(ctx context.Context) error {
		return nil
	})
	assert.Error(t, err)
	err = con.Startup()
	assert.Error(t, err)
}

func TestConnector_Restart(t *testing.T) {
	t.Parallel()

	startupCalled := 0
	shutdownCalled := 0
	endpointCalled := 0
	tickerCalled := 0

	// Set up a configurator
	plane := rand.AlphaNum64(12)
	configurator := New("configurator.sys")
	configurator.SetPlane(plane)

	err := configurator.Startup()
	assert.NoError(t, err)
	defer configurator.Shutdown()

	// Set up the connector
	con := New("restart.connector")
	con.SetPlane(plane)
	con.SetOnStartup(func(ctx context.Context) error {
		startupCalled++
		return nil
	})
	con.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled++
		return nil
	})
	con.Subscribe("/endpoint", func(w http.ResponseWriter, r *http.Request) error {
		endpointCalled++
		return nil
	})
	con.StartTicker("tick", time.Millisecond*500, func(ctx context.Context) error {
		tickerCalled++
		return nil
	})
	con.DefineConfig("config", cfg.DefaultValue("default"))

	assert.Equal(t, "default", con.configs["config"].Value)

	// Startup
	configurator.Subscribe("/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(`{"values":{"config":"overriden"}}`))
		return nil
	})
	err = con.Startup()
	assert.NoError(t, err)
	assert.Equal(t, 1, startupCalled)
	assert.Equal(t, 0, shutdownCalled)
	_, err = con.Request(con.lifetimeCtx, pub.GET("https://restart.connector/endpoint"))
	assert.NoError(t, err)
	assert.Equal(t, 1, endpointCalled)
	time.Sleep(time.Second)
	assert.True(t, tickerCalled > 0)
	assert.Equal(t, "overriden", con.Config("config"))

	// Shutdown
	err = con.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, 1, shutdownCalled)

	// Restart
	configurator.Unsubscribe("/values")
	configurator.Subscribe("/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(`{}`))
		return nil
	})
	startupCalled = 0
	shutdownCalled = 0
	endpointCalled = 0
	tickerCalled = 0

	err = con.Startup()
	assert.NoError(t, err)
	assert.Equal(t, 1, startupCalled)
	assert.Equal(t, 0, shutdownCalled)
	_, err = con.Request(con.lifetimeCtx, pub.GET("https://restart.connector/endpoint"))
	assert.NoError(t, err)
	assert.Equal(t, 1, endpointCalled)
	time.Sleep(time.Second)
	assert.True(t, tickerCalled > 0)
	assert.Equal(t, "default", con.Config("config"))

	// Shutdown
	err = con.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, 1, shutdownCalled)
}
