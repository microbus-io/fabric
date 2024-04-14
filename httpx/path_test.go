/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
	assert.Equal(t, "https://example.org:123/path", JoinHostAndPath("example.com", "//example.org:123/path"))
}
