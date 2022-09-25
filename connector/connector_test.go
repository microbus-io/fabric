package connector

import (
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/stretchr/testify/assert"
)

func TestConnector_HostAndID(t *testing.T) {
	c := NewConnector()
	assert.Empty(t, c.HostName())
	assert.NotEmpty(t, c.ID())
	c.SetHostName("example.com")
	assert.Equal(t, "example.com", c.HostName())
}

func TestConnector_BadHostName(t *testing.T) {
	c := NewConnector()
	badHosts := []string{
		"$.example.com",
		"my-example.com",
		"my_example.com",
		"example..com",
		"example.com.",
		".example.com",
		".",
		"",
	}
	for _, s := range badHosts {
		err := c.SetHostName(s)
		assert.Error(t, err)
	}
}

func TestConnector_CatchPanic(t *testing.T) {
	t.Parallel()

	// String
	err := catchPanic(func() error {
		panic("message")
	})
	assert.Error(t, err)
	assert.Equal(t, "message", err.Error())

	// Error
	err = catchPanic(func() error {
		panic(errors.New("panic"))
	})
	assert.Error(t, err)
	assert.Equal(t, "panic", err.Error())

	// Number
	err = catchPanic(func() error {
		panic(5)
	})
	assert.Error(t, err)
	assert.Equal(t, "5", err.Error())

	// Division by zero
	err = catchPanic(func() error {
		j := 1
		j--
		i := 5 / j
		i++
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "runtime error: integer divide by zero", err.Error())

	// Nil map
	err = catchPanic(func() error {
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
	err = catchPanic(func() error {
		return errors.New("standard")
	})
	assert.Error(t, err)
	assert.Equal(t, "standard", err.Error())
}
