package errors

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_RuntimeTrace(t *testing.T) {
	file, function, line1 := runtimeTrace(0)
	_, _, line2 := runtimeTrace(0)
	assert.Contains(t, file, "errors_test.go")
	assert.Equal(t, "errors.TestErrors_RuntimeTrace", function)
	assert.Equal(t, line1+1, line2)
}

func TestErrors_New(t *testing.T) {
	tracedErr := New("This is a new error!", "annotation1", "annotation2")
	assert.Error(t, tracedErr)
	assert.Equal(t, "This is a new error!", tracedErr.Error())
	assert.Contains(t, tracedErr.(*tracedErrorImpl).Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.(*tracedErrorImpl).Stack()[0].Annotations, "annotation1")
	assert.Len(t, tracedErr.(*tracedErrorImpl).Stack(), 1)
}

func TestErrors_TraceError(t *testing.T) {
	err := stderrors.New("Standard Error")
	assert.Error(t, err)

	tracedErr := Trace(err, "annotation1")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*tracedErrorImpl).Stack(), 1)

	tracedErr = Trace(tracedErr)
	assert.Len(t, tracedErr.(*tracedErrorImpl).Stack(), 2)
	assert.NotEmpty(t, tracedErr.String())

	tracedErr = Trace(tracedErr, "annotation2", "annotation3")
	assert.Len(t, tracedErr.(*tracedErrorImpl).Stack(), 3)
	assert.NotEmpty(t, tracedErr.String())

	err = Trace(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func Test_Convert(t *testing.T) {
	err := fmt.Errorf("Other standard error")
	assert.Error(t, err)

	tracedErr := Convert(err)
	assert.Error(t, tracedErr)
	assert.Empty(t, tracedErr.(*tracedErrorImpl).Stack())

	tracedErr = Trace(tracedErr, "annotate!")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*tracedErrorImpl).Stack(), 1)

	err = Convert(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}
