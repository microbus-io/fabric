/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package errors

import (
	stderrors "errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_RuntimeTrace(t *testing.T) {
	t.Parallel()

	file, function, line1, _ := RuntimeTrace(0)
	_, _, line2, _ := RuntimeTrace(0)
	assert.Contains(t, file, "errors_test.go")
	assert.Equal(t, "errors.TestErrors_RuntimeTrace", function)
	assert.Equal(t, line1+1, line2)
}

func TestErrors_New(t *testing.T) {
	t.Parallel()

	tracedErr := New("This is a new error!", "annotation1", "annotation2")
	assert.Error(t, tracedErr)
	assert.Equal(t, "This is a new error!", tracedErr.Error())
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Annotations, "annotation1")
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Annotations, "annotation1")
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_New")
}

func TestErrors_Newf(t *testing.T) {
	t.Parallel()

	tracedErr := Newf("Error %s", "Error")
	assert.Error(t, tracedErr)
	assert.Equal(t, "Error Error", tracedErr.Error())
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Newf")
}

func TestErrors_Trace(t *testing.T) {
	t.Parallel()

	err := stderrors.New("Standard Error")
	assert.Error(t, err)

	tracedErr := Trace(err, "annotation1")
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Trace")

	tracedErr = Trace(tracedErr)
	assert.Len(t, tracedErr.(*TracedError).stack, 2)
	assert.NotEmpty(t, tracedErr.(*TracedError).String())

	tracedErr = Trace(tracedErr, "annotation2", "annotation3")
	assert.Len(t, tracedErr.(*TracedError).stack, 3)
	assert.NotEmpty(t, tracedErr.(*TracedError).String())

	err = Trace(nil)
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func TestErrors_Convert(t *testing.T) {
	t.Parallel()

	err := stderrors.New("Other standard error")
	assert.Error(t, err)

	tracedErr := Convert(err)
	assert.Error(t, tracedErr)
	assert.Empty(t, tracedErr.stack)

	err = Trace(tracedErr, "annotate!")
	tracedErr = Convert(err)
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.stack, 1)

	tracedErr = Convert(nil)
	assert.Nil(t, tracedErr)
}

func TestErrors_JSON(t *testing.T) {
	t.Parallel()

	tracedErr := New("Error!")

	b, err := tracedErr.(*TracedError).MarshalJSON()
	assert.NoError(t, err)

	var unmarshal TracedError
	err = unmarshal.UnmarshalJSON(b)
	assert.NoError(t, err)

	assert.Equal(t, tracedErr.Error(), unmarshal.Error())
	assert.Equal(t, tracedErr.(*TracedError).String(), unmarshal.String())
}

func TestErrors_Format(t *testing.T) {
	t.Parallel()

	err := New("my error")

	s := fmt.Sprintf("%s", err)
	assert.Equal(t, "my error", s)

	v := fmt.Sprintf("%v", err)
	assert.Equal(t, "my error", v)

	vPlus := fmt.Sprintf("%+v", err)
	assert.Equal(t, err.(*TracedError).String(), vPlus)
	assert.Contains(t, vPlus, "errors.TestErrors_Format")
	assert.Contains(t, vPlus, "errors/errors_test.go:")

	vSharp := fmt.Sprintf("%#v", err)
	assert.Equal(t, err.(*TracedError).String(), vSharp)
	assert.Contains(t, vSharp, "errors.TestErrors_Format")
	assert.Contains(t, vSharp, "errors/errors_test.go:")
}

func TestErrors_Is(t *testing.T) {
	t.Parallel()

	err := Trace(os.ErrNotExist)
	assert.True(t, Is(err, os.ErrNotExist))
}
