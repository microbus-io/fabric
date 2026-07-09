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

package transport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"
	"github.com/nats-io/nats.go"
)

func BenchmarkTransport_NATSDirectPublishing(b *testing.B) {
	assert := testarossa.For(b)

	cn, err := nats.Connect("https://127.0.0.1:4222")
	assert.NoError(err)
	defer cn.Close()

	s, err := cn.Subscribe("somewhere", func(msg *nats.Msg) {})
	assert.NoError(err)
	defer s.Unsubscribe()

	for i := 128; i <= 512<<10; i *= 2 {
		desc := fmt.Sprintf("%dB", i)
		if i >= 1<<10 {
			desc = fmt.Sprintf("%dKB", i>>10)
		}
		b.Run(desc, func(b *testing.B) {
			body := []byte(utils.RandomIdentifier(i))
			b.ResetTimer()
			for b.Loop() {
				cn.Publish("somewhere", body)
			}
		})
	}

	// goos: darwin
	// goarch: arm64
	// pkg: github.com/microbus-io/fabric/transport
	// cpu: Apple M1 Pro
	// BenchmarkTransport_NATSDirectPublishing/128B-10         	 2970085	       392.6 ns/op	     255 B/op	       2 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/256B-10         	 2683281	       442.1 ns/op	     384 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/512B-10         	 2212436	       546.4 ns/op	     639 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/1KB-10          	 2101729	       573.4 ns/op	    1132 B/op	       2 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/2KB-10          	 1337725	       896.5 ns/op	    2173 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/4KB-10          	  784214	      1667 ns/op	    4222 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/8KB-10          	  370716	      3179 ns/op	    8298 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/16KB-10         	  193784	      6885 ns/op	   16723 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/32KB-10         	   80478	     15101 ns/op	   32875 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/64KB-10         	   52527	     20811 ns/op	   64890 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/128KB-10        	   31376	     38782 ns/op	  131063 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/256KB-10        	   15942	     75784 ns/op	  262136 B/op	       3 allocs/op
	// BenchmarkTransport_NATSDirectPublishing/512KB-10        	    8608	    146453 ns/op	  524411 B/op	       3 allocs/op
}

func TestTransport_LingeringSubscriptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assert := testarossa.For(t)

	h := func(msg *Msg) {}
	var c Conn
	err := c.Open(ctx, "", nil)
	assert.NoError(err)

	s1, err := c.Subscribe("s1.subject", h)
	assert.NoError(err)
	s2, err := c.Subscribe("s2.subject", h)
	assert.NoError(err)
	s3, err := c.Subscribe("s3.subject", h)
	assert.NoError(err)
	// 3 -> 2 -> 1
	assert.Equal(s3, c.head)
	assert.Equal(s2, c.head.next)
	assert.Equal(s1, c.head.next.next)
	assert.Nil(c.head.next.next.next)
	assert.Nil(s3.prev)
	assert.Equal(s3.next, s2)
	assert.Equal(s2.prev, s3)
	assert.Equal(s2.next, s1)
	assert.Equal(s1.prev, s2)
	assert.Nil(s1.next)

	assert.False(s2.done)
	if c.shortCircuitEnabled.Load() {
		assert.NotNil(s2.shortCircuitUnsub)
	}
	err = s2.Unsubscribe()
	assert.NoError(err)
	err = s2.Unsubscribe()
	assert.NoError(err)
	assert.True(s2.done)
	if c.shortCircuitEnabled.Load() {
		assert.Nil(s2.shortCircuitUnsub)
	}
	// 3 -> 1
	assert.Equal(s3, c.head)
	assert.Equal(s1, c.head.next)
	assert.Nil(c.head.next.next)
	assert.Nil(s3.prev)
	assert.Equal(s3.next, s1)
	assert.Nil(s2.prev)
	assert.Nil(s2.next)
	assert.Equal(s1.prev, s3)
	assert.Nil(s1.next)

	err = s3.Unsubscribe()
	assert.NoError(err)
	s4, err := c.Subscribe("s4.subject", h)
	assert.NoError(err)
	err = s4.Unsubscribe()
	assert.NoError(err)
	// 1
	assert.Equal(s1, c.head)
	assert.Nil(c.head.next)
	assert.Nil(s4.prev)
	assert.Nil(s4.next)
	assert.Nil(s3.prev)
	assert.Nil(s3.next)
	assert.Nil(s2.prev)
	assert.Nil(s2.next)
	assert.Nil(s1.prev)
	assert.Nil(s1.next)

	if c.shortCircuitEnabled.Load() {
		assert.False(shortCircuit.IsEmpty())
	}
	err = c.Close()
	assert.NoError(err)
	assert.Nil(c.head)
	if c.shortCircuitEnabled.Load() {
		assert.True(shortCircuit.IsEmpty())
	}
}

// TestTransport_ResolveArtifact pins the per-service-then-bare lookup chain
// for the four auth artifacts. The temp dir contains both forms of nats.creds
// and only the bare form of cert.pem; the resolver should pick correctly per
// hostname and per artifact.
func TestTransport_ResolveArtifact(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	dir := t.TempDir()
	prevWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("foo.example_nats.creds", "per-service")
	write("nats.creds", "shared")
	write("cert.pem", "shared-cert")

	cases := []struct {
		hostname string
		artifact string
		want     string
	}{
		// per-service file present → use it
		{"foo.example", "nats.creds", "foo.example_nats.creds"},
		// per-service file absent, bare default present → fall back
		{"bar.example", "nats.creds", "nats.creds"},
		// only bare file exists, regardless of hostname
		{"foo.example", "cert.pem", "cert.pem"},
		{"bar.example", "cert.pem", "cert.pem"},
		// neither form exists → empty
		{"foo.example", "key.pem", ""},
		{"", "key.pem", ""},
		// empty hostname disables per-service lookup; bare default still wins
		{"", "nats.creds", "nats.creds"},
	}
	for _, c := range cases {
		got := resolveArtifact(c.hostname, c.artifact)
		assert.Equal(c.want, got, fmt.Sprintf("hostname=%q artifact=%q", c.hostname, c.artifact))
	}
}

// TestTransport_Secure verifies Secure() in both deployment modes the CI matrix runs: with no NATS
// (short-circuit only) it reports secure by construction, and with a NATS connection it agrees with
// the connection's actual TLS state. It asserts no fixed value for the NATS case so it holds whether
// CI points at a plaintext or a TLS NATS.
func TestTransport_Secure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assert := testarossa.For(t)

	// Not opened: no NATS connection, secure by construction.
	var c Conn
	assert.True(c.Secure(), "an unopened transport has no wire and is secure")

	err := c.Open(ctx, "", nil)
	assert.NoError(err)
	defer c.Close()

	if nc := c.natsConn.Load(); nc == nil {
		// Short-circuit only (no MICROBUS_NATS): still secure.
		assert.True(c.Secure(), "a short-circuit-only transport is secure by construction")
	} else {
		// Connected to NATS: Secure() must match the connection's real TLS state.
		_, tlsErr := nc.TLSConnectionState()
		assert.Equal(tlsErr == nil, c.Secure(), "Secure() must reflect the NATS connection's TLS state")
	}
}
