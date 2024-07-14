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

package httpingress

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/rand"
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc.Init(func(svc *Service) {
			svc.SetTimeBudget(time.Second * 2)
			svc.SetPorts("4040,4443")
			svc.SetAllowedOrigins("allowed.origin")
			svc.SetPortMappings("4040:*->*, 4443:*->443")
			svc.SetServerLanguages("en,fr,es-ar")
			svc.AddMiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next connector.HTTPHandler) (err error) {
				r.Header.Add("Middleware", "Hello")
				err = next(w, r)
				w.Header().Add("Middleware", "Goodbye")
				return err // No trace
			})
		}),
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

func TestHttpingress_Ports(t *testing.T) {
	t.Parallel()

	con := connector.New("ports")
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	App.Add(con)
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}
	res, err := client.Get("http://localhost:4040/ports/ok")
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "ok", string(b))
		}
	}
	res, err = client.Get("http://localhost:4443/ports/ok")
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "ok", string(b))
		}
	}
}

func TestHttpingress_RequestMemoryLimit(t *testing.T) {
	// No parallel
	memLimit := Svc.RequestMemoryLimit()
	Svc.SetRequestMemoryLimit(1)
	defer Svc.SetRequestMemoryLimit(memLimit)

	entered := make(chan bool)
	done := make(chan bool)
	con := connector.New("request.memory.limit")
	con.Subscribe("POST", "ok", func(w http.ResponseWriter, r *http.Request) error {
		b, _ := io.ReadAll(r.Body)
		w.Write(b)
		return nil
	})
	con.Subscribe("POST", "hold", func(w http.ResponseWriter, r *http.Request) error {
		entered <- true
		<-done
		w.Write([]byte("done"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Small request at 25% of capacity
	testarossa.Zero(t, Svc.reqMemoryUsed)
	payload := rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 / 4)
	res, err := client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, payload, string(b))
		}
	}

	// Big request at 55% of capacity
	testarossa.Zero(t, Svc.reqMemoryUsed)
	payload = rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 * 55 / 100)
	res, err = client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}

	// Two small requests that together are over 50% of capacity
	testarossa.Zero(t, Svc.reqMemoryUsed)
	payload = rand.AlphaNum64(Svc.RequestMemoryLimit() * 1024 * 1024 / 3)
	returned := make(chan bool)
	go func() {
		res, err = client.Post("http://localhost:4040/request.memory.limit/hold", "text/plain", strings.NewReader(payload))
		returned <- true
	}()
	<-entered
	testarossa.NotZero(t, Svc.reqMemoryUsed)
	res, err = client.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}
	done <- true
	<-returned
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "done", string(b))
		}
	}

	testarossa.Zero(t, Svc.reqMemoryUsed)
}

func TestHttpingress_Compression(t *testing.T) {
	t.Parallel()

	con := connector.New("compression")
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(bytes.Repeat([]byte("Hello123"), 1024)) // 8KB
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequest("GET", "http://localhost:4040/compression/ok", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	testarossa.NoError(t, err)
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.True(t, len(b) < 8*1024)
		}
		testarossa.Equal(t, strconv.Itoa(len(b)), res.Header.Get("Content-Length"))
	}
}

func TestHttpingress_PortMapping(t *testing.T) {
	t.Parallel()

	con := connector.New("port.mapping")
	con.Subscribe("GET", "ok443", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	con.Subscribe("GET", ":555/ok555", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// External port 4040 grants access to all internal ports
	res, err := client.Get("http://localhost:4040/port.mapping/ok443")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4040/port.mapping:555/ok555")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4040/port.mapping:555/ok443")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, res.StatusCode)
	}

	// External port 4443 maps all requests to internal port 443
	res, err = client.Get("http://localhost:4443/port.mapping/ok443")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4443/port.mapping:555/ok555")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, res.StatusCode)
	}
	res, err = client.Get("http://localhost:4443/port.mapping:555/ok443")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
}

