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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/mem"
	"github.com/nats-io/nats.go"
)

var (
	shortCircuit trie
	pool         = map[string]*refCountedNATSConn{}
	poolLock     sync.Mutex
)

// refCountedNATSConn counts references to the real NATS connection.
type refCountedNATSConn struct {
	*nats.Conn
	refCount int
	cacheKey string
}

// Logger is used by the transport to log messages in the caller's context.
type Logger interface {
	LogInfo(ctx context.Context, msg string, args ...any)
	LogError(ctx context.Context, msg string, args ...any)
}

// Conn abstracts the connection to the transport.
type Conn struct {
	natsConn            atomic.Pointer[refCountedNATSConn]
	shortCircuitEnabled atomic.Bool
	head                *Subscription
	mux                 sync.Mutex
}

// resolveArtifact returns the path of an auth artifact in CWD: the per-service
// form {hostname}_{artifact} if it exists, else the bare {artifact}, else "".
func resolveArtifact(hostname, artifact string) string {
	stat := func(p string) bool { _, err := os.Stat(p); return err == nil }
	if hostname != "" {
		if prefixed := hostname + "_" + artifact; stat(prefixed) {
			return prefixed
		}
	}
	if stat(artifact) {
		return artifact
	}
	return ""
}

// Open opens the transport. hostname drives per-service auth-artifact lookup (pass "" to skip).
func (c *Conn) Open(ctx context.Context, hostname string, logger Logger) error {
	if v := env.Get("MICROBUS_SHORT_CIRCUIT"); v == "0" || strings.EqualFold(v, "false") {
		c.shortCircuitEnabled.Store(false)
	} else {
		c.shortCircuitEnabled.Store(true)
	}

	// URL
	u := env.Get("MICROBUS_NATS")
	if u == "" && !c.shortCircuitEnabled.Load() {
		u = "nats://127.0.0.1:4222"
	}
	if u == "" {
		if logger != nil {
			logger.LogInfo(ctx, "Using short-circuit only")
		}
		return nil
	}
	opts := []nats.Option{}

	hasher := sha256.New()
	hasher.Write([]byte(u))
	hasher.Write([]byte{'|'})

	// Deprecated env-var auth. Kept functional for backward compatibility;
	user := env.Get("MICROBUS_NATS_USER")
	pw := env.Get("MICROBUS_NATS_PASSWORD")
	token := env.Get("MICROBUS_NATS_TOKEN")
	userJWT := env.Get("MICROBUS_NATS_USER_JWT")
	seed := env.Get("MICROBUS_NATS_NKEY_SEED")
	if logger != nil {
		var deprecated []string
		for _, name := range []string{
			"MICROBUS_NATS_USER", "MICROBUS_NATS_PASSWORD", "MICROBUS_NATS_TOKEN",
			"MICROBUS_NATS_USER_JWT", "MICROBUS_NATS_NKEY_SEED",
		} {
			if env.Get(name) != "" {
				deprecated = append(deprecated, name)
			}
		}
		if len(deprecated) > 0 {
			logger.LogInfo(ctx,
				"Deprecated NATS env-var auth in use; switch to <hostname>_nats.creds or nats.creds files",
				"env", strings.Join(deprecated, ","),
			)
		}
	}
	if user != "" && pw != "" {
		hasher.Write([]byte(user))
		hasher.Write([]byte{'|'})
		hasher.Write([]byte(pw))
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.UserInfo(user, pw))
	}
	if token != "" {
		hasher.Write([]byte(token))
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.Token(token))
	}
	if userJWT != "" && seed != "" {
		hasher.Write([]byte(userJWT))
		hasher.Write([]byte{'|'})
		hasher.Write([]byte(seed))
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.UserJWTAndSeed(userJWT, seed))
	}

	// Root CA, client cert/key, and NATS user creds.
	if caPath := resolveArtifact(hostname, "ca.pem"); caPath != "" {
		content, _ := os.ReadFile(caPath)
		hasher.Write(content)
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.RootCAs(caPath))
	}
	certPath := resolveArtifact(hostname, "cert.pem")
	keyPath := resolveArtifact(hostname, "key.pem")
	if certPath != "" && keyPath != "" {
		content, _ := os.ReadFile(certPath)
		hasher.Write(content)
		hasher.Write([]byte{'|'})
		content, _ = os.ReadFile(keyPath)
		hasher.Write(content)
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.ClientCert(certPath, keyPath))
	}
	if credsPath := resolveArtifact(hostname, "nats.creds"); credsPath != "" {
		content, _ := os.ReadFile(credsPath)
		hasher.Write(content)
		hasher.Write([]byte{'|'})
		opts = append(opts, nats.UserCredentials(credsPath))
	}

	// Other options
	if logger != nil {
		opts = append(opts, nats.ErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
			sub := ""
			if s != nil {
				sub = s.Subject
			}
			logger.LogError(ctx, err.Error(),
				"subject", sub,
				"server", c.ConnectedServerId(),
			)
		}))
	}

	// Check connection pool first
	cacheKey := hex.EncodeToString(hasher.Sum(nil))
	poolLock.Lock()
	cached := pool[cacheKey]
	if cached != nil {
		cached.refCount++
	}
	poolLock.Unlock()
	if cached != nil {
		c.natsConn.Store(cached)
		return nil
	}

	// Connect
	cn, err := nats.Connect(u, opts...)
	if err != nil {
		return errors.Trace(err, "url", u)
	}

	// Log connection events
	if logger != nil {
		natsURL := cn.ConnectedUrl()
		natsServerID := cn.ConnectedServerId()
		logger.LogInfo(ctx, "Connected to NATS",
			"url", natsURL,
			"server", natsServerID,
		)
		cn.SetDisconnectErrHandler(func(cn *nats.Conn, err error) {
			logger.LogInfo(ctx, "Disconnected from NATS",
				"url", natsURL,
				"server", natsServerID,
			)
		})
		cn.SetReconnectHandler(func(cn *nats.Conn) {
			natsURL = cn.ConnectedUrl()
			natsServerID = cn.ConnectedServerId()
			logger.LogInfo(ctx, "Reconnected to NATS",
				"url", natsURL,
				"server", natsServerID,
			)
		})
	}

	// Pool the connection for later
	poolLock.Lock()
	cached = &refCountedNATSConn{
		Conn:     cn,
		refCount: 1,
		cacheKey: cacheKey,
	}
	pool[cacheKey] = cached
	poolLock.Unlock()

	c.natsConn.Store(cached)
	return nil
}

