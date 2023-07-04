/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpingress

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ResolveInternalURL(t *testing.T) {
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
		assert.NoError(t, err)
		u, err := url.Parse(testCases[i+1])
		assert.NoError(t, err)
		assert.Equal(t, u, resolveInternalURL(x, portMappings))
	}
}
