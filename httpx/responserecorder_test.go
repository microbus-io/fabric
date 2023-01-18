/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/frame"
	"github.com/stretchr/testify/assert"
)

func TestHttpx_ResponseRecorder(t *testing.T) {
	rr := NewResponseRecorder()

	// Write once
	rr.Header().Set("Foo", "Bar")
	rr.WriteHeader(http.StatusTeapot)

	bin := []byte("Lorem Ipsum")
	n, err := rr.Write(bin)
	assert.NoError(t, err)
	assert.Equal(t, len(bin), n)

	result := rr.Result()
	assert.Equal(t, bin, result.Body.(*BodyReader).Bytes())

	var buf bytes.Buffer
	err = result.Write(&buf)
	assert.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 418 I'm a teapot\r\nContent-Length: 11\r\nFoo: Bar\r\n\r\nLorem Ipsum", buf.String())

	// Write second time
	rr.Header().Set("Foo", "Baz")
	rr.WriteHeader(http.StatusConflict)

	bin2 := []byte(" Dolor Sit Amet")
	n, err = rr.Write(bin2)
	assert.NoError(t, err)
	assert.Equal(t, len(bin2), n)
	bin = append(bin, bin2...)

	result = rr.Result()
	assert.Equal(t, bin, result.Body.(*BodyReader).Bytes())

	buf.Reset()
	err = result.Write(&buf)
	assert.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 409 Conflict\r\nContent-Length: 26\r\nFoo: Baz\r\n\r\nLorem Ipsum Dolor Sit Amet", buf.String())
}

func TestHttpx_FrameOfResponseRecorder(t *testing.T) {
	utilsRecorder := NewResponseRecorder()
	utilsRecorder.Header().Set(frame.HeaderMsgId, "123")
	assert.Equal(t, "123", frame.Of(utilsRecorder).MessageID())
	httpResponse := utilsRecorder.Result()
	assert.Equal(t, "123", frame.Of(httpResponse).MessageID())
}
