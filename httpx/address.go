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

package httpx

import (
	"net/http"
	"net/netip"
	"strings"
)

// IsLocalhostAddress checks if the request's remote address is the local host.
func IsLocalhostAddress(r *http.Request) bool {
	addr := r.Header.Get("X-Forwarded-For")
	if addr == "" {
		addr = r.RemoteAddr
	}
	if addr == "localhost" || strings.HasPrefix(addr, "localhost:") ||
		addr == "127.0.0.1" || strings.HasPrefix(addr, "127.0.0.1:") ||
		addr == "[::1]" || strings.HasPrefix(addr, "[::1]:") {
		return true
	}
	ap, err := netip.ParseAddrPort(addr)
	if err == nil && ap.IsValid() && ap.Addr().IsLoopback() {
		return true
	}
	return false
}

// IsPrivateIPAddress checks if the request's remote address is on the local subnets, e.g. 192.168.X.X or 10.X.X.X.
func IsPrivateIPAddress(r *http.Request) bool {
	addr := r.Header.Get("X-Forwarded-For")
	if addr == "" {
		addr = r.RemoteAddr
	}
	if strings.HasPrefix(addr, "192.168.") || strings.HasPrefix(addr, "10.") {
		return true
	}
	ap, err := netip.ParseAddrPort(addr)
	if err == nil && ap.IsValid() && ap.Addr().IsPrivate() {
		return true
	}
	return false
}
