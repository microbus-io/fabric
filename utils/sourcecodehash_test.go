/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_SourceCodeHash(t *testing.T) {
	t.Parallel()

	h, err := SourceCodeSHA256(".")
	assert.NoError(t, err)
	b, err := hex.DecodeString(h)
	assert.NoError(t, err)
	assert.Len(t, b, 256/8)
}
