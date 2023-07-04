/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"testing"

	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestConnector_Ping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("ping.connector")
	con.SetPlane(rand.AlphaNum64(12))

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send messages
	for r := range con.Publish(ctx, pub.GET("https://ping.connector:888/ping")) {
		_, err := r.Get()
		assert.NoError(t, err)
	}
	for r := range con.Publish(ctx, pub.GET("https://"+con.id+".ping.connector:888/ping")) {
		_, err := r.Get()
		assert.NoError(t, err)
	}
	for r := range con.Publish(ctx, pub.GET("https://all:888/ping")) {
		_, err := r.Get()
		assert.NoError(t, err)
	}
	for r := range con.Publish(ctx, pub.GET("https://"+con.id+".all:888/ping")) {
		_, err := r.Get()
		assert.NoError(t, err)
	}
}
