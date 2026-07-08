/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
)

var (
	_ context.Context
	_ *testing.T
	_ *application.Application
	_ *connector.Connector
	_ pub.Option
	_ testarossa.TestingT
	_ metricsapi.Client
)

// MARKER: Collect

func TestMetrics_Collect(t *testing.T) {
	// No parallel - Setting envars
	env.Push("MICROBUS_PROMETHEUS_EXPORTER", "1")
	defer env.Pop("MICROBUS_PROMETHEUS_EXPORTER")

	ctx := t.Context()
	assert := testarossa.For(t)

	// Initialize the microservice under test
	svc := NewService()
	// svc.SetSecretKey(secretKey)

	con1 := connector.New("one.collect")
	con1.SetOnStartup(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	con1.Subscribe("Ten",
		func(w http.ResponseWriter, r *http.Request) error {
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte("1234567890"))
			return nil
		},
		sub.At("GET", "/ten"),
		sub.Web(),
	)
	con2 := connector.New("two.collect")

	// Initialize the testers
	tester := connector.New("tester.client")
	client := metricsapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
		con1,
		con2,
	)
	app.RunInTest(t)

	// Make a request to the service
	_, err := con1.GET(ctx, "https://one.collect/ten")
	assert.NoError(err)

	// Interact with the cache
	con1.DistribCache().Set(ctx, "A", []byte("1234567890"))
	var val []byte
	con1.DistribCache().Get(ctx, "A", &val)
	con1.DistribCache().Get(ctx, "B", &val)

	findLine := func(body []byte, name string, value string, attrs ...string) bool {
		var sb strings.Builder
		if name != "" {
			sb.WriteString(regexp.QuoteMeta(name))
			sb.WriteString(regexp.QuoteMeta("{"))
		}
		for i := 0; i < len(attrs); i += 2 {
			sb.WriteString(".*")
			sb.WriteString(regexp.QuoteMeta(attrs[i]))
			sb.WriteString(regexp.QuoteMeta(`="`))
			sb.WriteString(regexp.QuoteMeta(attrs[i+1]))
			sb.WriteString(regexp.QuoteMeta(`"`))
		}
		sb.WriteString(".*")
		if value != "" {
			sb.WriteString(regexp.QuoteMeta("} "))
			sb.WriteString(regexp.QuoteMeta(value))
		}
		re := regexp.MustCompile(sb.String())
		return re.Match(body)
	}

	var body []byte
	res, err := client.Collect(ctx, "")
	if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
		body, _ = io.ReadAll(res.Body)
	}

	t.Run("detect_all_services", func(t *testing.T) {
		assert := testarossa.For(t)

		assert.Contains(body, svc.Hostname())
		assert.Contains(body, con1.Hostname())
		assert.Contains(body, con2.Hostname())
	})

	t.Run("con1_startup_callback_duration", func(t *testing.T) {
		assert := testarossa.For(t)

		// The startup callback should take between 100ms and 500ms
		assert.True(findLine(body, "microbus_callback_duration_seconds_bucket", "0",
			"error", "OK",
			"id", con1.ID(),
			"name", "OnStartup",
			"service", con1.Hostname(),
			"type", "lifecycle",
			"le", "0.1",
		))
		assert.True(findLine(body, "microbus_callback_duration_seconds_bucket", "1",
			"error", "OK",
			"id", con1.ID(),
			"name", "OnStartup",
			"service", con1.Hostname(),
			"type", "lifecycle",
			"le", "0.5",
		))
		assert.True(findLine(body, "microbus_log_messages_total", "1",
			"id", con1.ID(),
			"message", "Startup",
			"service", con1.Hostname(),
			"severity", "INFO",
		))
		assert.True(findLine(body, "microbus_uptime_duration_seconds", "",
			"id", con1.ID(),
			"service", con1.Hostname(),
		))
	})

	t.Run("cache_elements", func(t *testing.T) {
		assert := testarossa.For(t)

		// Cache should have 1 element of 10 bytes
		assert.True(findLine(body, "microbus_cache_memory_bytes", "10",
			"id", con1.ID(),
			"service", con1.Hostname(),
		))
		assert.True(findLine(body, "microbus_cache_elements", "1",
			"id", con1.ID(),
			"service", con1.Hostname(),
		))
		assert.True(findLine(body, "microbus_cache_operations_total", "1",
			"hit", "miss",
			"id", con1.ID(),
			"op", "load",
			"service", con1.Hostname(),
		))
		assert.True(findLine(body, "microbus_cache_operations_total", "1",
			"hit", "local",
			"id", con1.ID(),
			"op", "load",
			"service", con1.Hostname(),
		))
		assert.True(findLine(body, "microbus_server_request_duration_seconds_count", "2",
			"code", "404",
			"error", "OK",
			"id", con1.ID(),
			"method", "GET",
			"name", "DcacheAll",
			"port", "888",
			"service", con1.Hostname(),
			"type", "web",
		))
	})

	t.Run("response", func(t *testing.T) {
		assert := testarossa.For(t)

		// The response size is 10 bytes
		assert.True(findLine(body, "microbus_server_response_body_bytes_sum", "10",
			"code", "200",
			"error", "OK",
			"id", con1.ID(),
			"method", "GET",
			"name", "Ten",
			"port", "443",
			"service", con1.Hostname(),
			"type", "web",
		))
		assert.True(findLine(body, "microbus_server_response_body_bytes_count", "1",
			"code", "200",
			"error", "OK",
			"id", con1.ID(),
			"method", "GET",
			"name", "Ten",
			"port", "443",
			"service", con1.Hostname(),
			"type", "web",
		))
	})

	t.Run("request", func(t *testing.T) {
		assert := testarossa.For(t)

		// The request should take between 100ms and 500ms
		assert.True(findLine(body, "microbus_server_request_duration_seconds_bucket", "0",
			"code", "200",
			"error", "OK",
			"id", con1.ID(),
			"method", "GET",
			"name", "Ten",
			"port", "443",
			"service", con1.Hostname(),
			"type", "web",
			"le", "0.1",
		))
		assert.True(findLine(body, "microbus_server_request_duration_seconds_bucket", "1",
			"code", "200",
			"error", "OK",
			"id", con1.ID(),
			"method", "GET",
			"name", "Ten",
			"port", "443",
			"service", con1.Hostname(),
			"type", "web",
			"le", "0.5",
		))
	})

	t.Run("acks", func(t *testing.T) {
		assert := testarossa.For(t)

		// Acks should be logged
		assert.Contains(body, "microbus_client_ack_roundtrip_latency_seconds_bucket")
	})
}

