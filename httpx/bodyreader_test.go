package httpx

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_BodyReader(t *testing.T) {
	bin := []byte("Lorem Ipsum")
	br := NewBodyReader(bin)
	bout, err := io.ReadAll(br)
	assert.NoError(t, err)
	assert.Equal(t, bin, bout)
	assert.Equal(t, bin, br.Bytes())
	err = br.Close()
	assert.NoError(t, err)
}
