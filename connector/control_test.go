/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
