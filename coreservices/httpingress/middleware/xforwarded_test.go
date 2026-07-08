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
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/testarossa"
)

// applyXForwarded runs a request through the XForwarded middleware and returns the headers seen downstream.
func applyXForwarded(t *testing.T, hops int, r *http.Request) http.Header {
	var captured http.Header
	mw := XForwarded(func() int { return hops })
	w := httpx.NewResponseRecorder()
	err := mw(func(w http.ResponseWriter, r *http.Request) error {
		captured = r.Header
		return nil
	})(w, r)
	testarossa.For(t).NoError(err)
	return captured
}

func TestXForwarded_EdgeIgnoresInboundHeaders(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	r, _ := http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Host = "ingress.example"
	r.RemoteAddr = "203.0.113.9:51000"
	r.Header.Set("X-Forwarded-Host", "www.spoofed.example")
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-For", "6.6.6.6")
	r.Header.Set("X-Forwarded-Prefix", "/spoof")
	r.Header.Set("X-Forwarded-Whatever", "junk")

	h := applyXForwarded(t, 0, r)
	assert.Equal("ingress.example", h.Get("X-Forwarded-Host"))
	assert.Equal("http", h.Get("X-Forwarded-Proto"))
	assert.Equal("203.0.113.9:51000", h.Get("X-Forwarded-For"))
	assert.Equal("", h.Get("X-Forwarded-Prefix"))
	assert.Equal("", h.Get("X-Forwarded-Whatever"))
	assert.Equal("/svc/path", h.Get("X-Forwarded-Path"))
}

func TestXForwarded_TrustedHopsTruncateForChain(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// One trusted proxy: the client-fabricated prefix of the chain is discarded
	r, _ := http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Header.Set("X-Forwarded-Host", "www.example.com")
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")
	h := applyXForwarded(t, 1, r)
	assert.Equal("www.example.com", h.Get("X-Forwarded-Host"))
	assert.Equal("https", h.Get("X-Forwarded-Proto"))
	assert.Equal("203.0.113.9", h.Get("X-Forwarded-For"))

	// Two trusted proxies: the client is the 2nd entry from the right
	r, _ = http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9, 198.51.100.7")
	h = applyXForwarded(t, 2, r)
	assert.Equal("203.0.113.9, 198.51.100.7", h.Get("X-Forwarded-For"))

	// A chain shorter than the trusted hops is passed as is
	r, _ = http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	h = applyXForwarded(t, 2, r)
	assert.Equal("203.0.113.9", h.Get("X-Forwarded-For"))

	// No inbound chain at all: fall back to the peer address
	r, _ = http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.RemoteAddr = "198.51.100.7:44000"
	h = applyXForwarded(t, 1, r)
	assert.Equal("198.51.100.7:44000", h.Get("X-Forwarded-For"))
}

func TestXForwarded_PrefixesCombineOutermostFirst(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// Multiple prefixes, whether comma-listed or on separate header lines, combine in order
	r, _ := http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Header.Set("X-Forwarded-Prefix", "/x, /y")
	r.Header.Add("X-Forwarded-Prefix", "/z/")
	h := applyXForwarded(t, 1, r)
	assert.Equal("/x/y/z", h.Get("X-Forwarded-Prefix"))

	// The junk and multi-value inbound headers never survive; exactly one value goes downstream
	assert.Equal(1, len(h.Values("X-Forwarded-Prefix")))
}

func TestXForwarded_HostAndProtoFirstValueWins(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	// Appending proxies (e.g. Apache) produce comma lists; the outermost proxy writes first
	r, _ := http.NewRequest("GET", "http://ingress.example/svc/path", nil)
	r.Header.Set("X-Forwarded-Host", "user-facing.example, internal-lb")
	r.Header.Set("X-Forwarded-Proto", "https, http")
	h := applyXForwarded(t, 2, r)
	assert.Equal("user-facing.example", h.Get("X-Forwarded-Host"))
	assert.Equal("https", h.Get("X-Forwarded-Proto"))
}