// Close closes the transport.
// It closes the NATS connection, if appropriate.
func (c *Conn) Close() error {
	// Lingering subscriptions
	for {
		c.mux.Lock()
		sub := c.head
		c.mux.Unlock()
		if sub == nil {
			break
		}
		_ = c.unsubscribe(sub)
	}
	// Disconnect from NATS
	natsConn := c.natsConn.Load()
	if natsConn != nil {
		poolLock.Lock()
		natsConn.refCount--
		lastRef := natsConn.refCount == 0
		if lastRef {
			delete(pool, natsConn.cacheKey)
		}
		poolLock.Unlock()
		if lastRef {
			natsConn.Close()
			c.natsConn.Store(nil)
		}
	}
	// Disable short circuit
	c.shortCircuitEnabled.Store(false)
	return nil
}

// MaxPayload returns the size limit that a message payload can have.
// In NATS's case, this is set by the server configuration and delivered to the client upon connect.
func (c *Conn) MaxPayload() int64 {
	natsConn := c.natsConn.Load()
	if natsConn != nil {
		return natsConn.MaxPayload()
	} else {
		return 1 << 20
	}
}

// Secure reports whether the transport carries messages over an encrypted path.
// The in-process short-circuit transport never touches a wire, so a short-circuit-only
// deployment (no NATS connection) is secure by construction. A NATS connection is secure
// only when it is TLS.
func (c *Conn) Secure() bool {
	natsConn := c.natsConn.Load()
	if natsConn == nil {
		return true
	}
	_, err := natsConn.TLSConnectionState()
	return err == nil
}

// Publish sends data to a subject, allowing for multiple recipients.
func (c *Conn) Publish(subject string, httpReq *http.Request) (err error) {
	natsConn := c.natsConn.Load()
	shortCircuitEnabled := c.shortCircuitEnabled.Load()
	if !shortCircuitEnabled && natsConn == nil {
		return errors.New("no transport")
	}

	if natsConn != nil {
		sz := 1<<10 + 1 + int(httpReq.ContentLength) // 2KB block minimum
		block := mem.Alloc(sz)
		defer mem.Free(block)
		buf := bytes.NewBuffer(block)
		err = httpReq.WriteProxy(buf)
		if err != nil {
			return errors.Trace(err)
		}
		err = natsConn.Publish(subject, buf.Bytes())
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}

	// Use short-circuit for multicast only if NATS is disabled, because all subscribers, not just local ones, must be reached
	if shortCircuitEnabled {
		_, err = c.deliverWithShortCircuit(subject, &Msg{Subject: subject, Request: httpReq})
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}

	return nil
}

