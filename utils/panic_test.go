package utils

import (
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/stretchr/testify/assert"
)

func TestUtils_CatchPanic(t *testing.T) {
	t.Parallel()

	// String
	err := CatchPanic(func() error {
		panic("message")
	})
	assert.Error(t, err)
	assert.Equal(t, "message", err.Error())

	// Error
	err = CatchPanic(func() error {
		panic(errors.New("panic"))
	})
	assert.Error(t, err)
	assert.Equal(t, "panic", err.Error())

	// Number
	err = CatchPanic(func() error {
		panic(5)
	})
	assert.Error(t, err)
	assert.Equal(t, "5", err.Error())

	// Division by zero
	err = CatchPanic(func() error {
		j := 1
		j--
		i := 5 / j
		i++
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "runtime error: integer divide by zero", err.Error())

	// Nil map
	err = CatchPanic(func() error {
		x := map[int]int{}
		if true {
			x = nil
		}
		x[5] = 6
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "assignment to entry in nil map", err.Error())

	// Standard error
	err = CatchPanic(func() error {
		return errors.New("standard")
	})
	assert.Error(t, err)
	assert.Equal(t, "standard", err.Error())
}
