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
	}
	invalid := []string{
		"hello world",
		"hello_world",
		"hello-world",
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
