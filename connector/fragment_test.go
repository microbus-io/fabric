/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"
)

func TestConnector_Frag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	var bodyReceived []byte
	con := New("frag.connector")
	con.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		bodyReceived, err = io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		w.Write(bodyReceived)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()
	con.maxFragmentSize = 128

	// Prepare the body to send
	bodySent := []byte(rand.AlphaNum64(int(con.maxFragmentSize)*2 + 16))

	// Send message and validate that it was received whole
	res, err := con.POST(ctx, "https://frag.connector/big", bodySent)
	if testarossa.NoError(t, err) {
		testarossa.SliceEqual(t, bodySent, bodyReceived)
		bodyResponded, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.SliceEqual(t, bodySent, bodyResponded)
		}
	}
}

func TestConnector_FragMulticast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	var alphaBodyReceived []byte
	alpha := New("frag.multicast.connector")
	alpha.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		alphaBodyReceived, err = io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		w.Write(alphaBodyReceived)
		return nil
	}, sub.NoQueue())

	var betaBodyReceived []byte
	beta := New("frag.multicast.connector")
	beta.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		betaBodyReceived, err = io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		w.Write(betaBodyReceived)
		return nil
	}, sub.NoQueue())

	// Startup the microservice
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
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
		testarossa.NoError(t, err)
		bodyResponded, err := io.ReadAll(res.Body)
		testarossa.NoError(t, err)
		testarossa.SliceEqual(t, bodySent, bodyResponded)
	}
	testarossa.SliceEqual(t, bodySent, alphaBodyReceived)
	testarossa.SliceEqual(t, bodySent, betaBodyReceived)
}

func TestConnector_FragLoadBalanced(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	var alphaBodyReceived []byte
	alpha := New("frag.load.balanced.connector")
	alpha.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		alphaBodyReceived, err = io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		w.Write(alphaBodyReceived)
		return nil
	}, sub.LoadBalanced())

	var betaBodyReceived []byte
	beta := New("frag.load.balanced.connector")
	beta.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		var err error
		betaBodyReceived, err = io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		w.Write(betaBodyReceived)
		return nil
	}, sub.LoadBalanced())

	// Startup the microservice
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
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
		if testarossa.NoError(t, err) {
			bodyResponded, err := io.ReadAll(res.Body)
			if testarossa.NoError(t, err) {
				testarossa.SliceEqual(t, bodySent, bodyResponded)
			}
		}
	}
	if alphaBodyReceived != nil {
		testarossa.SliceEqual(t, bodySent, alphaBodyReceived)
		testarossa.Nil(t, betaBodyReceived)
	} else {
		testarossa.SliceEqual(t, bodySent, betaBodyReceived)
		testarossa.Nil(t, alphaBodyReceived)
	}
}