func TestMetrics_Gzip(t *testing.T) {
	// No parallel - Setting envars
	env.Push("MICROBUS_PROMETHEUS_EXPORTER", "1")
	defer env.Pop("MICROBUS_PROMETHEUS_EXPORTER")

	ctx := t.Context()
	assert := testarossa.For(t)

	// Initialize the microservice under test
	svc := NewService()
	// svc.SetSecretKey(secretKey)

	// Initialize the testers
	tester := connector.New("tester.client")
	client := metricsapi.NewClient(tester).WithOptions(
		// Add options as required
		pub.Header("Accept-Encoding", "gzip"),
	)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	res, err := client.Collect(ctx, "")
	if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
		assert.Equal("gzip", res.Header.Get("Content-Encoding"))
		unzipper, err := gzip.NewReader(res.Body)
		assert.NoError(err)
		body, err := io.ReadAll(unzipper)
		unzipper.Close()
		assert.NoError(err)
		assert.Contains(body, "microbus_log_messages")
	}
}

func TestMetrics_SecretKey(t *testing.T) {
	// No parallel - Setting envars
	env.Push("MICROBUS_PROMETHEUS_EXPORTER", "1")
	defer env.Pop("MICROBUS_PROMETHEUS_EXPORTER")

	ctx := t.Context()

	// Initialize the microservice under test
	svc := NewService()
	// svc.SetSecretKey(secretKey)

	// Initialize the testers
	tester := connector.New("tester.client")
	client := metricsapi.NewClient(tester).WithOptions(
		// Add options as required
		pub.Header("Accept-Encoding", "gzip"),
	)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	t.Run("no_key_provided", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey(utils.RandomIdentifier(16))

		_, err := client.Collect(ctx, "")
		assert.Contains(err, "incorrect secret key")
	})

	t.Run("incorrect_key_provided", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey(utils.RandomIdentifier(16))

		_, err := client.Collect(ctx, "?secretkey="+utils.RandomIdentifier(16))
		assert.Contains(err, "incorrect secret key")
	})

	t.Run("correct_key_provided", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey(utils.RandomIdentifier(16))

		_, err := client.Collect(ctx, "?secretkey="+svc.SecretKey())
		assert.NoError(err)

		_, err = client.Collect(ctx, "?secretKey="+svc.SecretKey())
		assert.NoError(err)
	})

	t.Run("correct_key_in_bearer_header", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey(utils.RandomIdentifier(16))

		bearerClient := metricsapi.NewClient(tester).WithOptions(
			pub.Header("Authorization", "Bearer "+svc.SecretKey()),
		)
		_, err := bearerClient.Collect(ctx, "")
		assert.NoError(err)
	})

	t.Run("incorrect_key_in_bearer_header", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey(utils.RandomIdentifier(16))

		bearerClient := metricsapi.NewClient(tester).WithOptions(
			pub.Header("Authorization", "Bearer "+utils.RandomIdentifier(16)),
		)
		_, err := bearerClient.Collect(ctx, "")
		assert.Contains(err, "incorrect secret key")

		// The header takes precedence over the query argument
		_, err = bearerClient.Collect(ctx, "?secretkey="+svc.SecretKey())
		assert.Contains(err, "incorrect secret key")
	})

	t.Run("no_key_required", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey("")

		_, err := client.Collect(ctx, "")
		assert.NoError(err)
	})

	t.Run("no_key_required_but_still_provided", func(t *testing.T) {
		assert := testarossa.For(t)

		svc.SetSecretKey("")

		_, err := client.Collect(ctx, "?secretkey="+utils.RandomIdentifier(16))
		assert.NoError(err)
	})
}
