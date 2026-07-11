package busstopapi

import (
	"strings"
	"testing"

	"github.com/microbus-io/dv8"
	"github.com/microbus-io/testarossa"
)

func TestBusStop_ValidateObject(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Prepare a valid object
	validObject := BusStop{
		// HINT: Initialize the object's fields with valid values
		Example: "Valid value",
	}
	err := dv8.Validate(ctx, &validObject)
	assert.NoError(err)

	// HINT: Check validation of individual object fields
	t.Run("example_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.Example = strings.Repeat("X", 1024) // Too long
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
}
