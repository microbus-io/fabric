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

package httpx

import (
	"net/url"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestHttpx_JoinHostAndPath(t *testing.T) {
	testarossa.Equal(t, "https://example.com:443", JoinHostAndPath("example.com", ""))
	testarossa.Equal(t, "https://example.com:443/", JoinHostAndPath("example.com", "/"))
	testarossa.Equal(t, "https://example.com:443/path", JoinHostAndPath("example.com", "/path"))
	testarossa.Equal(t, "https://example.com:443/path", JoinHostAndPath("example.com", "path"))
	testarossa.Equal(t, "https://example.com:123", JoinHostAndPath("example.com", ":123"))
	testarossa.Equal(t, "https://example.com:123/path", JoinHostAndPath("example.com", ":123/path"))
	testarossa.Equal(t, "https://example.org:123/path", JoinHostAndPath("example.com", "https://example.org:123/path"))
	testarossa.Equal(t, "https://example.org:123/path", JoinHostAndPath("example.com", "//example.org:123/path"))
}

func TestHttpx_ParseURLValid(t *testing.T) {
	valid := map[string]string{
		"https://example.com:123/path":         "https://example.com:123/path",
		"https://example.com/path":             "https://example.com:443/path",
		"https://example.com/":                 "https://example.com:443/",
		"https://example.com":                  "https://example.com:443",
		"https://example":                      "https://example:443",
		"//example.com/path":                   "https://example.com:443/path",
		"http://example.com/path":              "http://example.com:80/path",
		"https://example.com/path/sub?q=1&m=2": "https://example.com:443/path/sub?q=1&m=2",
		"https://example.com:123/0":            "https://example.com:123/0",
	}

	for k, v := range valid {
		u, err := ParseURL(k)
		testarossa.NoError(t, err, "%s", k)
		testarossa.Equal(t, v, u.String())
	}
}

func TestHttpx_ParseURLInvalid(t *testing.T) {
	invalid := []string{
		"https://example.com:BAD/path",
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
		testarossa.Error(t, err, "%s", x)
		testarossa.Nil(t, u)
	}
}

func TestHttpx_FillPathArguments(t *testing.T) {
	testCases := []string{
		"https://example.com/article/{user}/comment/{comment}?user=123&comment=456", "https://example.com/article/123/comment/456",
		"https://example.com/article/{user}/comment/{comment}?user=123&comment=456&x=789", "https://example.com/article/123/comment/456?x=789",
		"https://example.com/article/{}/comment/{}?path1=123&path2=456&x=789", "https://example.com/article/123/comment/456?x=789",
		"https://example.com/fixed/{named}/{}/{suffix+}?named=1&path2=2&suffix=3/4&q=5", "https://example.com/fixed/1/2/3/4?q=5",
		"https://example.com/fixed/{named}/{}/{suffix+}", "https://example.com/fixed///",
		"https://example.com/fixed/{named}/{suffix+}?named=" + url.QueryEscape("[a&b/c]") + "&suffix=" + url.QueryEscape("[d&e/f]"), "https://example.com/fixed/" + url.PathEscape("[a&b/c]") + "/" + url.PathEscape("[d&e") + "/" + url.PathEscape("f]"),
	}
	for i := 0; i < len(testCases); i += 2 {
		resolved, err := FillPathArguments(testCases[i])
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, testCases[i+1], resolved)
		}
	}
}
