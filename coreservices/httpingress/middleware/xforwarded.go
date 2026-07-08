/*
Copyright (c) 2023-2025 Microbus LLC and various contributors

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
	"strings"

	"github.com/microbus-io/fabric/connector"
)

/*
XForwarded returns a middleware that sanitizes and sets the `X-Forwarded` headers of the request.
These headers are used by downstream microservices to identify the client and compose absolute URLs.

trustedProxyHops indicates how many trusted reverse proxies (CDN, load balancer) sit between the internet and
the ingress. When 0, the ingress faces the internet directly and all inbound `X-Forwarded` headers are ignored
and rewritten from the actual request, so clients cannot spoof their address or origin. When N or more, the last
N entries of `X-Forwarded-For` (proxies append, so trust counts from the right) and the proxy-authored
`X-Forwarded-Host`, `-Proto` and `-Prefix` (the outermost proxy writes first, so the first value wins) are
trusted, and any untrusted remainder is discarded. Multiple `X-Forwarded-Prefix` values, contributed by proxies
that each stripped a routing prefix, are combined into one, outermost first. In all cases exactly one sanitized
set of `X-Forwarded` headers is passed downstream, so microservices never apply trust logic themselves.
*/
func XForwarded(trustedProxyHops func() int) Middleware {
	return func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			host := r.Host
			proto := scheme
			forwardedFor := []string{r.RemoteAddr}
			prefix := ""
			hops := trustedProxyHops()
			if hops > 0 {
				ff := headerList(r.Header, "X-Forwarded-For")
				if len(ff) > hops {
					// Entries beyond the trusted hops are client input relayed by the proxies
					ff = ff[len(ff)-hops:]
				}
				if len(ff) > 0 {
					forwardedFor = ff
				}
				if hh := headerList(r.Header, "X-Forwarded-Host"); len(hh) > 0 {
					host = hh[0]
				}
				if pp := headerList(r.Header, "X-Forwarded-Proto"); len(pp) > 0 {
					proto = pp[0]
				}
				var sb strings.Builder
				for _, p := range headerList(r.Header, "X-Forwarded-Prefix") {
					p = "/" + strings.Trim(p, "/")
					if p != "/" {
						sb.WriteString(p)
					}
				}
				prefix = sb.String()
			}
			// Only the canonical headers authored here travel downstream
			var xfNames []string
			for k := range r.Header {
				if strings.HasPrefix(k, "X-Forwarded-") {
					xfNames = append(xfNames, k)
				}
			}
			for _, k := range xfNames {
				r.Header.Del(k)
			}
			r.Header.Set("X-Forwarded-Host", host)
			r.Header.Set("X-Forwarded-Proto", proto)
			r.Header.Set("X-Forwarded-For", strings.Join(forwardedFor, ", "))
			if prefix != "" {
				r.Header.Set("X-Forwarded-Prefix", prefix)
			}
			r.Header.Set("X-Forwarded-Path", r.URL.Path)
			return next(w, r) // No trace
		}
	}
}

// headerList collects the comma-separated items across all values of the named header, in order.
func headerList(h http.Header, name string) []string {
	var items []string
	for _, v := range h.Values(name) {
		for item := range strings.SplitSeq(v, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}
