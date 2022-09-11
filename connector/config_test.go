package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	v, ok := Config[int]("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v)
}

func TestScanDirectory(t *testing.T) {
	fn, err := scanDirectory("../", true)
	_ = fn
	assert.NoError(t, err)
}