// Request sends data to a subject, targeting only a single recipient.
func (c *Conn) Request(subject string, httpReq *http.Request) (err error) {
	natsConn := c.natsConn.Load()
	shortCircuitEnabled := c.shortCircuitEnabled.Load()
	if !shortCircuitEnabled && natsConn == nil {
		return errors.New("no transport")
	}

	if shortCircuitEnabled {
		// Try over short circuit
		ok, err := c.deliverWithShortCircuit(subject, &Msg{Subject: subject, Request: httpReq})
		if err != nil {
			return errors.Trace(err)
		}
		if ok {
			return nil
		}
		// If !ok, try with NATS
	}

	// Go over NATS
	if natsConn != nil {
		sz := 1<<10 + 1 + int(httpReq.ContentLength) // 2KB block minimum
		block := mem.Alloc(sz)
		defer mem.Free(block)
		buf := bytes.NewBuffer(block)
		err = httpReq.WriteProxy(buf)
		if err != nil {
			return errors.Trace(err)
		}
		err = natsConn.Publish(subject, buf.Bytes())
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}

	return nil
}

// Response sends a response to a subject, targeting only a single recipient.
func (c *Conn) Response(subject string, httpRes *http.Response) (err error) {
	natsConn := c.natsConn.Load()
	shortCircuitEnabled := c.shortCircuitEnabled.Load()
	if !shortCircuitEnabled && natsConn == nil {
		return errors.New("no transport")
	}

	if shortCircuitEnabled {
		// Try over short circuit
		ok, err := c.deliverWithShortCircuit(subject, &Msg{Subject: subject, Response: httpRes})
		if err != nil {
			return errors.Trace(err)
		}
		if ok {
			return nil
		}
		// If !ok, try with NATS
	}

	// Go over NATS
	if natsConn != nil {
		sz := 1<<10 + 1 + int(httpRes.ContentLength) // 2KB block minimum
		block := mem.Alloc(sz)
		defer mem.Free(block)
		buf := bytes.NewBuffer(block)
		err = httpRes.Write(buf)
		if err != nil {
			return errors.Trace(err)
		}
		err = natsConn.Publish(subject, buf.Bytes())
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}

	return nil
}

// deliverWithShortCircuit delivers the message via the short-circuit, if appropriate.
func (c *Conn) deliverWithShortCircuit(subject string, msg *Msg) (delivered bool, err error) {
	handlers := shortCircuit.Handlers(subject)
	if len(handlers) == 0 {
		// No handlers available locally, try via NATS
		return false, nil
	}
	if len(handlers) == 1 {
		// A single handler can be called directly with the request or response object
		handlers[0](msg)
		return true, nil
	}
	if msg.Request != nil {
		sz := 1<<10 + 1 + int(msg.Request.ContentLength)
		buf := bytes.NewBuffer(make([]byte, 0, sz))
		err := msg.Request.WriteProxy(buf)
		if err != nil {
			return false, errors.Trace(err)
		}
		// Every subscriber gets a separate copy of the request
		msg.Data = buf.Bytes()
		msg.Request = nil
	} else if msg.Response != nil {
		sz := 1<<10 + 1 + int(msg.Response.ContentLength)
		buf := bytes.NewBuffer(make([]byte, 0, sz))
		err := msg.Response.Write(buf)
		if err != nil {
			return false, errors.Trace(err)
		}
		// Every subscriber gets a separate copy of the response
		msg.Data = buf.Bytes()
		msg.Response = nil
	}
	if msg.Data != nil {
		for _, h := range handlers {
			h(&Msg{Subject: subject, Data: msg.Data})
		}
		return true, nil
	}
	return false, errors.New("malformed msg")
}

