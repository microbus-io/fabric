/*
Copyright 2023 Microbus LLC and various contributors

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

package httpingress

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/services/httpingress/httpingressapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *httpingressapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Create a middleware microservice
	middleware := connector.New("middleware.host")
	middleware.Subscribe("/serve/", func(w http.ResponseWriter, r *http.Request) error {
		options := []pub.Option{
			pub.Method(r.Method),
			pub.URL("https:/" + strings.TrimPrefix(r.URL.RequestURI(), "/serve")),
			pub.Body(r.Body),
			pub.Unicast(),
		}
		for h := range r.Header {
			options = append(options, pub.Header(h, r.Header.Get(h)))
		}
		res, err := middleware.Request(r.Context(), options...)
		if err != nil {
			return errors.Trace(err)
		}
		for h := range res.Header {
			w.Header()[h] = res.Header[h]
		}
		w.WriteHeader(res.StatusCode)
		_, err = io.Copy(w, res.Body)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc.With(
			TimeBudget(time.Second*2),
			Ports("4040,4443"),
			AllowedOrigins("allowed.origin"),
			PortMappings("4040:*->*, 4443:*->443"),
			Middleware("https://middleware.host/serve"),
		),
		middleware,
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

func TestHttpingress_Ports(t *testing.T) {
	t.Parallel()

	con := connector.New("ports")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}
	res, err := client.Get("http://localhost:4040/ports/ok")
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "ok", string(b))
		}
	}
	res, err = client.Get("http://localhost:4443/ports/ok")
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "ok", string(b))
		}
	}
}

func TestHttpingress_RequestMemoryLimit(t *testing.T) {
	// No parallel
	memLimit := Svc.RequestMemoryLimit()
	Svc.With(RequestMemoryLimit(1))
	defer Svc.With(RequestMemoryLimit(memLimit))

	entered := make(chan bool)
	done := make(chan bool)
	con := connector.New("request.memory.limit")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		b, _ := io.ReadAll(r.Body)
		w.Write(b)
		return nil
	})
	con.Subscribe("hold", func(w http.ResponseWriter, r *http.Request) error {
		entered <- true
		<-done
		w.Write([]byte("done"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Small request at 25% of capacity
	assert.Zero(t, Svc.reqMemoryUsed)
	payload := rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 / 4)
	res, err := client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, payload, string(b))
		}
	}

	// Big request at 55% of capacity
	assert.Zero(t, Svc.reqMemoryUsed)
	payload = rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 * 55 / 100)
	res, err = client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}

	// Two small requests that together are over 50% of capacity
	assert.Zero(t, Svc.reqMemoryUsed)
	payload = rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 / 3)
	returned := make(chan bool)
	go func() {
		res, err = client.Post("http://localhost:4040/request.memory.limit/hold", "text/plain", strings.NewReader(payload))
		returned <- true
	}()
	<-entered
	assert.NotZero(t, Svc.reqMemoryUsed)
	res, err = client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}
	done <- true
	<-returned
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "done", string(b))
		}
	}

	assert.Zero(t, Svc.reqMemoryUsed)
}

func TestHttpingress_Compression(t *testing.T) {
	t.Parallel()

	con := connector.New("compression")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(bytes.Repeat([]byte("Hello123"), 1024)) // 8KB
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequest("GET", "http://localhost:4040/compression/ok", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	assert.NoError(t, err)
	res, err := client.Do(req)
	if assert.NoError(t, err) {
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.True(t, len(b) < 8*1024)
		}
		assert.Equal(t, strconv.Itoa(len(b)), res.Header.Get("Content-Length"))
	}
}

func TestHttpingress_PortMapping(t *testing.T) {
	t.Parallel()

	con := connector.New("port.mapping")
	con.Subscribe("ok443", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	con.Subscribe(":555/ok555", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// External port 4040 grants access to all internal ports
	res, err := client.Get("http://localhost:4040/port.mapping/ok443")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4040/port.mapping:555/ok555")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4040/port.mapping:555/ok443")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	}

	// External port 4443 maps all requests to internal port 443
	res, err = client.Get("http://localhost:4443/port.mapping/ok443")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4443/port.mapping:555/ok555")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4443/port.mapping:555/ok443")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
}

func TestHttpingress_ForwardedHeaders(t *testing.T) {
	t.Parallel()

	con := connector.New("forwarded.headers")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		var sb strings.Builder
		for _, h := range []string{"X-Forwarded-Host", "X-Forwarded-Prefix", "X-Forwarded-Proto", "X-Forwarded-For"} {
			if r.Header.Get(h) != "" {
				sb.WriteString(h)
				sb.WriteString(": ")
				sb.WriteString(r.Header.Get(h))
				sb.WriteString("\n")
			}
		}
		w.Write([]byte(sb.String()))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Make a standard request
	req, err := http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
	assert.NoError(t, err)
	res, err := client.Do(req)
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			body := string(b)
			assert.True(t, strings.Contains(body, "X-Forwarded-Host: localhost:4040\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-Prefix: /forwarded.headers\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-Proto: http\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-For: "))
		}
	}

	// Make a request appear to be coming through an upstream proxy server
	req, err = http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
	req.Header.Set("X-Forwarded-Host", "www.example.com")
	req.Header.Set("X-Forwarded-Prefix", "/app")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	assert.NoError(t, err)
	res, err = client.Do(req)
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			body := string(b)
			assert.True(t, strings.Contains(body, "X-Forwarded-Host: www.example.com\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-Prefix: /app/forwarded.headers\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-Proto: https\n"))
			assert.True(t, strings.Contains(body, "X-Forwarded-For: 1.2.3.4"))
		}
	}
}

func TestHttpingress_Root(t *testing.T) {
	t.Parallel()

	client := http.Client{Timeout: time.Second * 2}
	res, err := client.Get("http://localhost:4040/")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	}

	con := connector.New("root")
	con.Subscribe("", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("Root"))
		return nil
	})
	App.Join(con)
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	res, err = client.Get("http://localhost:4040/")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
}

func TestHttpingress_CORS(t *testing.T) {
	t.Parallel()

	callCount := 0
	con := connector.New("cors")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		callCount++
		w.Write([]byte("ok"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Request with no origin header
	count := callCount
	req, err := http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	assert.NoError(t, err)
	res, err := client.Do(req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, count+1, callCount)
	}

	// Request with disallowed origin header
	count = callCount
	req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "disallowed.origin")
	assert.NoError(t, err)
	res, err = client.Do(req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusForbidden, res.StatusCode)
		assert.Equal(t, count, callCount)
	}

	// Request with allowed origin header
	count = callCount
	req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "allowed.origin")
	assert.NoError(t, err)
	res, err = client.Do(req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "allowed.origin", res.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, count+1, callCount)
	}

	// Preflight request with allowed origin header
	count = callCount
	req, err = http.NewRequest("OPTIONS", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "allowed.origin")
	assert.NoError(t, err)
	res, err = client.Do(req)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		assert.Equal(t, count, callCount)
	}
}

func TestHttpingress_ParseForm(t *testing.T) {
	t.Parallel()

	con := connector.New("parse.form")
	con.Subscribe("ok", func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseForm()
		if err != nil {
			return errors.Trace(err)
		}
		w.Write([]byte("ok"))
		return nil
	})
	con.Subscribe("more", func(w http.ResponseWriter, r *http.Request) error {
		r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
		err := r.ParseForm()
		if err != nil {
			return errors.Trace(err)
		}
		w.Write([]byte("ok"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Under 10MB
	var buf bytes.Buffer
	buf.WriteString("x=")
	buf.WriteString(rand.AlphaNum64(9 * 1024 * 1024))
	res, err := client.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "ok", string(b))
		}
	}

	// Go sets a 10MB limit on forms by default
	// https://go.dev/src/net/http/request.go#L1258
	buf.WriteString(rand.AlphaNum64(2 * 1024 * 1024)) // Now 11MB
	res, err = client.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}

	// MaxBytesReader can be used to extend the limit
	res, err = client.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "ok", string(b))
		}
	}

	// Going above the MaxBytesReader limit
	buf.WriteString(rand.AlphaNum64(2 * 1024 * 1024)) // Now 13MB
	res, err = client.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}
}

func TestHttpingress_Middleware(t *testing.T) {
	t.Parallel()

	con := connector.New("final.destination")
	con.Subscribe(":555/ok", func(w http.ResponseWriter, r *http.Request) error {
		// The request should be coming from the middleware
		assert.Equal(t, "middleware.host", frame.Of(r).FromHost())
		// Headers should pass through
		assert.Equal(t, "Bearer 123456", r.Header.Get("Authorization"))
		// Middleware should not add itself to the prefix
		assert.Equal(t, "/final.destination:555", r.Header.Get("X-Forwarded-Prefix"))
		w.Write([]byte("ok"))
		return nil
	})
	App.Join(con)
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", "http://localhost:4040/final.destination:555/ok", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer 123456")
	res, err := client.Do(req)
	if assert.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, "ok", string(b))
		}
	}
}

func TestHttpingress_OnChangedPorts(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedAllowedOrigins(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedPortMappings(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedReadTimeout(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedWriteTimeout(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedReadHeaderTimeout(t *testing.T) {
	t.Skip() // Not tested
}