func BenchmarkConnector_Frag(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	con := New("frag.benchmark.connector")
	con.Subscribe("POST", "big", func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		testarossa.NoError(b, err)
		w.Write(body)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	testarossa.NoError(b, err)
	defer con.Shutdown()

	// Prepare the body to send
	payload := []byte(rand.AlphaNum64(16 * 1024 * 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Send message and validate that it was received whole
		res, err := con.POST(ctx, "https://frag.benchmark.connector/big", payload)
		testarossa.NoError(b, err)
		_, err = io.ReadAll(res.Body)
		testarossa.NoError(b, err)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 16":
	// 16MB payload: 24 ms/op
	// 32MB payload: 45 ms/op
	// 64MB payload: 85 ms/op
	// 128MB payload: 182 ms/op
	// 256MB payload: 375 ms/op
}

func TestConnector_DefragRequest(t *testing.T) {
	t.Parallel()

	con := New("defrag.request.connector")
	makeChunk := func(msgID string, fragIndex int, fragMax int, content string) *http.Request {
		r, err := http.NewRequest("GET", "", strings.NewReader(content))
		testarossa.NoError(t, err)
		f := frame.Of(r)
		f.SetFromID("12345678")
		f.SetMessageID(msgID)
		f.SetFragment(fragIndex, fragMax)
		return r
	}

	// One chunk only: should return the exact same object
	r := makeChunk("one", 1, 1, strings.Repeat("1", 1024))
	integrated, err := con.defragRequest(r)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, r, integrated)
	}

	// Three chunks: should return the integrated chunk after the final chunk
	r = makeChunk("three", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragRequest(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("three", 2, 3, strings.Repeat("2", 1024))
	integrated, err = con.defragRequest(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("three", 3, 3, strings.Repeat("3", 1024))
	integrated, err = con.defragRequest(r)
	if testarossa.NoError(t, err) && testarossa.NotNil(t, integrated) {
		body, err := io.ReadAll(integrated.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, strings.Repeat("1", 1024)+strings.Repeat("2", 1024)+strings.Repeat("3", 1024), string(body))
		}
	}

	// Three chunks not in order: should return the integrated chunk after the final chunk
	r = makeChunk("outoforder", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragRequest(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("outoforder", 3, 3, strings.Repeat("3", 1024))
	integrated, err = con.defragRequest(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("outoforder", 2, 3, strings.Repeat("2", 1024))
	integrated, err = con.defragRequest(r)
	if testarossa.NoError(t, err) && testarossa.NotNil(t, integrated) {
		body, err := io.ReadAll(integrated.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, strings.Repeat("1", 1024)+strings.Repeat("2", 1024)+strings.Repeat("3", 1024), string(body))
		}
	}

	// Missing the first chunk: should fail
	r = makeChunk("missingone", 2, 3, strings.Repeat("2", 1024))
	_, err = con.defragRequest(r)
	testarossa.Error(t, err)

	// Taking too long: should timeout
	r = makeChunk("delayed", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragRequest(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	time.Sleep(con.networkHop * 2)
	r = makeChunk("delayed", 2, 3, strings.Repeat("2", 1024))
	_, err = con.defragRequest(r)
	testarossa.Error(t, err)
	r = makeChunk("delayed", 3, 3, strings.Repeat("3", 1024))
	_, err = con.defragRequest(r)
	testarossa.Error(t, err)
}

func TestConnector_DefragResponse(t *testing.T) {
	t.Parallel()

	con := New("defrag.response.connector")
	makeChunk := func(msgID string, fragIndex int, fragMax int, content string) *http.Response {
		w := httptest.NewRecorder()
		f := frame.Of(w)
		f.SetFromID("12345678")
		f.SetMessageID(msgID)
		f.SetFragment(fragIndex, fragMax)
		w.Write([]byte(content))
		return w.Result()
	}

	// One chunk only: should return the exact same object
	r := makeChunk("one", 1, 1, strings.Repeat("1", 1024))
	integrated, err := con.defragResponse(r)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, r, integrated)
	}

	// Three chunks: should return the integrated chunk after the final chunk
	r = makeChunk("three", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragResponse(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("three", 2, 3, strings.Repeat("2", 1024))
	integrated, err = con.defragResponse(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("three", 3, 3, strings.Repeat("3", 1024))
	integrated, err = con.defragResponse(r)
	if testarossa.NoError(t, err) && testarossa.NotNil(t, integrated) {
		body, err := io.ReadAll(integrated.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, strings.Repeat("1", 1024)+strings.Repeat("2", 1024)+strings.Repeat("3", 1024), string(body))
		}
	}

	// Three chunks not in order: should return the integrated chunk after the final chunk
	r = makeChunk("outoforder", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragResponse(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("outoforder", 3, 3, strings.Repeat("3", 1024))
	integrated, err = con.defragResponse(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	r = makeChunk("outoforder", 2, 3, strings.Repeat("2", 1024))
	integrated, err = con.defragResponse(r)
	if testarossa.NoError(t, err) && testarossa.NotNil(t, integrated) {
		body, err := io.ReadAll(integrated.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, strings.Repeat("1", 1024)+strings.Repeat("2", 1024)+strings.Repeat("3", 1024), string(body))
		}
	}

	// Missing the first chunk: should fail
	r = makeChunk("missingone", 2, 3, strings.Repeat("2", 1024))
	_, err = con.defragResponse(r)
	testarossa.Error(t, err)

	// Taking too long: should timeout
	r = makeChunk("delayed", 1, 3, strings.Repeat("1", 1024))
	integrated, err = con.defragResponse(r)
	testarossa.NoError(t, err)
	testarossa.Nil(t, integrated)
	time.Sleep(con.networkHop * 2)
	r = makeChunk("delayed", 2, 3, strings.Repeat("2", 1024))
	_, err = con.defragResponse(r)
	testarossa.Error(t, err)
	r = makeChunk("delayed", 3, 3, strings.Repeat("3", 1024))
	_, err = con.defragResponse(r)
	testarossa.Error(t, err)
}
