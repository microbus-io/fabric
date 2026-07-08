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
	"syscall"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/trc"
)

/*
Service implements the http.egress.core microservice.

The HTTP egress microservice relays HTTP requests to the internet.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	// HINT: Add member variables here
	httpClient *http.Client
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	// A single shared client reused across all requests, backed by its own transport (cloned from the default
	// so connection pooling and timeouts are sane, and so the transport is ours to extend). Per-request time
	// bounds come from the caller's context deadline attached in MakeRequest, not a static Client.Timeout.
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// SSRF guard. In internet-reachable deployments, refuse to connect to non-public destinations (loopback,
	// link-local including the 169.254.169.254 cloud-metadata endpoint, private/unique-local ranges). The
	// check runs in the dialer's Control callback, which fires on the resolved IP of each connection attempt,
	// so it also defeats DNS rebinding (the IP checked is the exact IP dialed) and covers redirect targets.
	// LOCAL and TESTING allow all destinations for dev ergonomics.
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if svc.Deployment() == connector.PROD || svc.Deployment() == connector.LAB {
		dialer.Control = func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return errors.Trace(err)
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlockedIP(ip) {
				return errors.New("egress to non-public address is blocked", "address", address, http.StatusForbidden)
			}
			return nil
		}
	}
	transport.DialContext = dialer.DialContext

	svc.httpClient = &http.Client{Transport: transport}
	return nil
}

// isBlockedIP reports whether ip is a non-public destination the egress proxy must refuse to reach: loopback,
// link-local unicast (which includes the 169.254.169.254 cloud-metadata endpoint) and multicast, private and
// unique-local ranges, the unspecified address, and multicast. IPv4-mapped IPv6 addresses are normalized by the
// net.IP predicates, so e.g. ::ffff:127.0.0.1 is caught as loopback.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified()
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
MakeRequest proxies a request to a URL and returns the HTTP response, respecting the timeout set in the context.
The proxied request is expected to be posted in the body of the request in binary form (RFC7231).
*/
func (svc *Service) MakeRequest(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: MakeRequest
	ctx := r.Context()
	// The outbound call is bounded solely by the caller's deadline. Microbus always sets one, so its absence
	// means the request did not arrive through the framework; refuse rather than make an unbounded call.
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("request context has no deadline", http.StatusInternalServerError)
	}
	req, err := http.ReadRequest(bufio.NewReaderSize(r.Body, 64))
	if err != nil {
		return errors.Trace(err)
	}
	if req.URL.Port() == "" {
		switch req.URL.Scheme {
		case "https":
			req.URL.Host += ":443"
		case "http":
			req.URL.Host += ":80"
		}
	}
	req.RequestURI = "" // Avoid "http: Request.RequestURI can't be set in client requests"
	req.Header.Set("Accept-Encoding", "br;q=1.0,deflate;q=0.8,gzip;q=0.6")
	req = req.WithContext(ctx) // Attach the caller's context

	// OpenTelemetry: create a child span
	_, span := svc.StartSpan(ctx, req.URL.Hostname(),
		trc.Client(),
		trc.Request(req),
	)
	spanEnded := false
	defer func() {
		if !spanEnded {
			span.End()
		}
	}()

	resp, err := svc.httpClient.Do(req)
	if err == nil {
		// Decompress as required
		var decompressed bytes.Buffer
		isCompressed := true
		switch resp.Header.Get("Content-Encoding") {
		case "br":
			rdr := brotli.NewReader(resp.Body)
			_, err = io.Copy(&decompressed, rdr)
			if err != nil {
				err = errors.Trace(err)
			}
		case "deflate":
			rdr := flate.NewReader(resp.Body)
			_, err = io.Copy(&decompressed, rdr)
			if err != nil {
				err = errors.Trace(err)
			}
			rdr.Close()
		case "gzip":
			var rdr *gzip.Reader
			rdr, err = gzip.NewReader(resp.Body)
			if err != nil {
				err = errors.Trace(err)
			} else {
				_, err = io.Copy(&decompressed, rdr)
				if err != nil {
					err = errors.Trace(err)
				}
				rdr.Close()
			}
		default:
			isCompressed = false
		}
		if err == nil {
			if isCompressed {
				resp.Header.Del("Content-Encoding")
				resp.Header.Set("Content-Length", strconv.Itoa(decompressed.Len()))
			}
			for k, vv := range resp.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			if isCompressed {
				_, err = io.Copy(w, &decompressed)
			} else {
				_, err = io.Copy(w, resp.Body)
			}
		}
	}
	if err != nil {
		// OpenTelemetry: record the error
		span.SetError(err)
		svc.ForceTrace(ctx)
	} else {
		span.SetOK(http.StatusOK)
	}
	span.End()
	spanEnded = true
	return errors.Trace(err)
}
