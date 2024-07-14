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
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"
)

func TestHttpx_DeepObject(t *testing.T) {
	type Point struct {
		X int
		Y int
	}
	type Doc struct {
		I       int       `json:"i"`
		Zero    int       `json:"z,omitempty"`
		B       bool      `json:"b"`
		F       float32   `json:"f"`
		S       string    `json:"s"`
		Pt      Point     `json:"pt"`
		Empty   *Point    `json:"e,omitempty"`
		Null    *Point    `json:"n"`
		Special string    `json:"sp"`
		T       time.Time `json:"t"`
	}

	// Encode
	d1 := Doc{
		I:       5,
		B:       true,
		F:       5.67,
		S:       "Hello",
		Special: "Q&A",
		Pt:      Point{X: 3, Y: 4},
		T:       time.Date(2001, 10, 1, 12, 0, 0, 0, time.UTC),
	}
	values, err := EncodeDeepObject(d1)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "5", values.Get("i"))
		testarossa.Equal(t, "true", values.Get("b"))
		testarossa.Equal(t, "5.67", values.Get("f"))
		testarossa.Equal(t, "Hello", values.Get("s"))
		testarossa.Equal(t, "Q&A", values.Get("sp"))
		testarossa.Equal(t, "3", values.Get("pt[X]"))
		testarossa.Equal(t, "4", values.Get("pt[Y]"))
		testarossa.Equal(t, "2001-10-01T12:00:00Z", values.Get("t"))
	}

	var d2 Doc
	err = DecodeDeepObject(values, &d2)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, d1, d2)
	}
}

func TestHttpx_DeepObjectRequestPath(t *testing.T) {
	t.Parallel()

	var data struct {
		X struct {
			A int
			B int
			Y struct {
				C int
				D int
			}
		}
		S string
		B bool
		E string
	}
	r, err := http.NewRequest("GET", `/path?x.a=5&x[b]=3&x.y.c=1&x[y][d]=2&s=str&b=true&e=`, nil)
	testarossa.NoError(t, err)
	err = DecodeDeepObject(r.URL.Query(), &data)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 5, data.X.A)
	testarossa.Equal(t, 3, data.X.B)
	testarossa.Equal(t, 1, data.X.Y.C)
	testarossa.Equal(t, 2, data.X.Y.D)
	testarossa.Equal(t, "str", data.S)
	testarossa.Equal(t, true, data.B)
	testarossa.Equal(t, "", data.E)
}

func TestHttpx_DeepObjectDecodeOne(t *testing.T) {
	t.Parallel()

	data := struct {
		S string  `json:"s"`
		I int     `json:"i"`
		F float64 `json:"f"`
		B bool    `json:"b"`
	}{}

	// Into string
	err := decodeOne("s", "hello", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "hello", data.S)
	}
	err = decodeOne("s", `"hello"`, &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "hello", data.S)
	}
	err = decodeOne("s", "5", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, "5", data.S)
	}

	// Into int
	err = decodeOne("i", "5", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, 5, data.I)
	}

	// Into float64
	err = decodeOne("f", "5", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, 5.0, data.F)
	}
	err = decodeOne("f", "5.6", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, 5.6, data.F)
	}

	// Into bool
	err = decodeOne("b", "true", &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, true, data.B)
	}
	err = decodeOne("b", `"true"`, &data)
	if testarossa.NoError(t, err) {
		testarossa.Equal(t, true, data.B)
	}
}
