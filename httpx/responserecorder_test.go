package httpx

import (
	"bytes"
	"net/http"
	"testing"

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
