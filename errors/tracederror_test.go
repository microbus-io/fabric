package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_runtimeTrace(t *testing.T) {
	file, function, line := runtimeTrace(0)
	assert.Contains(t, file, "tracederror_test.go")
	assert.Equal(t, "errors.Test_runtimeTrace", function)
	assert.Equal(t, 10, line)
}
