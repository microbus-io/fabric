/*
Copyright 2023 Microbus LLC and various contributors

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
	"strings"
)

// IsLocalhostAddress checks if the request's remote address is the local host.
func IsLocalhostAddress(r *http.Request) bool {
	addr := r.RemoteAddr
	return strings.HasPrefix(addr, "127.0.0.1:") ||
		strings.HasPrefix(addr, "[::1]:") ||
		strings.HasPrefix(addr, "localhost:")
}
