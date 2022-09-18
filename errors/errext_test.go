package errext

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapUnwrap(t *testing.T) {
	err := errors.New("Example")
	assert.Error(t, err)

	// Wrap 2x
	err = Wrap(err)
	assert.Error(t, err)

	err = Wrap(err)
	assert.Error(t, err)

	// Unwrap 2x
	err = errors.Unwrap(err)
	assert.Error(t, err)

	err = errors.Unwrap(err)
	assert.Error(t, err)

	assert.Equal(t, "Example", err.Error())

	// Unwrap one more time and get nil for error
	err = errors.Unwrap(err)
	assert.NoError(t, err)
}
