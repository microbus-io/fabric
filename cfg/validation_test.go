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

package cfg

import (
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestCfg_ValidateValue(t *testing.T) {
	t.Parallel()

	good := []string{
		"str", "lorem ipsum",
		"str [a-z]+", "m",

		"bool", "true",

		"int", "5",
		"int (0,10]", "10",
		"int [0,10)", "0",
		"int [0,)", "100",

		"float", "1.6",
		"float (0,10.1]", "10.1",
		"float [0,10)", "0",
		"float [0,10)", "9.99999",
		"float [0,)", "1e5",
		"float [,-2)", "-3",

		"dur", "1s",
		"dur [1s,1h]", "1m",
		"dur [1ns,1ms]", "1us",
		"dur (1s,1h]", "1h",
		"dur [1s,1h)", "1s",
		"dur [1s,)", "123h",

		"set x|y|z", "x",
		"set x||z", "",

		"url", "https://example.com/path",

		"email", "hello@example.com",
		"email", "<hello@example.com>",
		"email", "Hello <hello@example.com>",

		"json", `{"x":5,"y":10}`,
		"json", `[1,2,3,4]`,
		"json", `5`,
		"json", `"x"`,
	}

	bad := []string{
		"str [a-z]+", "9",

		"bool", "maybe",

		"int", "x",
		"int", "1.5",
		"int (0,10]", "0",
		"int [0,10)", "10",
		"int [0,)", "-5",

		"float", "x",
		"float (1.2,10]", "1.2",
		"float [0,9.8)", "9.8",
		"float [0,)", "-5",

		"dur", "6x",
		"dur (0s,10s]", "0s",
		"dur [0s,10s)", "10s",
		"dur [0s,)", "-5s",

		"set x|y|z", "q",
		"set x|y|z", "",

		"url", "",
		"url", "https:///path",
		"url", "https://",
		"url", "xyz",
		"url", "example.com",

		"email", "",
		"email", "hello",
		"email", "hello@",
		"email", "hello@x",
		"email", "hello <hello>",
		"email", "hello <hello@>",
		"email", "hello <hello@x>",

		"json", "",
		"json", "foo",
		"json", `{"x":5,"y":10`,
		"json", `[1,2,]`,
		"json", `[1,2,3`,
	}

	for i := 0; i < len(good); i += 2 {
		testarossa.True(t, Validate(good[i], good[i+1]), "%v %v", good[i], good[i+1])
	}
	for i := 0; i < len(bad); i += 2 {
		testarossa.False(t, Validate(bad[i], bad[i+1]), "%v %v", bad[i], bad[i+1])
	}
}

func TestCfg_CheckRule(t *testing.T) {
	t.Parallel()

	checks := map[string]bool{
		"str":        true,  // No regexp
		"str [0-9]+": true,  // Good regexp
		"str [*.":    false, // Bad regexp

		"bool":          true,  // No spec
		"bool anything": false, // Spec not allowed

		"int":          true,  // No range
		"int (0,99]":   true,  // Good range
		"int (,99]":    true,  // Good range
		"int (,]":      true,  // Good range
		"int 0,99]":    false, // Bad range
		"int [0,99.5]": false, // Bad range

		"float":          true,  // No range
		"float (0,99.9]": true,  // Good range
		"float (0,1e6]":  true,  // Good range
		"float (0,xyz]":  false, // Bad range

		"dur":          true,  // No range
		"dur (1ms,5h]": true,  // Good range
		"dur (1ns,5m]": true,  // Good range
		"dur (1us,5s]": true,  // Good range
		"dur (1x,5s]":  false, // Bad range

		"set x|y|z": true,  // Good set
		"set":       false, // No set

		"url":             true,  // No spec
		"url example.com": false, // Spec not allowed

		"email":          true,  // No spec
		"email anything": false, // Spec not allowed

		"json":          true,  // No spec
		"json anything": false, // Spec not allowed
	}
	for r, ok := range checks {
		testarossa.Equal(t, ok, checkRule(r), "%v", r)
	}
}

func TestCfg_NormalizedType(t *testing.T) {
	t.Parallel()

	checks := map[string]string{
		"":         "str",
		"str":      "str",
		"string":   "str",
		"text":     "str",
		"TeXT":     "str",
		"xyz":      "str",
		"bool":     "bool",
		"Boolean":  "bool",
		"int":      "int",
		"integer":  "int",
		"long":     "int",
		"float":    "float",
		"double":   "float",
		"decimal":  "float",
		"number":   "float",
		"dur":      "dur",
		"duration": "dur",
		"set":      "set",
		"URL":      "url",
		"eMail":    "email",
		"JSON":     "json",
	}
	for in, exp := range checks {
		norm, _ := normalizedType(in)
		testarossa.Equal(t, exp, norm)
	}
}
