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
	"github.com/microbus-io/testarossa"
)

var (
	httpServer *http.Server
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

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
	err = httpServer.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func TestHttpegress_Get(t *testing.T) {
	t.Parallel()

	ctx := Context()
	client := httpegressapi.NewClient(Svc)

	// Echo
	resp, err := client.Get(ctx, "http://127.0.0.1:5050/echo")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		testarossa.Contains(t, string(raw), "GET /echo HTTP/1.1\r\n")
		testarossa.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		testarossa.Contains(t, string(raw), "User-Agent: Go-http-client")
	}

	// Not found
	resp, err = client.Get(ctx, "http://127.0.0.1:5050/x")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	_, err = client.Get(ctx, "not a url")
	testarossa.Error(t, err)

	// Shorter deadline
	shortCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	_, err = client.Get(shortCtx, "http://127.0.0.1:5050/slow")
	cancel()
	if testarossa.Error(t, err) {
		testarossa.Contains(t, err.Error(), "timeout")
	}
}

func TestHttpegress_Post(t *testing.T) {
	t.Parallel()

	ctx := Context()
	client := httpegressapi.NewClient(Svc)

	// Echo
	resp, err := client.Post(ctx, "http://127.0.0.1:5050/echo", "text/plain", strings.NewReader("Lorem Ipsum Dolor Sit Amet"))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		testarossa.Contains(t, string(raw), "POST /echo HTTP/1.1\r\n")
		testarossa.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		testarossa.Contains(t, string(raw), "User-Agent: Go-http-client")
		testarossa.Contains(t, string(raw), "Content-Type: text/plain\r\n")
		testarossa.Contains(t, string(raw), "Lorem Ipsum Dolor Sit Amet")
	}

	// Not found
	resp, err = client.Post(ctx, "http://127.0.0.1:5050/x", "", strings.NewReader("nothing"))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	_, err = client.Post(ctx, "not a url", "", strings.NewReader("nothing"))
	testarossa.Error(t, err)
}

func TestHttpegress_Do(t *testing.T) {
	t.Parallel()

	ctx := Context()
	client := httpegressapi.NewClient(Svc)

	// Echo
	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1:5050/echo", bytes.NewReader([]byte("Lorem Ipsum")))
	req.Header["Multi-Value"] = []string{"Foo", "Bar"}
	testarossa.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(ctx, req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, resp.StatusCode)
		raw, _ := io.ReadAll(resp.Body)
		testarossa.Contains(t, string(raw), "PUT /echo HTTP/1.1\r\n")
		testarossa.Contains(t, string(raw), "Host: 127.0.0.1:5050\r\n")
		testarossa.Contains(t, string(raw), "User-Agent: Go-http-client")
		testarossa.Contains(t, string(raw), "Content-Type: text/plain\r\n")
		testarossa.Contains(t, string(raw), "Multi-Value: Foo\r\n")
		testarossa.Contains(t, string(raw), "Multi-Value: Bar\r\n")
		testarossa.Contains(t, string(raw), "\r\n\r\nLorem Ipsum")
	}

	// Not found
	req, err = http.NewRequest(http.MethodPatch, "http://127.0.0.1:5050/x", bytes.NewReader([]byte("Lorem Ipsum")))
	testarossa.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err = client.Do(ctx, req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Bad URL
	req, err = http.NewRequest(http.MethodDelete, "not a url", nil)
	testarossa.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	_, err = client.Do(ctx, req)
	testarossa.Error(t, err)
}

func TestHttpegress_MakeRequest(t *testing.T) {
	t.Skip() // Tested above
}

func TestHttpegress_Mock(t *testing.T) {
	t.Parallel()

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

	con := connector.New("mock.http.egress")

	app := application.NewTesting()
	app.Add(
		mock,
		con,
	)
	err := app.Startup()
	testarossa.NoError(t, err)
	defer app.Shutdown()

	ctx := Context()
	client := httpegressapi.NewClient(con).ForHost(mock.ID() + "." + mock.Hostname()) // Address the mock by ID

	req, _ := http.NewRequest("DELETE", "https://example.com/ex/5", nil)
	resp, err := client.Do(ctx, req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, resp.StatusCode)
		testarossa.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(resp.Body)
		testarossa.Equal(t, string(raw), `{"deleted":true}`)
	}
}
