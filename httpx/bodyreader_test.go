/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

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
	br.Reset()
	bout, err = io.ReadAll(br)
	assert.NoError(t, err)
	assert.Equal(t, bin, bout)
	assert.Equal(t, bin, br.Bytes())
	err = br.Close()
	assert.NoError(t, err)
}
