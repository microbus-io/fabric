package connector

import (
	"context"
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/stretchr/testify/assert"
)

func TestConnector_StartupShutdown(t *testing.T) {
	t.Parallel()

	var startupCalled, shutdownCalled bool

	alpha := NewConnector()
	alpha.SetHostName("alpha.startupshutdown.connector")
	alpha.SetOnStartup(func(ctx context.Context) error {
		startupCalled = true
		return nil
	})
	alpha.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	assert.False(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.False(t, alpha.IsStarted())

	err := alpha.Startup()
	assert.NoError(t, err)
	assert.True(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.True(t, alpha.IsStarted())

	err = alpha.Shutdown()
	assert.NoError(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, alpha.IsStarted())
}

func TestConnector_StartupError(t *testing.T) {
	t.Parallel()

	var startupCalled, shutdownCalled bool

	alpha := NewConnector()
	alpha.SetHostName("alpha.startuperror.connector")
	alpha.SetOnStartup(func(ctx context.Context) error {
		startupCalled = true
		return errors.New("oops")
	})
	alpha.SetOnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	assert.False(t, startupCalled)
	assert.False(t, shutdownCalled)
	assert.False(t, alpha.IsStarted())

	err := alpha.Startup()
	assert.Error(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, alpha.IsStarted())

	err = alpha.Shutdown()
	assert.Error(t, err)
	assert.True(t, startupCalled)
	assert.True(t, shutdownCalled)
	assert.False(t, alpha.IsStarted())
}

func TestConnector_StartupPanic(t *testing.T) {
	t.Parallel()

	alpha := NewConnector()
	alpha.SetHostName("startuppanic.connector")
	alpha.SetOnStartup(func(ctx context.Context) error {
		panic("really bad")
	})
	err := alpha.Startup()
	assert.Error(t, err)
	assert.Equal(t, "really bad", err.Error())
}

func TestConnector_ShutdownPanic(t *testing.T) {
	t.Parallel()

	alpha := NewConnector()
	alpha.SetHostName("shutdownpanic.connector")
	alpha.SetOnShutdown(func(ctx context.Context) error {
		panic("really bad")
	})
	err := alpha.Startup()
	assert.NoError(t, err)
	err = alpha.Shutdown()
	assert.Error(t, err)
	assert.Equal(t, "really bad", err.Error())
}