func TestHttpingress_ForwardedHeaders(t *testing.T) {
	t.Parallel()

	con := connector.New("forwarded.headers")
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		var sb strings.Builder
		for _, h := range []string{"X-Forwarded-Host", "X-Forwarded-Prefix", "X-Forwarded-Proto", "X-Forwarded-For", "X-Forwarded-Path"} {
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
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Make a standard request
	req, err := http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
	testarossa.NoError(t, err)
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			body := string(b)
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Host: localhost:4040\n"))
			testarossa.False(t, strings.Contains(body, "X-Forwarded-Prefix:"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Proto: http\n"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-For: "))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Path: /forwarded.headers/ok"))
		}
	}

	// Make a request appear to be coming through an upstream proxy server
	req, err = http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
	req.Header.Set("X-Forwarded-Host", "www.example.com")
	req.Header.Set("X-Forwarded-Prefix", "/app")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			body := string(b)
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Host: www.example.com\n"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Prefix: /app\n"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Proto: https\n"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-For: 1.2.3.4"))
			testarossa.True(t, strings.Contains(body, "X-Forwarded-Path: /forwarded.headers/ok"))
		}
	}
}

func TestHttpingress_Root(t *testing.T) {
	t.Parallel()

	client := http.Client{Timeout: time.Second * 2}
	res, err := client.Get("http://localhost:4040/")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, res.StatusCode)
	}

	con := connector.New("root")
	con.Subscribe("GET", "", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("Root"))
		return nil
	})
	err = App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	res, err = client.Get("http://localhost:4040/")
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
}

func TestHttpingress_CORS(t *testing.T) {
	t.Parallel()

	callCount := 0
	con := connector.New("cors")
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		callCount++
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Request with no origin header
	count := callCount
	req, err := http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	testarossa.NoError(t, err)
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
		testarossa.Equal(t, count+1, callCount)
	}

	// Request with disallowed origin header
	count = callCount
	req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "disallowed.origin")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusForbidden, res.StatusCode)
		testarossa.Equal(t, count, callCount)
	}

	// Request with allowed origin header
	count = callCount
	req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "allowed.origin")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
		testarossa.Equal(t, "allowed.origin", res.Header.Get("Access-Control-Allow-Origin"))
		testarossa.Equal(t, count+1, callCount)
	}

	// Preflight request with allowed origin header
	count = callCount
	req, err = http.NewRequest("OPTIONS", "http://localhost:4040/cors/ok", nil)
	req.Header.Set("Origin", "allowed.origin")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNoContent, res.StatusCode)
		testarossa.Equal(t, count, callCount)
	}
}

func TestHttpingress_ParseForm(t *testing.T) {
	t.Parallel()

	con := connector.New("parse.form")
	con.Subscribe("POST", "ok", func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseForm()
		if err != nil {
			return errors.Trace(err)
		}
		w.Write([]byte("ok"))
		return nil
	})
	con.Subscribe("POST", "more", func(w http.ResponseWriter, r *http.Request) error {
		r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
		err := r.ParseForm()
		if err != nil {
			return errors.Trace(err)
		}
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	// Under 10MB
	var buf bytes.Buffer
	buf.WriteString("x=")
	buf.WriteString(rand.AlphaNum64(9 * 1024 * 1024))
	res, err := client.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "ok", string(b))
		}
	}

	// Go sets a 10MB limit on forms by default
	// https://go.dev/src/net/http/request.go#L1258
	buf.WriteString(rand.AlphaNum64(2 * 1024 * 1024)) // Now 11MB
	res, err = client.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}

	// MaxBytesReader can be used to extend the limit
	res, err = client.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "ok", string(b))
		}
	}

	// Going above the MaxBytesReader limit
	buf.WriteString(rand.AlphaNum64(2 * 1024 * 1024)) // Now 13MB
	res, err = client.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
	}
}

