/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_JoinHostAndPath(t *testing.T) {
	assert.Equal(t, "https://example.com:443", JoinHostAndPath("example.com", ""))
	assert.Equal(t, "https://example.com:443/", JoinHostAndPath("example.com", "/"))
	assert.Equal(t, "https://example.com:443/path", JoinHostAndPath("example.com", "/path"))
	assert.Equal(t, "https://example.com:443/path", JoinHostAndPath("example.com", "path"))
	assert.Equal(t, "https://example.com:123", JoinHostAndPath("example.com", ":123"))
	assert.Equal(t, "https://example.com:123/path", JoinHostAndPath("example.com", ":123/path"))
	assert.Equal(t, "https://example.org:123/path", JoinHostAndPath("example.com", "https://example.org:123/path"))
}
