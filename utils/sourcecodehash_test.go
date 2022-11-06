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
