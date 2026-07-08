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

package middleware

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/testarossa"
)

func TestCors_AllowedOrigin(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) {
		if origin == "https://allowed.example" {
			return origin, true
		}
		return "", false
	})

	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example/x", nil)
	r.Header.Set("Origin", "https://allowed.example")
	err := mw(func(w http.ResponseWriter, r *http.Request) error { return nil })(w, r)
	assert.NoError(err)
	assert.Equal("https://allowed.example", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal("true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(w.Header().Values("Vary"), "Origin")
}

func TestCors_WildcardOriginIsUncredentialed(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) {
		return "*", false
	})

	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example/x", nil)
	r.Header.Set("Origin", "https://anywhere.example")
	err := mw(func(w http.ResponseWriter, r *http.Request) error { return nil })(w, r)
	assert.NoError(err)
	assert.Equal("*", w.Header().Get("Access-Control-Allow-Origin"))
	_, hasCredentials := w.Header()["Access-Control-Allow-Credentials"]
	assert.False(hasCredentials)
	assert.Equal("*", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal("*, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal("*", w.Header().Get("Access-Control-Expose-Headers"))
}

func TestCors_NamedUncredentialedOrigin(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) {
		return origin, false
	})

	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example/x", nil)
	r.Header.Set("Origin", "https://reader.example")
	err := mw(func(w http.ResponseWriter, r *http.Request) error { return nil })(w, r)
	assert.NoError(err)
	assert.Equal("https://reader.example", w.Header().Get("Access-Control-Allow-Origin"))
	_, hasCredentials := w.Header()["Access-Control-Allow-Credentials"]
	assert.False(hasCredentials)
	assert.Contains(w.Header().Values("Vary"), "Origin")
}

func TestCors_RejectedOrigin(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) { return "", false })

	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example/x", nil)
	r.Header.Set("Origin", "https://attacker.example")
	err := mw(func(w http.ResponseWriter, r *http.Request) error {
		t.Fatal("downstream handler must not be called for a rejected origin")
		return nil
	})(w, r)
	assert.Error(err)
}

func TestCors_NoOriginPassesThrough(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) {
		t.Fatal("allowedOrigin must not be consulted when Origin is absent")
		return "", false
	})

	called := false
	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example/x", nil)
	err := mw(func(w http.ResponseWriter, r *http.Request) error {
		called = true
		return nil
	})(w, r)
	assert.NoError(err)
	assert.True(called)
	assert.Equal("", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCors_SameOriginPinningFromRequest(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// Simulates the default config: pin ACAO to scheme://host derived from
	// the request itself. X-Forwarded-* must be ignored so an edge attacker
	// cannot inflate ACAO to a host they control.
	allow := func(r *http.Request, origin string) (string, bool) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		return scheme + "://" + r.Host, false
	}
	mw := Cors(allow)

	// Plain HTTP request: scheme is derived from r.TLS == nil.
	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("GET", "http://ingress.example:4040/x", nil)
	r.Host = "ingress.example:4040"
	r.Header.Set("Origin", "https://attacker.example")
	r.Header.Set("X-Forwarded-Host", "attacker.tld")
	r.Header.Set("X-Forwarded-Proto", "https")
	err := mw(func(w http.ResponseWriter, r *http.Request) error { return nil })(w, r)
	assert.NoError(err)
	assert.Equal("http://ingress.example:4040", w.Header().Get("Access-Control-Allow-Origin"))

	// TLS request: scheme flips to https.
	w = httpx.NewResponseRecorder()
	r, _ = http.NewRequest("GET", "https://ingress.example/x", nil)
	r.Host = "ingress.example"
	r.TLS = &tls.ConnectionState{}
	r.Header.Set("Origin", "https://attacker.example")
	err = mw(func(w http.ResponseWriter, r *http.Request) error { return nil })(w, r)
	assert.NoError(err)
	assert.Equal("https://ingress.example", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCors_PreflightShortCircuits(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	mw := Cors(func(r *http.Request, origin string) (string, bool) { return origin, true })

	w := httpx.NewResponseRecorder()
	r, _ := http.NewRequest("OPTIONS", "http://ingress.example/x", nil)
	r.Header.Set("Origin", "https://allowed.example")
	r.Header.Set("Access-Control-Request-Method", "PUT")
	r.Header.Set("Access-Control-Request-Headers", "content-type, x-custom")
	err := mw(func(w http.ResponseWriter, r *http.Request) error {
		t.Fatal("preflight must not call the downstream handler")
		return nil
	})(w, r)
	assert.NoError(err)
	assert.Equal(http.StatusNoContent, w.Result().StatusCode)
	// In credentialed mode wildcards are literal, so the preflight's method and headers are reflected
	assert.Equal("PUT", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal("content-type, x-custom", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Contains(w.Header().Values("Vary"), "Access-Control-Request-Method")
	assert.Contains(w.Header().Values("Vary"), "Access-Control-Request-Headers")
}
