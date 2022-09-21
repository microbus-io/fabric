package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_runtimeTrace(t *testing.T) {
	file, function, line := runtimeTrace(0)
	assert.Contains(t, file, "tracederror_test.go")
	assert.Equal(t, "errors.Test_runtimeTrace", function)
	assert.Equal(t, 12, line)
}

func Test_New(t *testing.T) {
	tracedErr := New("This is a new error!", "annotation1", "annotation2")
	assert.Error(t, tracedErr)
	assert.Equal(t, "This is a new error!", tracedErr.Error())
	assert.Contains(t, tracedErr.Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.Stack()[0].String(), "[annotation1 annotation2]")
	assert.Len(t, tracedErr.Stack(), 1)
}

func Test_TraceError(t *testing.T) {
	err := errors.New("Standard Error")
	assert.Error(t, err)

	tracedErr := TraceError(err, "annotation1") // Implicit convertion of Go error to TracedError
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.Stack(), 1)

	tracedErr = TraceError(tracedErr)
	assert.Len(t, tracedErr.Stack(), 2)
	assert.NotEmpty(t, tracedErr.String())

	err = TraceError(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func Test_Convert(t *testing.T) {
	err := fmt.Errorf("Other standard error")
	assert.Error(t, err)

	tracedErr := Convert(err) // Explicit convertion of Go error to TracedError
	assert.Error(t, tracedErr)
	assert.Empty(t, tracedErr.Stack())
	// Note: stack is empty as it was only converted to a TracedError
	// it did not add a trace to the stack

	// Trace is only added if it is invoked with TraceError
	tracedErr = TraceError(tracedErr, "annotate!")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.Stack(), 1)

	err = Convert(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func Test_Marshal_Unmarshal(t *testing.T) {
	t.Skip("TODO")
}
