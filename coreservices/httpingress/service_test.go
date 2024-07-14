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

package httpingress

import (
	"net/url"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestHttpingress_ResolveInternalURL(t *testing.T) {
	portMappings := map[string]string{
		"8080:*": "*",
		"443:*":  "443",
		"80:*":   "443",
	}

	testCases := []string{
		"https://proxy:8080/service:555/path?arg=val",
		"https://service:555/path?arg=val",

		"https://proxy:8080/service:443/path",
		"https://service/path",

		"https://proxy:8080/service:80/path",
		"https://service:80/path",

		"https://proxy:8080/service/path",
		"https://service/path",

		"http://proxy:8080/service:555/path",
		"https://service:555/path",

		"https://proxy:443/service:555/path",
		"https://service/path",

		"https://proxy:443/service:443/path",
		"https://service/path",

		"https://proxy:443/service/path",
		"https://service/path",

		"https://proxy:80/service/path",
		"https://service/path",
	}
	for i := 0; i < len(testCases); i += 2 {
		x, err := url.Parse(testCases[i])
		testarossa.NoError(t, err)
		u, err := url.Parse(testCases[i+1])
		testarossa.NoError(t, err)
		ru, err := resolveInternalURL(x, portMappings)
		testarossa.NoError(t, err)
		testarossa.Equal(t, u, ru)
	}
}
