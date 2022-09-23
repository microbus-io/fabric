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
	assert.Contains(t, tracedErr.(*tracedError).Stack()[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.(*tracedError).Stack()[0].Annotations, "annotation1")
	assert.Len(t, tracedErr.(*tracedError).Stack(), 1)
}

func TestErrors_Newf(t *testing.T) {
	tracedErr := Newf("Error %s", "Error")
	assert.Error(t, tracedErr)
	assert.Equal(t, "Error Error", tracedErr.Error())
	assert.Len(t, tracedErr.(*tracedError).Stack(), 1)
}

func TestErrors_TraceError(t *testing.T) {
	err := stderrors.New("Standard Error")
	assert.Error(t, err)

	tracedErr := Trace(err, "annotation1")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*tracedError).Stack(), 1)

	tracedErr = Trace(tracedErr)
	assert.Len(t, tracedErr.(*tracedError).Stack(), 2)
	assert.NotEmpty(t, tracedErr.(*tracedError).String())

	tracedErr = Trace(tracedErr, "annotation2", "annotation3")
	assert.Len(t, tracedErr.(*tracedError).Stack(), 3)
	assert.NotEmpty(t, tracedErr.(*tracedError).String())

	err = Trace(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func TestErrors_Convert(t *testing.T) {
	err := fmt.Errorf("Other standard error")
	assert.Error(t, err)

	tracedErr := Convert(err)
	assert.Error(t, tracedErr)
	assert.Empty(t, tracedErr.(*tracedError).Stack())

	tracedErr = Trace(tracedErr, "annotate!")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*tracedError).Stack(), 1)

	err = Convert(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func TestErrors_JSON(t *testing.T) {
	tracedErr := New("Error!")

	b, err := tracedErr.(*tracedError).MarshalJSON()
	assert.NoError(t, err)

	var unmarshal tracedError
	err = unmarshal.UnmarshalJSON(b)
	assert.NoError(t, err)

	assert.Equal(t, tracedErr.Error(), unmarshal.Error())
	assert.Equal(t, tracedErr.(*tracedError).String(), unmarshal.String())
}