// QueueSubscribe expresses interest in a subject, which may contain wildcards.
// All subscribers with the same queue name form the queue group and only one member
// of the group is selected to receive any given message.
// The asterisk wildcard matches a single segment of the subject,
// e.g. america.usa.* will match america.usa.ca but not america.usa.ca.sfo.
// The gt wildcard must come at the end of the subject and matches any number of segments, e.g.
// e.g. america.usa.> will match america.usa.ca and america.usa.ca.sfo.
func (c *Conn) QueueSubscribe(subject string, queue string, handler MsgHandler) (sub *Subscription, err error) {
	sub = &Subscription{
		conn: c,
	}

	natsConn := c.natsConn.Load()
	if natsConn != nil {
		sub.natsSub, err = natsConn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
			handler(&Msg{Subject: msg.Subject, Data: msg.Data})
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		sub.natsSub.SetPendingLimits(-1, -1)
	}

	if c.shortCircuitEnabled.Load() {
		sub.shortCircuitUnsub = shortCircuit.Sub(subject, queue, handler)
	}

	c.mux.Lock()
	if c.head == nil {
		c.head = sub
	} else {
		c.head.prev = sub
		sub.next = c.head
		c.head = sub
	}
	c.mux.Unlock()

	return sub, nil
}

// Subscribe expresses interest in a subject, which may contain wildcards.
// All subscribers receive all messages.
// The asterisk wildcard matches a single segment of the subject,
// e.g. america.usa.* will match america.usa.ca but not america.usa.ca.sfo.
// The gt wildcard must come at the end of the subject and matches any number of segments, e.g.
// e.g. america.usa.> will match america.usa.ca and america.usa.ca.sfo.
func (c *Conn) Subscribe(subject string, handler MsgHandler) (sub *Subscription, err error) {
	sub = &Subscription{
		conn: c,
	}

	natsConn := c.natsConn.Load()
	if natsConn != nil {
		sub.natsSub, err = natsConn.Subscribe(subject, func(msg *nats.Msg) {
			handler(&Msg{Subject: msg.Subject, Data: msg.Data})
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		sub.natsSub.SetPendingLimits(-1, -1)
	}

	if c.shortCircuitEnabled.Load() {
		sub.shortCircuitUnsub = shortCircuit.Sub(subject, "", handler)
	}

	c.mux.Lock()
	if c.head == nil {
		c.head = sub
	} else {
		c.head.prev = sub
		sub.next = c.head
		c.head = sub
	}
	c.mux.Unlock()

	return sub, nil
}

// WaitForSub gives a bit of time for the subscription to be registered with NATS.
// It is a no op if NATS is not enabled.
func (c *Conn) WaitForSub() {
	natsConn := c.natsConn.Load()
	if natsConn != nil {
		time.Sleep(20 * time.Millisecond)
	}
}

// unsubscribe removes interest in the subject of the subscription.
func (c *Conn) unsubscribe(sub *Subscription) (err error) {
	if sub.done {
		return nil
	}
	natsSub := sub.natsSub
	if natsSub != nil {
		err = natsSub.Unsubscribe()
		if err != nil {
			return errors.Trace(err)
		}
		sub.natsSub = nil
	}
	shortCircuitUnsub := sub.shortCircuitUnsub
	if shortCircuitUnsub != nil {
		shortCircuitUnsub()
		sub.shortCircuitUnsub = nil
	}

	c.mux.Lock()
	if sub.prev == nil {
		if c.head == sub {
			c.head = sub.next
		}
	} else {
		sub.prev.next = sub.next
	}
	if sub.next != nil {
		sub.next.prev = sub.prev
	}
	sub.prev = nil
	sub.next = nil
	c.mux.Unlock()

	sub.done = true
	return nil
}

// Latency returns an estimate of the max roundtrip time between any publisher and subscriber.
func (c *Conn) Latency() time.Duration {
	natsConn := c.natsConn.Load()
	if natsConn != nil {
		if len(natsConn.Servers()) == 1 {
			// Single NATS server
			rtt, err := natsConn.RTT()
			if err != nil {
				return time.Millisecond * 250
			}
			// Add 50ms and round up to nearest 100ms
			roundTo := time.Duration(100)
			return (rtt*2 + (roundTo/2+roundTo-1)*time.Millisecond) / roundTo / time.Millisecond * roundTo * time.Millisecond
		}
	} else if c.shortCircuitEnabled.Load() {
		// Short circuit only
		return time.Millisecond * 10
	}
	// Default to 300ms
	return time.Millisecond * 300
}
