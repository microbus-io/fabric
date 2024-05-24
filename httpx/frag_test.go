/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	// Fragment
	remaining := bodySize
	fragReqs := []*http.Request{}
	frag, err := NewFragRequest(req, fragmentSize)
	assert.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		assert.NoError(t, err)
		assert.NotNil(t, r)
		fragReqs = append(fragReqs, r)

		contentLen := r.Header.Get("Content-Length")
		if remaining > fragmentSize {
			assert.Equal(t, strconv.FormatInt(fragmentSize, 10), contentLen)
		} else {
			assert.Equal(t, strconv.FormatInt(remaining, 10), contentLen)
		}
		remaining -= fragmentSize
	}

	// Defragment
	defrag := NewDefragRequest()
	for _, r := range fragReqs {
		err := defrag.Add(r)
		assert.NoError(t, err)
	}
	intReq, err := defrag.Integrated()
	assert.NoError(t, err)
	assert.NotNil(t, intReq)

	intBody, err := io.ReadAll(intReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, intBody)

	contentLen := intReq.Header.Get("Content-Length")
	assert.True(t, contentLen == strconv.Itoa(len(body)))

	if assert.Len(t, intReq.Header["Foo"], 2) {
		assert.Equal(t, "Bar 1", intReq.Header["Foo"][0])
		assert.Equal(t, "Bar 2", intReq.Header["Foo"][1])
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
		assert.NoError(t, err)
		assert.Equal(t, len(body), n)
		res = rec.Result()
	} else {
		rec := httptest.NewRecorder()
		rec.Header().Add("Foo", "Bar 1")
		rec.Header().Add("Foo", "Bar 2")
		n, err := rec.Write(body)
		assert.NoError(t, err)
		assert.Equal(t, len(body), n)
		res = rec.Result()
	}

	// Fragment
	remaining := bodySize
	fragRess := []*http.Response{}
	frag, err := NewFragResponse(res, fragmentSize)
	assert.NoError(t, err)
	for i := 1; i <= frag.N(); i++ {
		r, err := frag.Fragment(i)
		assert.NoError(t, err)
		assert.NotNil(t, r)
		fragRess = append(fragRess, r)

		contentLen := r.Header.Get("Content-Length")
		if remaining > fragmentSize {
			assert.Equal(t, strconv.FormatInt(fragmentSize, 10), contentLen)
		} else {
			assert.Equal(t, strconv.FormatInt(remaining, 10), contentLen)
		}
		remaining -= fragmentSize
	}

	// Defragment
	defrag := NewDefragResponse()
	for _, r := range fragRess {
		err := defrag.Add(r)
		assert.NoError(t, err)
	}
	intRes, err := defrag.Integrated()
	assert.NoError(t, err)
	assert.NotNil(t, intRes)

	intBody, err := io.ReadAll(intRes.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, intBody)

	contentLen := intRes.Header.Get("Content-Length")
	assert.True(t, contentLen == strconv.Itoa(len(body)))

	if assert.Len(t, intRes.Header["Foo"], 2) {
		assert.Equal(t, "Bar 1", intRes.Header["Foo"][0])
		assert.Equal(t, "Bar 2", intRes.Header["Foo"][1])
	}
}
