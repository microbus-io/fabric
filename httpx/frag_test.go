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
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestHttpx_FragRequest(t *testing.T) {
	t.Parallel()

	// Using BodyReader
	request(t, 128*1024, 1024, true)
	request(t, 128*1024+16, 1024, true)
	request(t, 1024, 32*1024, true)

	// Using ByteReader
	request(t, 128*1024, 1024, false)
	request(t, 128*1024+16, 1024, false)
	request(t, 1024, 32*1024, false)
}

func request(t *testing.T, bodySize int64, fragmentSize int64, optimized bool) {
	body := []byte(rand.AlphaNum64(int(bodySize)))
	var bodyReader io.Reader
	if optimized {
		bodyReader = NewBodyReader(body)
	} else {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest("GET", "https://www.example.com", bodyReader)
	req.Header.Add("Foo", "Bar 1")
	req.Header.Add("Foo", "Bar 2")
	testarossa.NoError(t, err)

	// Fragment
	remaining := bodySize
	fragReqs := []*http.Request{}
	frag, err := NewFragRequest(req, fragmentSize)
	testarossa.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		testarossa.NoError(t, err)
		testarossa.NotNil(t, r)
		fragReqs = append(fragReqs, r)

		contentLen := r.Header.Get("Content-Length")
		if remaining > fragmentSize {
			testarossa.Equal(t, strconv.FormatInt(fragmentSize, 10), contentLen)
		} else {
			testarossa.Equal(t, strconv.FormatInt(remaining, 10), contentLen)
		}
		remaining -= fragmentSize
	}

	// Defragment
	defrag := NewDefragRequest()
	for _, r := range fragReqs {
		err := defrag.Add(r)
		testarossa.NoError(t, err)
	}
	intReq, err := defrag.Integrated()
	testarossa.NoError(t, err)
	testarossa.NotNil(t, intReq)

	intBody, err := io.ReadAll(intReq.Body)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, body, intBody)

	contentLen := intReq.Header.Get("Content-Length")
	testarossa.True(t, contentLen == strconv.Itoa(len(body)))

	if testarossa.SliceLen(t, intReq.Header["Foo"], 2) {
		testarossa.Equal(t, "Bar 1", intReq.Header["Foo"][0])
		testarossa.Equal(t, "Bar 2", intReq.Header["Foo"][1])
	}
}

func TestHttpx_FragResponse(t *testing.T) {
	t.Parallel()

	// Using BodyReader
	response(t, 128*1024, 1024, true)
	response(t, 128*1024+16, 1024, true)
	response(t, 1024, 32*1024, true)

	// Using ByteReader
	response(t, 128*1024, 1024, false)
	response(t, 128*1024+16, 1024, false)
	response(t, 1024, 32*1024, false)
}

func response(t *testing.T, bodySize int64, fragmentSize int64, optimized bool) {
	body := []byte(rand.AlphaNum64(int(bodySize)))

	var res *http.Response
	if optimized {
		rec := NewResponseRecorder()
		rec.Header().Add("Foo", "Bar 1")
		rec.Header().Add("Foo", "Bar 2")
		n, err := rec.Write(body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, len(body), n)
		res = rec.Result()
	} else {
		rec := httptest.NewRecorder()
		rec.Header().Add("Foo", "Bar 1")
		rec.Header().Add("Foo", "Bar 2")
		n, err := rec.Write(body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, len(body), n)
		res = rec.Result()
	}

	// Fragment
	remaining := bodySize
	fragRess := []*http.Response{}
	frag, err := NewFragResponse(res, fragmentSize)
	testarossa.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		testarossa.NoError(t, err)
		testarossa.NotNil(t, r)
		fragRess = append(fragRess, r)

		contentLen := r.Header.Get("Content-Length")
		if remaining > fragmentSize {
			testarossa.Equal(t, strconv.FormatInt(fragmentSize, 10), contentLen)
		} else {
			testarossa.Equal(t, strconv.FormatInt(remaining, 10), contentLen)
		}
		remaining -= fragmentSize
	}

	// Defragment
	defrag := NewDefragResponse()
	for _, r := range fragRess {
		err := defrag.Add(r)
		testarossa.NoError(t, err)
	}
	intRes, err := defrag.Integrated()
	testarossa.NoError(t, err)
	testarossa.NotNil(t, intRes)

	intBody, err := io.ReadAll(intRes.Body)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, body, intBody)

	contentLen := intRes.Header.Get("Content-Length")
	testarossa.True(t, contentLen == strconv.Itoa(len(body)))

	if testarossa.SliceLen(t, intRes.Header["Foo"], 2) {
		testarossa.Equal(t, "Bar 1", intRes.Header["Foo"][0])
		testarossa.Equal(t, "Bar 2", intRes.Header["Foo"][1])
	}
}

func TestHttpx_DefragRequestNoContentLen(t *testing.T) {
	t.Parallel()

	bodySize := 128*1024 + 16
	body := []byte(rand.AlphaNum64(int(bodySize)))
	req, err := http.NewRequest("GET", "https://www.example.com", bytes.NewReader(body))
	testarossa.NoError(t, err)

	// Fragment the request
	frag, err := NewFragRequest(req, 1024)
	testarossa.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		testarossa.NoError(t, err)
		testarossa.NotNil(t, r)
		testarossa.True(t, r.ContentLength > 0)
		testarossa.NotEqual(t, "", r.Header.Get("Content-Length"))
	}

	// Defrag should still work without knowing the content length
	defrag := NewDefragRequest()
	for i := 1; i <= frag.N(); i++ {
		r, _ := frag.Fragment(i)
		r.Header.Del("Content-Length")
		r.ContentLength = -1
		err := defrag.Add(r)
		testarossa.NoError(t, err)
	}
	intReq, err := defrag.Integrated()
	testarossa.NoError(t, err)
	testarossa.NotNil(t, intReq)
	testarossa.Equal(t, -1, int(intReq.ContentLength))
	testarossa.Equal(t, "", intReq.Header.Get("Content-Length"))
	intBody, err := io.ReadAll(intReq.Body)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, body, intBody)
}

func TestHttpx_DefragResponseNoContentLen(t *testing.T) {
	t.Parallel()

	bodySize := 128*1024 + 16
	body := []byte(rand.AlphaNum64(int(bodySize)))

	rec := httptest.NewRecorder()
	n, err := rec.Write(body)
	testarossa.NoError(t, err)
	testarossa.Equal(t, len(body), n)
	res := rec.Result()

	// Fragment the request
	frag, err := NewFragResponse(res, 1024)
	testarossa.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		testarossa.NoError(t, err)
		testarossa.NotNil(t, r)
		testarossa.True(t, r.ContentLength > 0)
		testarossa.NotEqual(t, "", r.Header.Get("Content-Length"))
	}

	// Defrag should still work without knowing the content length
	defrag := NewDefragResponse()
	for i := 1; i <= frag.N(); i++ {
		r, _ := frag.Fragment(i)
		r.Header.Del("Content-Length")
		r.ContentLength = -1
		err := defrag.Add(r)
		testarossa.NoError(t, err)
	}
	intRes, err := defrag.Integrated()
	testarossa.NoError(t, err)
	testarossa.NotNil(t, intRes)
	testarossa.Equal(t, -1, int(intRes.ContentLength))
	testarossa.Equal(t, "", intRes.Header.Get("Content-Length"))
	intBody, err := io.ReadAll(intRes.Body)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, body, intBody)
}
