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

package httpegress

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/httpegress/httpegressapi"
)

var (
	_ context.Context
	_ *testing.T
	_ jwt.MapClaims
	_ application.Application
	_ connector.Connector
	_ frame.Frame
	_ pub.Option
	_ testarossa.Asserter
	_ httpegressapi.Client
)

func TestHttpegress_MakeRequest(t *testing.T) { // MARKER: MakeRequest
	// No t.Parallel: starting a web server
	ctx := t.Context()

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	client := httpegressapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	// Start a standard web server
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		r.Write(w)
	})
	http.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
	})
	http.HandleFunc("/br-compressed", func(w http.ResponseWriter, r *http.Request) {
		// Egress proxy should know to decompress
		w.Header().Set("Content-Encoding", "br")
		compressor := brotli.NewWriterLevel(w, brotli.BestSpeed)
		compressor.Write([]byte("The quick brown fox jumps over the lazy dog"))
		compressor.Close()
	})
	http.HandleFunc("/gzip-compressed", func(w http.ResponseWriter, r *http.Request) {
		// Egress proxy should know to decompress
		w.Header().Set("Content-Encoding", "gzip")
		compressor := gzip.NewWriter(w)
		compressor.Write([]byte("The quick brown fox jumps over the lazy dog"))
		compressor.Close()
	})
	http.HandleFunc("/deflate-compressed", func(w http.ResponseWriter, r *http.Request) {
		// Egress proxy should know to decompress
		w.Header().Set("Content-Encoding", "deflate")
		compressor, _ := flate.NewWriter(w, flate.BestSpeed)
		compressor.Write([]byte("The quick brown fox jumps over the lazy dog"))
		compressor.Close()
	})
	httpServer := &http.Server{
		Addr: "127.0.0.1:5050",
	}
	go func() {
		httpServer.ListenAndServe()
	}()
	t.Cleanup(func() {
		httpServer.Shutdown(context.Background())
	})
	time.Sleep(200 * time.Millisecond) // Give enough time for web server to start

	t.Run("get", func(t *testing.T) {
		assert := testarossa.For(t)

		// Echo
		resp, err := client.Get(ctx, "http://127.0.0.1:5050/echo")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, resp.StatusCode)
			raw, _ := io.ReadAll(resp.Body)
			assert.Contains(string(raw), "GET /echo HTTP/1.1\r\n")
			assert.Contains(string(raw), "Host: 127.0.0.1:5050\r\n")
			assert.Contains(string(raw), "User-Agent: Go-http-client")
		}

		// Not found
		resp, err = client.Get(ctx, "http://127.0.0.1:5050/x")
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, resp.StatusCode)
		}

		// Bad URL
		_, err = client.Get(ctx, "not a url")
		assert.Error(err)

		// Shorter deadline: the egress call is now bounded by the caller's context, so the outbound request
		// is cancelled at the deadline instead of running to completion (or leaking) behind the bus timeout.
		shortCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_, err = client.Get(shortCtx, "http://127.0.0.1:5050/slow")
		cancel()
		if assert.Error(err) {
			assert.Contains(err.Error(), "deadline exceeded")
		}
	})

	t.Run("post", func(t *testing.T) {
		assert := testarossa.For(t)

		// Echo
		resp, err := client.Post(ctx, "http://127.0.0.1:5050/echo", "text/plain", strings.NewReader("Lorem Ipsum Dolor Sit Amet"))
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, resp.StatusCode)
			raw, _ := io.ReadAll(resp.Body)
			assert.Contains(string(raw), "POST /echo HTTP/1.1\r\n")
			assert.Contains(string(raw), "Host: 127.0.0.1:5050\r\n")
			assert.Contains(string(raw), "User-Agent: Go-http-client")
			assert.Contains(string(raw), "Content-Type: text/plain\r\n")
			assert.Contains(string(raw), "Lorem Ipsum Dolor Sit Amet")
		}

		// Not found
		resp, err = client.Post(ctx, "http://127.0.0.1:5050/x", "", strings.NewReader("nothing"))
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, resp.StatusCode)
		}

		// Bad URL
		_, err = client.Post(ctx, "not a url", "", strings.NewReader("nothing"))
		assert.Error(err)
	})

	t.Run("do", func(t *testing.T) {
		assert := testarossa.For(t)

		// Echo
		req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:5050/echo", bytes.NewReader([]byte("Lorem Ipsum")))
		req.Header["Multi-Value"] = []string{"Foo", "Bar"}
		assert.NoError(err)
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(ctx, req)
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, resp.StatusCode)
			raw, _ := io.ReadAll(resp.Body)
			assert.Contains(string(raw), "PUT /echo HTTP/1.1\r\n")
			assert.Contains(string(raw), "Host: 127.0.0.1:5050\r\n")
			assert.Contains(string(raw), "User-Agent: Go-http-client")
			assert.Contains(string(raw), "Content-Type: text/plain\r\n")
			assert.Contains(string(raw), "Multi-Value: Foo\r\n")
			assert.Contains(string(raw), "Multi-Value: Bar\r\n")
			assert.Contains(string(raw), "\r\n\r\nLorem Ipsum")
		}

		// Not found
		req, err = http.NewRequest(http.MethodPatch, "http://127.0.0.1:5050/x", bytes.NewReader([]byte("Lorem Ipsum")))
		assert.NoError(err)
		req.Header.Set("Content-Type", "text/plain")

		resp, err = client.Do(ctx, req)
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, resp.StatusCode)
		}

		// Bad URL
		req, err = http.NewRequest(http.MethodDelete, "not a url", nil)
		assert.NoError(err)
		req.Header.Set("Content-Type", "text/plain")

		_, err = client.Do(ctx, req)
		assert.Error(err)
	})

	t.Run("compression", func(t *testing.T) {
		assert := testarossa.For(t)

		for _, algo := range []string{"br", "gzip", "deflate"} {
			req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:5050/"+algo+"-compressed", nil)
			assert.NoError(err)
			req.Header.Set("Accept-Encoding", "gzip")

			resp, err := client.Do(ctx, req)
			if assert.NoError(err) {
				assert.Equal(http.StatusOK, resp.StatusCode)
				raw, _ := io.ReadAll(resp.Body)
				// Response is never encoded
				assert.Equal("", resp.Header.Get("Content-Encoding"))
				assert.Equal("The quick brown fox jumps over the lazy dog", string(raw))
				assert.Equal(strconv.Itoa(len("The quick brown fox jumps over the lazy dog")), resp.Header.Get("Content-Length"))
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		assert := testarossa.For(t)
		_, err := client.Get(ctx, "!@#$^")
		assert.Error(err)
	})
}

func TestHttpegress_Mocked(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Initialize the mocked microservice
	mock := NewMock().
		MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
			req, _ := http.ReadRequest(bufio.NewReader(r.Body))
			if req.Method == "DELETE" && req.URL.String() == "https://example.com/ex/5" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"deleted":true}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return nil
		})

	// Initialize the testers
	tester := connector.New("tester.client")
	client := httpegressapi.NewClient(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		mock,
		tester,
	)
	app.RunInTest(t)

	req, err := http.NewRequest("DELETE", "https://example.com/ex/5", nil)
	assert.NoError(err)
	resp, err := client.Do(ctx, req)
	if assert.NoError(err) && assert.Equal(http.StatusOK, resp.StatusCode) {
		assert.Equal("application/json", resp.Header.Get("Content-Type"))
		raw, err := io.ReadAll(resp.Body)
		if assert.NoError(err) {
			assert.Equal(string(raw), `{"deleted":true}`)
		}
	}
}

// MARKER: MakeRequest

func TestHttpegress_IsBlockedIP(t *testing.T) { // MARKER: MakeRequest
	assert := testarossa.For(t)

	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"169.254.169.254",                       // cloud metadata (link-local)
		"10.1.2.3", "172.16.0.1", "192.168.1.1", // RFC1918 private
		"fd00:ec2::254", // IPv6 unique-local (AWS metadata)
		"0.0.0.0", "::", // unspecified
		"::ffff:127.0.0.1", // IPv4-mapped loopback
	}
	for _, s := range blocked {
		assert.True(isBlockedIP(net.ParseIP(s)), "expected %s to be blocked", s)
	}

	allowed := []string{
		"8.8.8.8", "1.1.1.1", "93.184.216.34", // public IPv4
		"2001:4860:4860::8888", // public IPv6
	}
	for _, s := range allowed {
		assert.False(isBlockedIP(net.ParseIP(s)), "expected %s to be allowed", s)
	}
}

func TestHTTPEgress_MakeRequest(t *testing.T) { // MARKER: MakeRequest
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := httpegressapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			res, err := client.MakeRequest(ctx, "", nil)
			if assert.NoError(err) {
				assert.Expect(res.StatusCode, http.StatusOK)
			}
		})
	*/
}
