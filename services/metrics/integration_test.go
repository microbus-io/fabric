package metrics

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/services/metrics/metricsapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *metricsapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc.With(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestMetrics_Collect(t *testing.T) {
	t.Parallel()

	ctx := Context()

	// Join two new services
	con1 := connector.New("one.collect")
	con1.SetOnStartup(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	con1.Subscribe("/ten", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("1234567890"))
		return nil
	})
	con2 := connector.New("two.collect")

	App.Join(con1, con2)
	err := con1.Startup()
	assert.NoError(t, err)
	defer con1.Shutdown()
	err = con2.Startup()
	assert.NoError(t, err)
	defer con2.Shutdown()

	// Make a request to the service
	_, err = con1.GET(ctx, "https://one.collect/ten")
	assert.NoError(t, err)

	// Interact with the cache
	con1.DistribCache().Store(ctx, "A", []byte("1234567890"))
	con1.DistribCache().Load(ctx, "A")
	con1.DistribCache().Load(ctx, "B")

	Collect(ctx).
		// All three services should be detected
		BodyContains(t, "metrics.sys").
		BodyContains(t, "one.collect").
		BodyContains(t, "two.collect").
		// The startup callback should take between 100ms and 250ms
		BodyContains(t, `microbus_callback_duration_seconds_bucket{error="false",handler="onstartup",id="`+con1.ID()+`",service="one.collect",ver="0",le="0.1"} 0`).
		BodyContains(t, `microbus_callback_duration_seconds_bucket{error="false",handler="onstartup",id="`+con1.ID()+`",service="one.collect",ver="0",le="0.5"} 1`).
		BodyContains(t, `microbus_log_messages_total{id="`+con1.ID()+`",message="Startup",service="one.collect",severity="INFO",ver="0"} 1`).
		BodyContains(t, `microbus_uptime_duration_seconds_total{id="`+con1.ID()+`",service="one.collect",ver="0"}`).
		// Cache should have 1 element of 10 bytes
		BodyContains(t, `microbus_cache_weight_total{id="`+con1.ID()+`",service="one.collect",ver="0"} 10`).
		BodyContains(t, `microbus_cache_len_total{id="`+con1.ID()+`",service="one.collect",ver="0"} 1`).
		BodyContains(t, `microbus_cache_misses_total{id="`+con1.ID()+`",service="one.collect",ver="0"} 1`).
		BodyContains(t, `microbus_cache_hits_total{id="`+con1.ID()+`",service="one.collect",ver="0"} 1`).
		BodyContains(t, `microbus_request_count_total{code="404",error="false",host="one.collect",id="`+con1.ID()+`",method="GET",path="/dcache/all",port="888",service="one.collect",ver="0"} 2`).
		// The response size is 10 bytes
		BodyContains(t, `microbus_response_size_bytes_sum{code="200",error="false",handler="one.collect:443/ten",id="`+con1.ID()+`",method="GET",port="443",service="one.collect",ver="0"} 10`).
		BodyContains(t, `microbus_response_size_bytes_count{code="200",error="false",handler="one.collect:443/ten",id="`+con1.ID()+`",method="GET",port="443",service="one.collect",ver="0"} 1`).
		// The request should take between 100ms and 250ms
		BodyContains(t, `microbus_request_count_total{code="200",error="false",host="one.collect",id="`+con1.ID()+`",method="GET",path="/ten",port="443",service="one.collect",ver="0"} 1`).
		BodyContains(t, `microbus_response_duration_seconds_bucket{code="200",error="false",handler="one.collect:443/ten",id="`+con1.ID()+`",method="GET",port="443",service="one.collect",ver="0",le="0.1"} 0`).
		BodyContains(t, `microbus_response_duration_seconds_bucket{code="200",error="false",handler="one.collect:443/ten",id="`+con1.ID()+`",method="GET",port="443",service="one.collect",ver="0",le="0.5"} 1`).
		// Acks should be logged
		BodyContains(t, "microbus_ack_duration_seconds_bucket")
}

func TestMetrics_GZip(t *testing.T) {
	t.Parallel()

	ctx := Context()

	Collect(ctx, Header("Accept-Encoding", "gzip")).Assert(t, func(t *testing.T, res *http.Response, err error) {
		assert.NoError(t, err)
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
		unzipper, err := gzip.NewReader(res.Body)
		assert.NoError(t, err)
		body, err := io.ReadAll(unzipper)
		unzipper.Close()
		assert.NoError(t, err)
		assert.True(t, bytes.Contains(body, []byte("metrics.sys")))
	})
}

func TestMetrics_SecretKey(t *testing.T) {
	// No parallel
	ctx := Context()
	Svc.With(SecretKey("secret1234"))
	Collect(ctx).Error(t, "incorrect secret key")
	Svc.With(SecretKey(""))
	Collect(ctx).NoError(t)
}
