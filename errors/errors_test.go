package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapUnwrap(t *testing.T) {
	err := New("Example")
	assert.Error(t, err)

	// Wrap 2x
	err = Wrap(err)
	assert.Error(t, err)

	err = Wrap(err)
	assert.Error(t, err)

	// Unwrap 2x
	err = Unwrap(err)
	assert.Error(t, err)

	err = Unwrap(err)
	assert.Error(t, err)

	assert.Equal(t, "Example", err.Error())

	// Unwrap one more time and get nil for error
	err = Unwrap(err)
	assert.NoError(t, err)
}
