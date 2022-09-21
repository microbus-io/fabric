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

func Test_New(t *testing.T) {
	tracedErr := New("This is a new error!", "annotation1", "annotation2")
	assert.Equal(t, "This is a new error!", tracedErr.Error())
	assert.Contains(t, tracedErr.Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.Stack()[0].String(), "[annotation1 annotation2]")
}