func TestHttpingress_InternalHeaders(t *testing.T) {
	t.Parallel()

	con := connector.New("internal.headers")
	con.Subscribe("GET", ":555/ok", func(w http.ResponseWriter, r *http.Request) error {
		// No Microbus headers should be accepted from client
		testarossa.Equal(t, "", r.Header.Get(frame.HeaderPrefix+"In-Request"))
		// Microbus headers generated internally should pass through the middleware chain
		testarossa.Equal(t, Hostname, frame.Of(r).FromHost())

		w.Header().Set(frame.HeaderPrefix+"In-Response", "STOP")
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", "http://localhost:4040/internal.headers:555/ok", nil)
	testarossa.NoError(t, err)
	req.Header.Set(frame.HeaderPrefix+"In-Request", "STOP")
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		// No Microbus headers should leak outside
		testarossa.Equal(t, "", res.Header.Get(frame.HeaderPrefix+"In-Response"))
		for h := range res.Header {
			testarossa.False(t, strings.HasPrefix(h, frame.HeaderPrefix))
		}
	}
}

func TestHttpingress_Middleware(t *testing.T) {
	t.Parallel()

	con := connector.New("final.destination")
	con.Subscribe("GET", ":555/ok", func(w http.ResponseWriter, r *http.Request) error {
		// Headers should pass through
		testarossa.Equal(t, "Bearer 123456", r.Header.Get("Authorization"))
		// Middleware added a request header
		testarossa.Equal(t, "Hello", r.Header.Get("Middleware"))
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", "http://localhost:4040/final.destination:555/ok", nil)
	testarossa.NoError(t, err)
	req.Header.Set("Authorization", "Bearer 123456")
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "ok", string(b))
		}
		// Middleware added a response header
		testarossa.Equal(t, "Goodbye", res.Header.Get("Middleware"))
	}
}

func TestHttpingress_BlockedPaths(t *testing.T) {
	t.Parallel()

	con := connector.New("blocked.paths")
	con.Subscribe("GET", "admin.php", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	con.Subscribe("GET", "admin.ppp", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("ok"))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", "http://localhost:4040/blocked.paths/admin.php", nil)
	testarossa.NoError(t, err)
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusNotFound, res.StatusCode)
	}
	req, err = http.NewRequest("GET", "http://localhost:4040/blocked.paths/admin.ppp", nil)
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, http.StatusOK, res.StatusCode)
	}
}

func TestHttpingress_MatchAcceptedLanguages(t *testing.T) {
	t.Parallel()

	con := connector.New("accepted.languages")
	con.Subscribe("GET", "echo", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(r.Header.Get("Accept-Language")))
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{Timeout: time.Second * 2}

	req, err := http.NewRequest("GET", "http://localhost:4040/accepted.languages/echo", nil)
	req.Header.Set("Accept-Language", "en-us,es;q=0.5")
	testarossa.NoError(t, err)
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "en", string(b))
		}
	}
	req.Header.Set("Accept-Language", "es;q=0.5")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "es-AR", string(b))
		}
	}
	req.Header.Set("Accept-Language", "fr-ca;q=0.8,en;q=0.4")
	testarossa.NoError(t, err)
	res, err = client.Do(req)
	if testarossa.NoError(t, err) {
		b, err := io.ReadAll(res.Body)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, "fr", string(b))
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

func TestHttpingress_OnChangedBlockedPaths(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_OnChangedServerLanguages(t *testing.T) {
	t.Skip() // Not tested
}

func TestHttpingress_MultiValueHeaders(t *testing.T) {
	t.Parallel()

	con := connector.New("multi.value.headers")
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		if testarossa.Equal(t, 3, len(r.Header["Multi-Value"])) {
			testarossa.Equal(t, "Send 1", r.Header["Multi-Value"][0])
			testarossa.Equal(t, "Send 2", r.Header["Multi-Value"][1])
			testarossa.Equal(t, "Send 3", r.Header["Multi-Value"][2])
		}
		w.Header()["Multi-Value"] = []string{
			"Return 1",
			"Return 2",
		}
		return nil
	})
	err := App.AddAndStartup(con)
	testarossa.NoError(t, err)
	defer con.Shutdown()

	client := http.Client{} // Timeout: time.Second * 2}
	req, err := http.NewRequest("GET", "http://localhost:4040/multi.value.headers/ok", nil)
	testarossa.NoError(t, err)
	req.Header["Multi-Value"] = []string{
		"Send 1",
		"Send 2",
		"Send 3",
	}
	res, err := client.Do(req)
	if testarossa.NoError(t, err) {
		if testarossa.Equal(t, 2, len(res.Header["Multi-Value"])) {
			testarossa.Equal(t, "Return 1", res.Header["Multi-Value"][0])
			testarossa.Equal(t, "Return 2", res.Header["Multi-Value"][1])
		}
	}
}
