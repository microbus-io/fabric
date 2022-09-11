package connector

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartupShutdown(t *testing.T) {
	t.Parallel()

	var startupCalled, shutdownCalled bool

	alpha := NewConnector()
	alpha.SetHostName("alpha.startupshutdown.test")
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

func TestStartupError(t *testing.T) {
	t.Parallel()

	var startupCalled, shutdownCalled bool

	alpha := NewConnector()
	alpha.SetHostName("alpha.startuperror.test")
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
