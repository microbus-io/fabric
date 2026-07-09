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

package httpingress

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/accesstoken"
	"github.com/microbus-io/fabric/coreservices/bearertoken"
	"github.com/microbus-io/fabric/coreservices/bearertoken/bearertokenapi"
	"github.com/microbus-io/fabric/coreservices/httpingress/middleware"
	"github.com/microbus-io/fabric/coreservices/metrics/metricsapi"
)

func TestHttpingress_Incoming(t *testing.T) {
	// No t.Parallel: starting a web server
	ctx := t.Context()

	entered := make(chan bool)
	done := make(chan bool)
	callCount := 0
	var request *http.Request
	countActors := 0

	// Initialize the microservice under test
	svc := NewService()
	svc.SetTimeBudget(time.Second * 2)
	svc.SetPorts("4040,40443")
	svc.SetAllowedCredentialedOrigins("allowed.origin")
	// Internal :443 is implicit; :5555 is needed by several downstream test services below.
	svc.SetAllowedInternalPorts("5500, 5555")
	svc.Middleware().Append("HelloGoodbye", middleware.OnRoutePrefix("/greeting:5555/", middleware.Group(
		func(next connector.HTTPHandler) connector.HTTPHandler {
			return func(w http.ResponseWriter, r *http.Request) (err error) {
				r.Header.Add("Middleware", "Hello")
				return next(w, r) // No trace
			}
		},
		func(next connector.HTTPHandler) connector.HTTPHandler {
			return func(w http.ResponseWriter, r *http.Request) (err error) {
				err = next(w, r)
				w.Header().Add("Middleware", "Goodbye")
				return err // No trace
			}
		},
	)))
	svc.Middleware().Append("401Redirect", middleware.ErrorPageRedirect(http.StatusUnauthorized, "/login-page"))

	// Initialize the testers
	tester := connector.New("tester.client")
	client := metricsapi.NewClient(tester)
	_ = client
	httpClient := http.Client{Timeout: time.Second * 4}

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		accesstoken.NewService(),
		bearertoken.NewService(),
		svc,
		tester,
		connector.New("ports").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("request.memory.limit").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					b, _ := io.ReadAll(r.Body)
					w.Write(b)
					return nil
				},
				sub.At("POST", "ok"),
				sub.Web(),
			)
			c.Subscribe("Hold",
				func(w http.ResponseWriter, r *http.Request) error {
					entered <- true
					<-done
					w.Write([]byte("done"))
					return nil
				},
				sub.At("POST", "hold"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("compression").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Header().Set("Content-Type", "text/plain")
					w.Write(bytes.Repeat([]byte("Hello123"), 1024)) // 8KB
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("port.mapping").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok443",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "ok443"),
				sub.Web(),
			)
			c.Subscribe("Ok555",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", ":5555/ok555"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("forwarded.headers").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
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
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("root").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Root",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("Root"))
					return nil
				},
				sub.At("GET", "/"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("cors").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					callCount++
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("parse.form").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					err := r.ParseForm()
					if err != nil {
						return errors.Trace(err)
					}
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("POST", "ok"),
				sub.Web(),
			)
			c.Subscribe("More",
				func(w http.ResponseWriter, r *http.Request) error {
					r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
					err := r.ParseForm()
					if err != nil {
						return errors.Trace(err)
					}
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("POST", "more"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("internal.headers").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					request = r
					w.Header().Set(frame.HeaderPrefix+"In-Response", "STOP")
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", ":5555/ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("greeting").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok555",
				func(w http.ResponseWriter, r *http.Request) error {
					request = r
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", ":5555/ok"),
				sub.Web(),
			)
			c.Subscribe("Ok5500",
				func(w http.ResponseWriter, r *http.Request) error {
					request = r
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", ":5500/ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("blocked.paths").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("AdminPhp",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "admin.php"),
				sub.Web(),
			)
			c.Subscribe("AdminPpp",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "admin.ppp"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("no.cache").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("ok"))
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("auth.token.entry").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					if ok, _ := frame.Of(r).IfActor(`iss`); ok {
						countActors++
					}
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("authorization").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Protected",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("Access Granted"))
					return nil
				},
				sub.At("GET", "protected"),
				sub.Web(),
				sub.RequiredClaims("role=='major'"),
			)
			c.Subscribe("LoginPage",
				func(w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("Login"))
					return nil
				},
				sub.At("GET", "//login-page"),
				sub.Web(),
			)
			return nil
		}),
		connector.New("multi.value.headers").Init(func(c *connector.Connector) (err error) {
			c.Subscribe("Ok",
				func(w http.ResponseWriter, r *http.Request) error {
					request = r
					w.Header()["Multi-Value"] = []string{
						"Return 1",
						"Return 2",
					}
					return nil
				},
				sub.At("GET", "ok"),
				sub.Web(),
			)
			return nil
		}),
	)
	app.RunInTest(t)

	t.Run("ports", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := httpClient.Get("http://localhost:4040/ports/ok")
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
			}
		}
		res, err = httpClient.Get("http://localhost:40443/ports/ok")
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
			}
		}
	})

	t.Run("uri_too_long", func(t *testing.T) {
		assert := testarossa.For(t)

		// The hostname and path map to a NATS subject, capped at 1024 by the connector
		res, err := httpClient.Get("http://localhost:4040/ports/" + strings.Repeat("x", 1100))
		if assert.NoError(err) {
			assert.Equal(http.StatusRequestURITooLong, res.StatusCode)
		}
	})

	t.Run("trust_root_and_control_ports_blocked", func(t *testing.T) {
		assert := testarossa.For(t)

		// Ports :666 (trust-root) and :888 (control plane) are unconditionally
		// blocked at the ingress regardless of deployment mode or PortMappings.
		// Even if a service registered an endpoint on these ports, the ingress
		// must return 404 before routing.
		for _, blocked := range []int{666, 888} {
			url := fmt.Sprintf("http://localhost:4040/ports:%d/ok", blocked)
			res, err := httpClient.Get(url)
			if assert.NoError(err) {
				assert.Equal(http.StatusNotFound, res.StatusCode, "port :%d should be blocked at ingress", blocked)
			}
		}
	})

	t.Run("request_memory_limit", func(t *testing.T) {
		assert := testarossa.For(t)

		origLimit := svc.RequestMemoryLimit()
		svc.SetRequestMemoryLimit(1) // 1MB
		defer svc.SetRequestMemoryLimit(origLimit)

		// Small request at 25% of capacity
		assert.Zero(svc.reqMemoryUsed)
		payload := utils.RandomIdentifier(svc.RequestMemoryLimit() * 1024 * 1024 / 4)
		res, err := httpClient.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal(payload, string(b))
			}
		}

		// Big request at 55% of capacity
		assert.Zero(svc.reqMemoryUsed)
		payload = utils.RandomIdentifier(svc.RequestMemoryLimit() * 1024 * 1024 * 55 / 100)
		res, err = httpClient.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
		if assert.NoError(err) {
			assert.Equal(http.StatusRequestEntityTooLarge, res.StatusCode)
		}

		// Two small requests that together are over 50% of capacity
		assert.Zero(svc.reqMemoryUsed)
		payload = utils.RandomIdentifier(svc.RequestMemoryLimit() * 1024 * 1024 / 3)
		returned := make(chan bool)
		go func() {
			res, err = httpClient.Post("http://localhost:4040/request.memory.limit/hold", "text/plain", strings.NewReader(payload))
			returned <- true
		}()
		<-entered
		assert.NotZero(svc.reqMemoryUsed)
		res, err = httpClient.Post("http://localhost:4040/request.memory.limit/ok", "text/plain", strings.NewReader(payload))
		if assert.NoError(err) {
			assert.Equal(http.StatusRequestEntityTooLarge, res.StatusCode)
		}
		done <- true
		<-returned
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("done", string(b))
			}
		}

		assert.Zero(svc.reqMemoryUsed)
	})

	t.Run("compression", func(t *testing.T) {
		assert := testarossa.For(t)

		req, err := http.NewRequest("GET", "http://localhost:4040/compression/ok", nil)
		assert.NoError(err)
		req.Header.Set("Accept-Encoding", "gzip")
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal("gzip", res.Header.Get("Content-Encoding"))
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.True(len(b) < 8*1024)
			}
			assert.Equal(strconv.Itoa(len(b)), res.Header.Get("Content-Length"))
		}
	})

	t.Run("internal_ports_firewall", func(t *testing.T) {
		assert := testarossa.For(t)

		// :443 is the implicit default; the path with no port goes there.
		res, err := httpClient.Get("http://localhost:4040/port.mapping/ok443")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
		}
		// :5555 is in AllowedInternalPorts and routes without rewriting.
		res, err = httpClient.Get("http://localhost:4040/port.mapping:5555/ok555")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
		}
		// :5555 is allowed but the requested handler does not exist there.
		res, err = httpClient.Get("http://localhost:4040/port.mapping:5555/ok443")
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, res.StatusCode)
		}

		// Same behavior via the second external listener; no port rewriting happens here either.
		res, err = httpClient.Get("http://localhost:40443/port.mapping/ok443")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
		}
		res, err = httpClient.Get("http://localhost:40443/port.mapping:5555/ok555")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
		}
		res, err = httpClient.Get("http://localhost:40443/port.mapping:5555/ok443")
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, res.StatusCode)
		}

		// A port not in AllowedInternalPorts is rejected by the firewall with a 404.
		res, err = httpClient.Get("http://localhost:4040/port.mapping:556/ok555")
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, res.StatusCode)
		}
	})

	t.Run("forwarded_headers", func(t *testing.T) {
		assert := testarossa.For(t)

		// Make a standard request
		req, err := http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
		assert.NoError(err)
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				body := string(b)
				assert.True(strings.Contains(body, "X-Forwarded-Host: localhost:4040\n"))
				assert.False(strings.Contains(body, "X-Forwarded-Prefix:"))
				assert.True(strings.Contains(body, "X-Forwarded-Proto: http\n"))
				assert.True(strings.Contains(body, "X-Forwarded-For: "))
				assert.True(strings.Contains(body, "X-Forwarded-Path: /forwarded.headers/ok"))
			}
		}

		// With no trusted proxies (the default), client-supplied X-Forwarded headers are ignored and rewritten
		req, err = http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
		assert.NoError(err)
		req.Header.Set("X-Forwarded-Host", "www.spoofed.example")
		req.Header.Set("X-Forwarded-Prefix", "/app")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		req.Header.Set("X-Forwarded-Proto", "https")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				body := string(b)
				assert.True(strings.Contains(body, "X-Forwarded-Host: localhost:4040\n"))
				assert.False(strings.Contains(body, "X-Forwarded-Prefix:"))
				assert.True(strings.Contains(body, "X-Forwarded-Proto: http\n"))
				assert.False(strings.Contains(body, "1.2.3.4"))
				assert.True(strings.Contains(body, "X-Forwarded-Path: /forwarded.headers/ok"))
			}
		}

		// Behind one trusted proxy, its X-Forwarded headers are trusted,
		// but the untrusted portion of the For chain is truncated
		err = svc.SetTrustedProxyHops(1)
		assert.NoError(err)
		defer svc.SetTrustedProxyHops(0)
		req, err = http.NewRequest("GET", "http://localhost:4040/forwarded.headers/ok", nil)
		assert.NoError(err)
		req.Header.Set("X-Forwarded-Host", "www.example.com")
		req.Header.Set("X-Forwarded-Prefix", "/app")
		req.Header.Set("X-Forwarded-For", "6.6.6.6, 1.2.3.4") // 6.6.6.6 fabricated by the client, 1.2.3.4 appended by the proxy
		req.Header.Set("X-Forwarded-Proto", "https")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				body := string(b)
				assert.True(strings.Contains(body, "X-Forwarded-Host: www.example.com\n"))
				assert.True(strings.Contains(body, "X-Forwarded-Prefix: /app\n"))
				assert.True(strings.Contains(body, "X-Forwarded-Proto: https\n"))
				assert.True(strings.Contains(body, "X-Forwarded-For: 1.2.3.4\n"))
				assert.False(strings.Contains(body, "6.6.6.6"))
				assert.True(strings.Contains(body, "X-Forwarded-Path: /forwarded.headers/ok"))
			}
		}
	})

	t.Run("root", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := httpClient.Get("http://localhost:4040/")
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Expect(body, []byte("Root"))
			}
		}
	})

	t.Run("cors", func(t *testing.T) {
		assert := testarossa.For(t)

		// Request with no origin header
		count := callCount
		req, err := http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
		assert.NoError(err)
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal(count+1, callCount)
		}

		// Request with disallowed origin header
		count = callCount
		req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
		assert.NoError(err)
		req.Header.Set("Origin", "disallowed.origin")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusForbidden, res.StatusCode)
			assert.Equal(count, callCount)
		}

		// Request with allowed origin header
		count = callCount
		req, err = http.NewRequest("GET", "http://localhost:4040/cors/ok", nil)
		assert.NoError(err)
		req.Header.Set("Origin", "allowed.origin")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal("allowed.origin", res.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(count+1, callCount)
		}

		// Preflight request with allowed origin header
		count = callCount
		req, err = http.NewRequest("OPTIONS", "http://localhost:4040/cors/ok", nil)
		assert.NoError(err)
		req.Header.Set("Origin", "allowed.origin")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusNoContent, res.StatusCode)
			assert.Equal(count, callCount)
		}
	})

	t.Run("parse_form", func(t *testing.T) {
		assert := testarossa.For(t)

		// Under 10MB
		var buf bytes.Buffer
		buf.WriteString("x=")
		buf.WriteString(utils.RandomIdentifier(9 * 1024 * 1024))
		res, err := httpClient.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
			}
		}

		// Go sets a 10MB limit on forms by default
		// https://go.dev/src/net/http/request.go#L1258
		buf.WriteString(utils.RandomIdentifier(2 * 1024 * 1024)) // Now 11MB
		res, err = httpClient.Post("http://localhost:4040/parse.form/ok", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
		if assert.NoError(err) {
			assert.Equal(http.StatusRequestEntityTooLarge, res.StatusCode)
		}

		// MaxBytesReader can be used to extend the limit
		res, err = httpClient.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
			}
		}

		// Going above the MaxBytesReader limit
		buf.WriteString(utils.RandomIdentifier(2 * 1024 * 1024)) // Now 13MB
		res, err = httpClient.Post("http://localhost:4040/parse.form/more", "application/x-www-form-urlencoded", bytes.NewReader(buf.Bytes()))
		if assert.NoError(err) {
			assert.Equal(http.StatusRequestEntityTooLarge, res.StatusCode)
		}
	})

	t.Run("block_internal_headers", func(t *testing.T) {
		assert := testarossa.For(t)

		req, err := http.NewRequest("GET", "http://localhost:4040/internal.headers:5555/ok", nil)
		assert.NoError(err)
		req.Header.Set(frame.HeaderPrefix+"In-Request", "STOP")
		req.Header.Set(strings.ToUpper(frame.HeaderPrefix)+"In-Request-Upper", "STOP")
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			// No Microbus headers should be accepted from client
			assert.Equal("", request.Header.Get(frame.HeaderPrefix+"In-Request"))
			assert.Equal("", request.Header.Get(strings.ToUpper(frame.HeaderPrefix+"In-Request-Upper")))
			// Microbus headers generated internally should pass through the middleware chain
			assert.Equal(Hostname, frame.Of(request).FromHost())

			// No Microbus headers should leak outside
			assert.Equal("", res.Header.Get(frame.HeaderPrefix+"In-Response"))
			assert.Equal("", res.Header.Get(strings.ToUpper(frame.HeaderPrefix+"In-Request-Upper")))
			for h := range res.Header {
				assert.False(strings.HasPrefix(h, frame.HeaderPrefix))
			}
		}
	})

	t.Run("on_route", func(t *testing.T) {
		assert := testarossa.For(t)

		req, err := http.NewRequest("GET", "http://localhost:4040/greeting:5555/ok", nil)
		assert.NoError(err)
		req.Header.Set("Authorization", "Bearer 123456")
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
				// Headers should pass through
				assert.Equal("Bearer 123456", request.Header.Get("Authorization"))
				// Middleware added a request header
				assert.Equal("Hello", request.Header.Get("Middleware"))
				// Middleware added a response header
				assert.Equal("Goodbye", res.Header.Get("Middleware"))
			}
		}

		req, err = http.NewRequest("GET", "http://localhost:4040/greeting:5500/ok", nil)
		assert.NoError(err)
		req.Header.Set("Authorization", "Bearer 123456")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			b, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Equal("ok", string(b))
				// Headers should pass through
				assert.Equal("Bearer 123456", request.Header.Get("Authorization"))
				// Middleware did not run on this route
				assert.Equal("", request.Header.Get("Middleware"))
				assert.Equal("", res.Header.Get("Middleware"))
			}
		}
	})

	t.Run("blocked_paths", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := httpClient.Get("http://localhost:4040/blocked.paths/admin.php")
		if assert.NoError(err) {
			assert.Equal(http.StatusNotFound, res.StatusCode)
		}
		res, err = httpClient.Get("http://localhost:4040/blocked.paths/admin.ppp")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
		}
	})

	t.Run("default_fav_icon", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := httpClient.Get("http://localhost:4040/favicon.ico")
		if assert.NoError(err) {
			assert.Equal(http.StatusOK, res.StatusCode)
			assert.Equal("image/x-icon", res.Header.Get("Content-Type"))
			icon, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.NotZero(len(icon))
			}
		}
	})

	t.Run("no_cache_response_headers", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := httpClient.Get("http://localhost:4040/no.cache/ok")
		if assert.NoError(err) {
			assert.Contains(res.Header.Get("Cache-Control"), "no-cache")
			assert.Contains(res.Header.Get("Cache-Control"), "no-store")
			assert.Contains(res.Header.Get("Cache-Control"), "max-age=0")
		}
	})

	t.Run("auth_token_entry", func(t *testing.T) {
		assert := testarossa.For(t)

		now := time.Now().Truncate(time.Second)

		req, err := http.NewRequest("GET", "http://localhost:4040/auth.token.entry/ok", nil)
		assert.NoError(err)

		// No token
		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(0, countActors)

		// Token by unknown issuer
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": "my.issuer",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		})
		signedJWT, err := jwtToken.SignedString([]byte("some-key"))
		assert.NoError(err)
		req.Header.Set("Authorization", "Bearer "+signedJWT)

		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(0, countActors)

		// Attempt to impersonate issuer (wrong key, no kid)
		jwtToken = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": "https://" + bearertokenapi.Hostname,
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		})
		signedJWT, err = jwtToken.SignedString([]byte("wrong-key"))
		assert.NoError(err)
		req.Header.Set("Authorization", "Bearer "+signedJWT)

		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(0, countActors)

		// Do not accept incoming Microbus-Actor header
		req.Header.Del("Authorization")
		req.Header.Set(frame.HeaderActor, `{"iss":"https://`+bearertokenapi.Hostname+`"}`)

		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(0, countActors)

		// Valid as Authorization Bearer header
		signedJWT, err = bearertokenapi.NewClient(tester).Mint(ctx, nil)
		assert.NoError(err)
		req.Header.Del(frame.HeaderActor)
		req.Header.Set("Authorization", "Bearer "+signedJWT)

		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(1, countActors)

		// Also in Authorization cookie
		req.Header.Del("Authorization")
		req.AddCookie(&http.Cookie{
			Name:     "Authorization",
			Value:    signedJWT,
			MaxAge:   60,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
		})

		_, err = httpClient.Do(req)
		assert.NoError(err)
		assert.Equal(2, countActors)
	})

	t.Run("authorization", func(t *testing.T) {
		assert := testarossa.For(t)

		req, err := http.NewRequest("GET", "http://localhost:4040/authorization/protected", nil)
		assert.NoError(err)

		// Request not originating from a browser should be denied
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusUnauthorized, res.StatusCode)
		}

		// Request origination from a browser should be redirected to the login page
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Dest", "document")
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			body, _ := io.ReadAll(res.Body)
			assert.Equal("Login", string(body))
		}

		// Request with insufficient auth token should be rejected
		signedToken, err := bearertokenapi.NewClient(tester).Mint(ctx, map[string]any{
			"role": "minor",
		})
		assert.NoError(err)
		req.AddCookie(&http.Cookie{
			Name:     "Authorization",
			Value:    signedToken,
			MaxAge:   60,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
		})
		assert.Len(req.Cookies(), 1)
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			assert.Equal(http.StatusForbidden, res.StatusCode)
		}

		// Request with valid auth token should be served
		signedToken, err = bearertokenapi.NewClient(tester).Mint(ctx, map[string]any{
			"role": "major",
		})
		assert.NoError(err)
		req.Header.Del("Cookie")
		req.AddCookie(&http.Cookie{
			Name:     "Authorization",
			Value:    signedToken,
			MaxAge:   60,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
		})
		assert.Len(req.Cookies(), 1)
		res, err = httpClient.Do(req)
		if assert.NoError(err) {
			body, _ := io.ReadAll(res.Body)
			assert.Equal("Access Granted", string(body))
		}
	})

	t.Run("multi_value_headers", func(t *testing.T) {
		assert := testarossa.For(t)

		req, err := http.NewRequest("GET", "http://localhost:4040/multi.value.headers/ok", nil)
		assert.NoError(err)
		req.Header["Multi-Value"] = []string{
			"Send 1",
			"Send 2",
			"Send 3",
		}
		res, err := httpClient.Do(req)
		if assert.NoError(err) {
			if assert.Len(request.Header["Multi-Value"], 3) {
				assert.Equal("Send 1", request.Header["Multi-Value"][0])
				assert.Equal("Send 2", request.Header["Multi-Value"][1])
				assert.Equal("Send 3", request.Header["Multi-Value"][2])
			}
			if assert.Len(res.Header["Multi-Value"], 2) {
				assert.Equal("Return 1", res.Header["Multi-Value"][0])
				assert.Equal("Return 2", res.Header["Multi-Value"][1])
			}
		}
	})
}

