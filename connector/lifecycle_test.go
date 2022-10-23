package connector

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/errors"
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
