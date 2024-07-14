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

package sub

import (
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestSub_NewSub(t *testing.T) {
	t.Parallel()

	type testCase struct {
		spec         string
		expectedHost string
		expectedPort string
		expectedPath string
	}
	testCases := []testCase{
		{"", "www.example.com", "443", ""},
		{"/", "www.example.com", "443", "/"},
		{":555", "www.example.com", "555", ""},
		{":555/", "www.example.com", "555", "/"},
		{":0/", "www.example.com", "0", "/"},
		{":99999/", "www.example.com", "99999", "/"},
		{"/path/with/slash", "www.example.com", "443", "/path/with/slash"},
		{"path/with/no/slash", "www.example.com", "443", "/path/with/no/slash"},
		{"https://good.example.com", "good.example.com", "443", ""},
		{"https://good.example.com/", "good.example.com", "443", "/"},
		{"https://good.example.com:555", "good.example.com", "555", ""},
		{"https://good.example.com:555/", "good.example.com", "555", "/"},
		{"https://good.example.com:555/path", "good.example.com", "555", "/path"},
	}

	for _, tc := range testCases {
		s, err := NewSub("GET", "www.example.com", tc.spec, nil)
		testarossa.NoError(t, err)
		testarossa.Equal(t, tc.expectedHost, s.Host)
		testarossa.Equal(t, tc.expectedPort, s.Port)
		testarossa.Equal(t, tc.expectedPath, s.Path)
	}
}

func TestSub_InvalidPort(t *testing.T) {
	t.Parallel()

	badSpecs := []string{
		":x/path",
		":-5/path",
	}
	for _, s := range badSpecs {
		_, err := NewSub("GET", "www.example.com", s, nil)
		testarossa.Error(t, err)
	}
}

func TestSub_Method(t *testing.T) {
	t.Parallel()

	badSpecs := []string{
		"123",
		"A B",
		"ABC123",
		"!",
		"",
		"*",
	}
	for _, s := range badSpecs {
		_, err := NewSub(s, "www.example.com", "/", nil)
		testarossa.Error(t, err)
	}

	okSpecs := []string{
		"POST", "POST",
		"post", "POST",
		"ANyThinG", "ANYTHING",
		"ANY", "ANY",
	}
	for i := 0; i < len(okSpecs); i += 2 {
		sub, err := NewSub(okSpecs[i], "www.example.com", "/", nil)
		if testarossa.NoError(t, err) {
			testarossa.Equal(t, okSpecs[i+1], sub.Method)
		}
	}
}

func TestSub_Apply(t *testing.T) {
	t.Parallel()

	s, err := NewSub("GET", "www.example.com", "/path", nil)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "www.example.com", s.Queue)
	testarossa.Equal(t, "GET", s.Method)

	s.Apply(NoQueue())
	testarossa.Equal(t, "", s.Queue)
	s.Apply(Queue("foo"))
	testarossa.Equal(t, "foo", s.Queue)
	s.Apply(DefaultQueue())
	testarossa.Equal(t, "www.example.com", s.Queue)
	s.Apply(Pervasive())
	testarossa.Equal(t, "", s.Queue)
	s.Apply(LoadBalanced())
	testarossa.Equal(t, "www.example.com", s.Queue)

	err = s.Apply(Queue("$$$"))
	testarossa.Error(t, err)
}

func TestSub_Canonical(t *testing.T) {
	t.Parallel()

	s, err := NewSub("GET", "www.example.com", ":567/path", nil)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "www.example.com:567/path", s.Canonical())

	s, err = NewSub("GET", "www.example.com", "/path", nil)
	testarossa.NoError(t, err)
	testarossa.Equal(t, "www.example.com:443/path", s.Canonical()) // default port 443

	s, err = NewSub("GET", "www.example.com", "http://zzz.example.com/path", nil) // http
	testarossa.NoError(t, err)
	testarossa.Equal(t, "zzz.example.com:80/path", s.Canonical()) // default port 80 for http
}

func TestSub_PathArguments(t *testing.T) {
	t.Parallel()

	_, err := NewSub("GET", "www.example.com", ":567/path/{named}/{suffix+}", nil)
	testarossa.NoError(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{}/{+}", nil)
	testarossa.NoError(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{}", nil)
	testarossa.NoError(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{+}", nil)
	testarossa.NoError(t, err)

	_, err = NewSub("GET", "www.example.com", ":567/path/x{x}x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{x}x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/x{x}", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/x{+}", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/}/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/x}/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{x/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/}{/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{{}/x", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{%!@}", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{%!@+}", nil)
	testarossa.Error(t, err)
	_, err = NewSub("GET", "www.example.com", ":567/path/{+}/{}", nil)
	testarossa.Error(t, err)
}