func TestHttpingress_ResolveInternalURL(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// resolveInternalURL no longer rewrites ports; the path-embedded :port survives. Whether the
	// internal port is allowed is decided by isInternalPortAllowed downstream, not here.
	testCases := []string{
		"https://proxy:8080/service:5555/path?arg=val",
		"https://service:5555/path?arg=val",

		"https://proxy:8080/service:443/path",
		"https://service/path",

		"https://proxy:8080/service:80/path",
		"https://service:80/path",

		"https://proxy:8080/service/path",
		"https://service/path",

		"http://proxy:8080/service:5555/path",
		"https://service:5555/path",

		"https://proxy:443/service:5555/path",
		"https://service:5555/path",

		"https://proxy:443/service:443/path",
		"https://service/path",

		"https://proxy:443/service/path",
		"https://service/path",

		"https://proxy:80/service/path",
		"https://service/path",
	}
	for i := 0; i < len(testCases); i += 2 {
		x, err := url.Parse(testCases[i])
		assert.NoError(err)
		u, err := url.Parse(testCases[i+1])
		assert.NoError(err)
		ru, err := resolveInternalURL(x)
		assert.NoError(err)
		assert.Equal(u, ru)
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

func TestHTTPIngress_OnChangedPorts(t *testing.T) { // MARKER: Ports
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetPorts(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedAllowedOrigins(t *testing.T) { // MARKER: AllowedOrigins
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()
	svc.SetPorts("40911") // Avoid contention on the default port with parallel tests

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	t.Run("removed_config_refuses_non_empty_value", func(t *testing.T) {
		assert := testarossa.For(t)

		err := svc.SetAllowedOrigins("https://app.example")
		assert.Error(err, "AllowedOrigins has been split")
	})
}

func TestHTTPIngress_OnChangedPortMappings(t *testing.T) { // MARKER: PortMappings
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetPortMappings(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedAllowedInternalPorts(t *testing.T) { // MARKER: AllowedInternalPorts
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetAllowedInternalPorts(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedReadTimeout(t *testing.T) { // MARKER: ReadTimeout
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetReadTimeout(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedWriteTimeout(t *testing.T) { // MARKER: WriteTimeout
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetWriteTimeout(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedReadHeaderTimeout(t *testing.T) { // MARKER: ReadHeaderTimeout
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetReadHeaderTimeout(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedBlockedPaths(t *testing.T) { // MARKER: BlockedPaths
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := svc.SetBlockedPaths(value)
			assert.NoError(err)
		})
	*/
}

func TestHTTPIngress_OnChangedAllowedCredentialedOrigins(t *testing.T) { // MARKER: AllowedCredentialedOrigins
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()
	svc.SetPorts("40912") // Avoid contention on the default port with parallel tests

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	t.Run("named_origins_accepted", func(t *testing.T) {
		assert := testarossa.For(t)

		err := svc.SetAllowedCredentialedOrigins("https://app.example, https://admin.example")
		assert.NoError(err)
		assert.True(svc.credentialedOrigins["https://app.example"])
		assert.True(svc.credentialedOrigins["https://admin.example"])
	})

	t.Run("wildcard_rejected", func(t *testing.T) {
		assert := testarossa.For(t)

		err := svc.SetAllowedCredentialedOrigins("https://app.example, *")
		assert.Error(err, "cannot be credentialed")
	})
}

func TestHTTPIngress_OnChangedAllowedUncredentialedOrigins(t *testing.T) { // MARKER: AllowedUncredentialedOrigins
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()
	svc.SetPorts("40913") // Avoid contention on the default port with parallel tests

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
	)
	app.RunInTest(t)

	t.Run("wildcard_and_named_origins_accepted", func(t *testing.T) {
		assert := testarossa.For(t)

		err := svc.SetAllowedUncredentialedOrigins("https://reader.example, *")
		assert.NoError(err)
		assert.True(svc.uncredentialedOrigins["https://reader.example"])
		assert.True(svc.uncredentialedOrigins["*"])
	})
}

// TestHTTPIngress_BearerKeyRotationEvictsStaleKey verifies that a successful bearer-token JWKS fetch
// replaces the issuer's cached key set rather than merging into it, so a kid the issuer has rotated
// out of its JWKS stops being trusted. Without eviction a rotated-out (e.g. compromised) bearer key
// would remain usable for the life of the process.
func TestHTTPIngress_BearerKeyRotationEvictsStaleKey(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	kidOf := func(pub ed25519.PublicKey) string {
		hash := sha256.Sum256(pub)
		return base64.RawURLEncoding.EncodeToString(hash[:8])
	}

	// Two key pairs: the issuer serves key1 first, then rotates to key2 only.
	pub1, _, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(err)
	kid1, kid2 := kidOf(pub1), kidOf(pub2)

	// The mocked bearer token service serves whichever key `served`/`servedKid` point at.
	served, servedKid := pub1, kid1
	bearerMock := bearertoken.NewMock()
	bearerMock.MockJWKS(func(ctx context.Context) (keys []bearertokenapi.JWK, err error) {
		return []bearertokenapi.JWK{{
			KTY: "OKP",
			CRV: "Ed25519",
			X:   base64.RawURLEncoding.EncodeToString(served),
			KID: servedKid,
		}}, nil
	})

	svc := NewService()
	tester := connector.New("tester.client")
	_ = tester

	app := application.New()
	app.Add(
		bearerMock,
		svc,
		tester,
	)
	app.RunInTest(t)

	host := bearertokenapi.Hostname

	// First fetch caches key1.
	err = svc.fetchBearerTokenKeys(ctx, host)
	assert.NoError(err)
	_, found := svc.lookupBearerTokenKey(host, kid1)
	assert.True(found, "key1 should be cached after the first fetch")

	// The issuer rotates key1 out; it now publishes only key2.
	served, servedKid = pub2, kid2

	// Clear the 1s debounce so the next fetch actually reaches the issuer.
	svc.bearerTokenMu.Lock()
	delete(svc.lastJWKSFetch, host)
	svc.bearerTokenMu.Unlock()

	// The rotation fetch replaces the issuer's key set: key2 is cached, key1 is evicted.
	err = svc.fetchBearerTokenKeys(ctx, host)
	assert.NoError(err)
	_, found = svc.lookupBearerTokenKey(host, kid2)
	assert.True(found, "key2 should be cached after the rotation fetch")
	_, found = svc.lookupBearerTokenKey(host, kid1)
	assert.False(found, "key1 must be evicted once the issuer stops publishing it")
}

// TestHTTPIngress_PortInUse verifies that startHTTPServers reports a bind failure rather than
// falsely reporting a healthy start. A first ingress binds :4040; a second ingress on the same
// port must fail Startup because the bind (net.Listen) now surfaces synchronously.
func TestHTTPIngress_PortInUse(t *testing.T) {
	// No t.Parallel: binds a real OS port shared with other non-parallel ingress tests.
	assert := testarossa.For(t)

	// First ingress binds :4040 and keeps serving for the test's lifetime.
	svc1 := NewService()
	svc1.SetPorts("4040")
	app := application.New()
	app.Add(svc1)
	app.RunInTest(t)

	// Second ingress on the same port. Its startup must fail on the bind, not race a timer.
	svc2 := NewService()
	svc2.SetPorts("4040")
	svc2.SetDeployment(connector.TESTING)
	svc2.SetPlane("portinuse")

	startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := svc2.Startup(startCtx)
	if assert.Error(err, "second ingress must fail to bind an in-use port") {
		assert.Contains(strings.ToLower(err.Error()), "address already in use")
	} else {
		svc2.Shutdown(startCtx)
	}
}
