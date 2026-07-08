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

package trc

import (
	"net/http"
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestTrc_RequestAttributesExcludeHeadersAndQuery(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	r, err := http.NewRequest("GET", "https://www.example.com:8443/path/to/it?city=SF&secretKey=hunter2", nil)
	assert.NoError(err)
	r.Header.Set("Authorization", "Bearer eyJhbGciOi.secret.token")
	r.Header.Set("Cookie", "Authorization=eyJhbGciOi.secret.token")
	r.Header.Set("Content-Type", "application/json")

	attrs := attributesOfRequest(r)
	byKey := map[string]string{}
	for _, kv := range attrs {
		byKey[string(kv.Key)] = kv.Value.String()
	}

	// Structural attributes are recorded
	assert.Equal("GET", byKey["http.method"])
	assert.Equal("https", byKey["url.scheme"])
	assert.Equal("www.example.com", byKey["server.address"])
	assert.Equal("/path/to/it", byKey["url.path"])

	// Headers and query arguments must never be recorded, in any deployment:
	// they routinely carry credentials
	for key, value := range byKey {
		assert.False(strings.HasPrefix(key, "http.request.header."), "header attribute %s leaked", key)
		assert.NotEqual("url.query", key)
		assert.NotContains(value, "secret")
		assert.NotContains(value, "hunter2")
	}
}
