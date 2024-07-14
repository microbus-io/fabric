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

package metrics

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *metricsapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc,
	)
	if err != nil {
		return err
	}
	return nil
}

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
	return nil
}

func TestMetrics_Collect(t *testing.T) {
	t.Parallel()

	ctx := Context()
	Collect_Get(t, ctx, "").
		// All three services should be detected
		BodyContains("metrics.core").
		BodyNotContains("one.collect").
		BodyNotContains("two.collect")

	// Join two new services
	con1 := connector.New("one.collect")
	con1.SetOnStartup(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	con1.Subscribe("GET", "/ten", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("1234567890"))
		return nil
	})
	con2 := connector.New("two.collect")

	err := App.AddAndStartup(con1, con2)
	testarossa.NoError(t, err)
	defer con1.Shutdown()
	defer con2.Shutdown()

	// Make a request to the service
	_, err = con1.GET(ctx, "https://one.collect/ten")
	testarossa.NoError(t, err)

	// Interact with the cache
	con1.DistribCache().Store(ctx, "A", []byte("1234567890"))
	con1.DistribCache().Load(ctx, "A")
	con1.DistribCache().Load(ctx, "B")

	// Loop until the new services are discovered
	for {
		tc := Collect_Get(t, ctx, "")
		res, err := tc.Get()
		testarossa.NoError(t, err)
		body, err := io.ReadAll(res.Body)
		testarossa.NoError(t, err)
		if bytes.Contains(body, []byte("metrics.core")) &&
			bytes.Contains(body, []byte("one.collect")) &&
			bytes.Contains(body, []byte("two.collect")) {
			break
		}
	}

	Collect_Get(t, ctx, "").
		// All three services should be detected
		BodyContains("metrics.core").
		BodyContains("one.collect").
		BodyContains("two.collect").
		// The startup callback should take between 100ms and 500ms
		BodyContains(`microbus_callback_duration_seconds_bucket{error="OK",handler="startup",id="` + con1.ID() + `",service="one.collect",ver="0",le="0.1"} 0`).
		BodyContains(`microbus_callback_duration_seconds_bucket{error="OK",handler="startup",id="` + con1.ID() + `",service="one.collect",ver="0",le="0.5"} 1`).
		BodyContains(`microbus_log_messages_total{id="` + con1.ID() + `",message="Startup",service="one.collect",severity="INFO",ver="0"} 1`).
		BodyContains(`microbus_uptime_duration_seconds_total{id="` + con1.ID() + `",service="one.collect",ver="0"}`).
		// Cache should have 1 element of 10 bytes
		BodyContains(`microbus_cache_weight_total{id="` + con1.ID() + `",service="one.collect",ver="0"} 10`).
		BodyContains(`microbus_cache_len_total{id="` + con1.ID() + `",service="one.collect",ver="0"} 1`).
		BodyContains(`microbus_cache_misses_total{id="` + con1.ID() + `",service="one.collect",ver="0"} 1`).
		BodyContains(`microbus_cache_hits_total{id="` + con1.ID() + `",service="one.collect",ver="0"} 1`).
		BodyContains(`microbus_request_count_total{code="404",error="OK",host="one.collect",id="` + con1.ID() + `",method="GET",port="888",service="one.collect",ver="0"} 2`).
		// The response size is 10 bytes
		BodyContains(`microbus_response_size_bytes_sum{code="200",error="OK",handler="one.collect:443/ten",id="` + con1.ID() + `",method="GET",port="443",service="one.collect",ver="0"} 10`).
		BodyContains(`microbus_response_size_bytes_count{code="200",error="OK",handler="one.collect:443/ten",id="` + con1.ID() + `",method="GET",port="443",service="one.collect",ver="0"} 1`).
		// The request should take between 100ms and 500ms
		BodyContains(`microbus_request_count_total{code="200",error="OK",host="one.collect",id="` + con1.ID() + `",method="GET",port="443",service="one.collect",ver="0"} 1`).
		BodyContains(`microbus_response_duration_seconds_bucket{code="200",error="OK",handler="one.collect:443/ten",id="` + con1.ID() + `",method="GET",port="443",service="one.collect",ver="0",le="0.1"} 0`).
		BodyContains(`microbus_response_duration_seconds_bucket{code="200",error="OK",handler="one.collect:443/ten",id="` + con1.ID() + `",method="GET",port="443",service="one.collect",ver="0",le="0.5"} 1`).
		// Acks should be logged
		BodyContains("microbus_ack_duration_seconds_bucket")
}

func TestMetrics_GZip(t *testing.T) {
	t.Parallel()

	r, _ := http.NewRequest("GET", "", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	Collect(t, r).
		Assert(func(t *testing.T, res *http.Response, err error) {
			testarossa.NoError(t, err)
			testarossa.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
			unzipper, err := gzip.NewReader(res.Body)
			testarossa.NoError(t, err)
			body, err := io.ReadAll(unzipper)
			unzipper.Close()
			testarossa.NoError(t, err)
			testarossa.True(t, bytes.Contains(body, []byte("microbus_log_messages_total")))
		})
}

func TestMetrics_SecretKey(t *testing.T) {
	// No parallel
	ctx := Context()
	Svc.SetSecretKey("secret1234")
	Collect_Get(t, ctx, "").
		Error("incorrect secret key").
		ErrorCode(http.StatusNotFound)
	Svc.SetSecretKey("")
	Collect_Get(t, ctx, "").NoError()
}
