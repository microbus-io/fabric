package connector

import (
	"context"
	"testing"

	stderrors "errors"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/stretchr/testify/assert"
)

func TestConnector_Log(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := NewConnector()
	service.SetHostName("logservice.connector")
	assert.False(t, service.IsStarted())

	// Logger not initialized, should not log.
	service.LogDebug(ctx, "This is a log debug message", log.String("someStr", "some string"))
	service.LogInfo(ctx, "This is a log info message", log.String("someStr", "some string"))
	service.LogWarn(ctx, "This is a log warn message", stderrors.New("failed"), log.String("someStr", "some string"))
	service.LogError(ctx, "This is a log error message", stderrors.New("failed"), log.String("someStr", "some string"))

	err := service.Startup()
	assert.NoError(t, err)
	assert.True(t, service.IsStarted())

	// Logger initialized, should log.
	service.LogDebug(ctx, "This is a log debug message", log.String("someStr", "some string"))
	service.LogInfo(ctx, "This is a log info message", log.String("someStr", "some string"))
	service.LogWarn(ctx, "This is a log warn message", errors.New("failed"), log.String("someStr", "some string"))
	service.LogError(ctx, "This is a log error message", errors.New("failed"), log.String("someStr", "some string"))

	err = service.Shutdown()
	assert.NoError(t, err)
}
