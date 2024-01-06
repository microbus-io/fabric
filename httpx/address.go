/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
