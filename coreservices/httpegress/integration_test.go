/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpegress

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/httpegress/httpegressapi"
	"github.com/stretchr/testify/assert"
)

var (
	httpServer *http.Server
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	App.Include(
		Svc,
	)

	err := App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		r.Write(w)
	})
	http.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
	})
	httpServer = &http.Server{
		Addr: "127.0.0.1:5050",
	}
	go func() {
		_ = httpServer.ListenAndServe()
	}()
	time.Sleep(200 * time.Millisecond) // Give enough time for web server to start

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := httpServer.Shutdown(context.Background())
	if err != nil {
		return err
	}

	err = App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestHttpegress_Get(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	client := httpegressapi.NewClient(Svc)

	// Echo
	resp, err := client.Get(ctx, "http://127.0.0.1:5050/echo")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(raw), "GET /echo HTTP/1.1\r\n")
		assert.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		assert.Contains(t, string(raw), "User-Agent: Go-http-client")
	}

	// Not found
	resp, err = client.Get(ctx, "http://127.0.0.1:5050/x")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	_, err = client.Get(ctx, "not a url")
	assert.Error(t, err)

	// Shorter deadline
	shortCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	_, err = client.Get(shortCtx, "http://127.0.0.1:5050/slow")
	cancel()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "timeout")
	}
}

func TestHttpegress_Post(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	client := httpegressapi.NewClient(Svc)

	// Echo
	resp, err := client.Post(ctx, "http://127.0.0.1:5050/echo", "text/plain", strings.NewReader("Lorem Ipsum Dolor Sit Amet"))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(raw), "POST /echo HTTP/1.1\r\n")
		assert.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		assert.Contains(t, string(raw), "User-Agent: Go-http-client")
		assert.Contains(t, string(raw), "Content-Type: text/plain\r\n")
		assert.Contains(t, string(raw), "Lorem Ipsum Dolor Sit Amet")
	}

	// Not found
	resp, err = client.Post(ctx, "http://127.0.0.1:5050/x", "", strings.NewReader("nothing"))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	_, err = client.Post(ctx, "not a url", "", strings.NewReader("nothing"))
	assert.Error(t, err)
}

func TestHttpegress_Do(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	client := httpegressapi.NewClient(Svc)

	// Echo
	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:5050/echo", bytes.NewReader([]byte("Lorem Ipsum")))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(raw), "PUT /echo HTTP/1.1\r\n")
		assert.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		assert.Contains(t, string(raw), "User-Agent: Go-http-client")
		assert.Contains(t, string(raw), "Content-Type: text/plain\r\n")
		assert.Contains(t, string(raw), "\r\n\r\nLorem Ipsum")
	}

	// Not found
	req, err = http.NewRequest(http.MethodPatch, "http://127.0.0.1:5050/x", bytes.NewReader([]byte("Lorem Ipsum")))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err = client.Do(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	req, err = http.NewRequest(http.MethodDelete, "not a url", nil)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	_, err = client.Do(ctx, req)
	assert.Error(t, err)
}

func TestHttpegress_MakeRequest(t *testing.T) {
	t.Skip() // Tested above
}

func TestHttpegress_Mock(t *testing.T) {
	t.Parallel()

	mock := NewMock()
	mock.MockMakeRequest = func(w http.ResponseWriter, r *http.Request) (err error) {
		req, _ := http.ReadRequest(bufio.NewReader(r.Body))
		if req.Method == "DELETE" && req.URL.String() == "https://example.com/ex/5" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"deleted":true}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return nil
	}

	con := connector.New("mock.http.egress")

	app := application.NewTesting()
	app.Include(
		mock,
		con,
	)
	err := app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	ctx := Context(t)
	client := httpegressapi.NewClient(con).ForHost(mock.ID() + "." + mock.HostName()) // Address the mock by ID

	req, _ := http.NewRequest("DELETE", "https://example.com/ex/5", nil)
	resp, err := client.Do(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, string(raw), `{"deleted":true}`)
	}
}
