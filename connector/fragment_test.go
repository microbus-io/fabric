/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/stretchr/testify/assert"
)

func TestConnector_Frag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var bodyReceived []byte
	con := New("frag.connector")
	con.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		bodyReceived, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Write(bodyReceived)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()
	con.maxFragmentSize = 128

	// Prepare the body to send
	bodySent := []byte(rand.AlphaNum64(int(con.maxFragmentSize)*2 + 16))

	// Send message and validate that it was received whole
	res, err := con.POST(ctx, "https://frag.connector/big", bodySent)
	assert.NoError(t, err)
	bodyResponded, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, bodySent, bodyReceived)
	assert.Equal(t, bodySent, bodyResponded)
}

func TestConnector_FragMulticast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	var alphaBodyReceived []byte
	alpha := New("frag.multicast.connector")
	alpha.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		alphaBodyReceived, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Write(alphaBodyReceived)
		return nil
	}, sub.NoQueue())

	var betaBodyReceived []byte
	beta := New("frag.multicast.connector")
	beta.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		betaBodyReceived, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Write(betaBodyReceived)
		return nil
	}, sub.NoQueue())

	// Startup the microservice
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	alpha.maxFragmentSize = 1024
	beta.maxFragmentSize = 1024

	// Prepare the body to send
	bodySent := []byte(rand.AlphaNum64(int(alpha.maxFragmentSize)*2 + 16))

	// Send message and validate that it was received whole
	ch := alpha.Publish(
		ctx,
		pub.POST("https://frag.multicast.connector/big"),
		pub.Body(bodySent),
	)
	for r := range ch {
		res, err := r.Get()
		assert.NoError(t, err)
		bodyResponded, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, bodySent, bodyResponded)
	}
	assert.Equal(t, bodySent, alphaBodyReceived)
	assert.Equal(t, bodySent, betaBodyReceived)
}

func TestConnector_FragLoadBalanced(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	var alphaBodyReceived []byte
	alpha := New("frag.load.balanced.connector")
	alpha.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		alphaBodyReceived, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Write(alphaBodyReceived)
		return nil
	}, sub.LoadBalanced())

	var betaBodyReceived []byte
	beta := New("frag.load.balanced.connector")
	beta.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		betaBodyReceived, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Write(betaBodyReceived)
		return nil
	}, sub.LoadBalanced())

	// Startup the microservice
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	alpha.maxFragmentSize = 128
	beta.maxFragmentSize = 128

	// Prepare the body to send
	bodySent := []byte(rand.AlphaNum64(int(alpha.maxFragmentSize)*2 + 16))

	// Send message and validate that it was received whole
	ch := alpha.Publish(
		ctx,
		pub.POST("https://frag.load.balanced.connector/big"),
		pub.Body(bodySent),
	)
	for r := range ch {
		res, err := r.Get()
		if assert.NoError(t, err) {
			bodyResponded, err := io.ReadAll(res.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, bodySent, bodyResponded)
			}
		}
	}
	if alphaBodyReceived != nil {
		assert.Equal(t, bodySent, alphaBodyReceived)
		assert.Nil(t, betaBodyReceived)
	} else {
		assert.Equal(t, bodySent, betaBodyReceived)
		assert.Nil(t, alphaBodyReceived)
	}
}

func BenchmarkConnector_Frag(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	con := New("frag.benchmark.connector")
	con.Subscribe("big", func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		assert.NoError(b, err)
		w.Write(body)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(b, err)
	defer con.Shutdown()

	// Prepare the body to send
	payload := []byte(rand.AlphaNum64(16 * 1024 * 1024))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Send message and validate that it was received whole
		res, err := con.POST(ctx, "https://frag.benchmark.connector/big", payload)
		assert.NoError(b, err)
		_, err = io.ReadAll(res.Body)
		assert.NoError(b, err)
	}

	b.StopTimer()

	// On 2021 MacBook Pro M1 16":
	// 16MB payload: 20 ms/op
	// 32MB payload: 38 ms/op
	// 64MB payload: 75 ms/op
	// 128MB payload: 145 ms/op
	// 256MB payload: 300 ms/op
}
