/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"net/http"
	"strings"
)

// IsLocalhostAddress checks if the request's remote address is the local host.
func IsLocalhostAddress(r *http.Request) bool {
	addr := r.Header.Get("X-Forwarded-For")
	if addr == "" {
		addr = r.RemoteAddr
	}
	return addr == "localhost" || strings.HasPrefix(addr, "localhost:") ||
		addr == "127.0.0.1" || strings.HasPrefix(addr, "127.0.0.1:") ||
		addr == "[::1]" || strings.HasPrefix(addr, "[::1]:")
}
