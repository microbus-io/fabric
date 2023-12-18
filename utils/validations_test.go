/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_ValidateHostName(t *testing.T) {
	valid := []string{
		"hello",
		"hello.WORLD",
		"123.456",
		"1",
		"hello_world",
		"hello-world",
	}
	invalid := []string{
		"hello world",
		"hello..world",
		"hello.",
		".hello",
		"~hello",
		"$",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateHostName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateHostName(x), "%s", x)
	}
}

func TestUtils_ValidateConfigName(t *testing.T) {
	valid := []string{
		"hello",
		"WORLD",
		"hello123",
	}
	invalid := []string{
		"hello world",
		"hello-world",
		"hello_world",
		"hello.world",
		"1hello",
		"_hello",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateConfigName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateConfigName(x), "%s", x)
	}
}

func TestUtils_ValidateTickerName(t *testing.T) {
	valid := []string{
		"hello",
		"WORLD",
		"hello123",
	}
	invalid := []string{
		"hello world",
		"hello-world",
		"hello_world",
		"hello.world",
		"1hello",
		"_hello",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateTickerName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateTickerName(x), "%s", x)
	}
}

func TestUtils_ParseURLValid(t *testing.T) {
	valid := map[string]string{
		"https://example.com:123/path":         "https://example.com:123/path",
		"https://example.com/path":             "https://example.com:443/path",
		"https://example.com/":                 "https://example.com:443/",
		"https://example.com":                  "https://example.com:443",
		"https://example":                      "https://example:443",
		"//example.com/path":                   "https://example.com:443/path",
		"http://example.com/path":              "http://example.com:80/path",
		"https://example.com/path/sub?q=1&m=2": "https://example.com:443/path/sub?q=1&m=2",
	}

	for k, v := range valid {
		u, err := ParseURL(k)
		assert.NoError(t, err, "%s", k)
		assert.Equal(t, v, u.String())
	}
}

func TestUtils_ParseURLInvalid(t *testing.T) {
	invalid := []string{
		"https://example.com:99999/path",
		"https://$.com:123/path",
		"https://example..com/path",
		"https://example.com:123:456/path",
		"example.com/path",
		"/example.com/path",
		"/path",
		"",
	}
	for _, x := range invalid {
		u, err := ParseURL(x)
		assert.Error(t, err, "%s", x)
		assert.Nil(t, u)
	}
}
