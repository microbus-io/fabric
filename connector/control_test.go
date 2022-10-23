package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_Ping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("ping.connector")

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	_, err = con.GET(ctx, "https://ping.connector:888/ping")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".ping.connector:888/ping")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://all:888/ping")
	assert.NoError(t, err)
	_, err = con.GET(ctx, "https://"+con.id+".all:888/ping")
	assert.NoError(t, err)
}
