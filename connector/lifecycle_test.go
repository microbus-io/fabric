package connector

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/clock"
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

	mockClock := clock.NewMock()

	con := New("startup.timeout.connector")
	con.SetClock(mockClock)

	step := make(chan bool)
	con.SetOnStartup(func(ctx context.Context) error {
		step <- true
		<-ctx.Done()
		return nil
	}, cb.TimeBudget(time.Second*10))

	go func() {
		err := con.Startup()
		assert.NoError(t, err)
		step <- true
	}()
	<-step
	mockClock.Add(time.Second * 15)
	<-step
	assert.True(t, con.IsStarted())

	con.Shutdown()
}

func TestConnector_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMock()

	con := New("shutdown.timeout.connector")
	con.SetClock(mockClock)

	step := make(chan bool)
	con.SetOnShutdown(func(ctx context.Context) error {
		step <- true
		<-ctx.Done()
		return nil
	}, cb.TimeBudget(time.Second*10))

	err := con.Startup()
	assert.NoError(t, err)
	assert.True(t, con.IsStarted())

	go func() {
		err := con.Shutdown()
		assert.NoError(t, err)
		step <- true
	}()
	<-step
	mockClock.Add(time.Second * 15)
	<-step
	assert.False(t, con.IsStarted())
}
