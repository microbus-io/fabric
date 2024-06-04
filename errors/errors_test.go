/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package errors

import (
	stderrors "errors"
	"fmt"
	"os"
	"strings"
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

	tracedErr := New("This is a new error!")
	assert.Error(t, tracedErr)
	assert.Equal(t, "This is a new error!", tracedErr.Error())
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_New")
}

func TestErrors_Newf(t *testing.T) {
	t.Parallel()

	err := Newf("Error %s", "Error")
	assert.Error(t, err)
	assert.Equal(t, "Error Error", err.Error())
	assert.Len(t, err.(*TracedError).stack, 1)
	assert.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newf")
}

func TestErrors_Newc(t *testing.T) {
	t.Parallel()

	err := Newc(400, "User Error")
	assert.Error(t, err)
	assert.Equal(t, "User Error", err.Error())
	assert.Equal(t, 400, err.(*TracedError).StatusCode)
	assert.Equal(t, 400, StatusCode(err))
	assert.Len(t, err.(*TracedError).stack, 1)
	assert.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newc")
}

func TestErrors_Newcf(t *testing.T) {
	t.Parallel()

	err := Newcf(400, "User %s", "Error")
	assert.Error(t, err)
	assert.Equal(t, "User Error", err.Error())
	assert.Equal(t, 400, err.(*TracedError).StatusCode)
	assert.Equal(t, 400, StatusCode(err))
	assert.Len(t, err.(*TracedError).stack, 1)
	assert.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newcf")
}

func TestErrors_Trace(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Standard Error")
	assert.Error(t, stdErr)

	err := Trace(stdErr)
	assert.Error(t, err)
	assert.Len(t, err.(*TracedError).stack, 1)
	assert.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Trace")

	err = Trace(err)
	assert.Len(t, err.(*TracedError).stack, 2)
	assert.NotEmpty(t, err.(*TracedError).String())

	err = Trace(err)
	assert.Len(t, err.(*TracedError).stack, 3)
	assert.NotEmpty(t, err.(*TracedError).String())

	stdErr = Trace(nil)
	assert.NoError(t, stdErr)
	assert.Nil(t, stdErr)
}

func TestErrors_Convert(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Other standard error")
	assert.Error(t, stdErr)

	err := Convert(stdErr)
	assert.Error(t, err)
	assert.Empty(t, err.stack)

	stdErr = Trace(err)
	err = Convert(stdErr)
	assert.Error(t, err)
	assert.Len(t, err.stack, 1)

	err = Convert(nil)
	assert.Nil(t, err)
}

func TestErrors_JSON(t *testing.T) {
	t.Parallel()

	err := New("Error!")

	b, jsonErr := err.(*TracedError).MarshalJSON()
	assert.NoError(t, jsonErr)

	var unmarshal TracedError
	jsonErr = unmarshal.UnmarshalJSON(b)
	assert.NoError(t, jsonErr)

	assert.Equal(t, err.Error(), unmarshal.Error())
	assert.Equal(t, err.(*TracedError).String(), unmarshal.String())
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

func TestErrors_Join(t *testing.T) {
	t.Parallel()

	e1 := stderrors.New("E1")
	e2 := Newc(400, "E2")
	e3 := New("E3")
	e3 = Trace(e3)
	e4a := stderrors.New("E4a")
	e4b := stderrors.New("E4b")
	e4 := Join(e4a, e4b)
	j := Join(e1, e2, nil, e3, e4)
	assert.True(t, Is(j, e1))
	assert.True(t, Is(j, e2))
	assert.True(t, Is(j, e3))
	assert.True(t, Is(j, e4))
	assert.True(t, Is(j, e4a))
	assert.True(t, Is(j, e4b))
	jj, ok := j.(*TracedError)
	if assert.True(t, ok) {
		assert.Len(t, jj.stack, 1)
		assert.Equal(t, 500, jj.StatusCode)
	}

	assert.Nil(t, Join(nil, nil))
	assert.Equal(t, e3, Join(e3, nil))
}

func TestErrors_String(t *testing.T) {
	t.Parallel()

	err := Newc(400, "Oops!")
	err = Trace(err)
	s := err.(*TracedError).String()
	assert.Contains(t, s, "Oops!")
	assert.Contains(t, s, "[400]")
	assert.Contains(t, s, "/fabric/errors/errors_test.go:")
	firstDash := strings.Index(s, "-")
	assert.Greater(t, firstDash, 0)
	secondDash := strings.Index(s[firstDash+1:], "-")
	assert.Greater(t, secondDash, 0)
}

func TestErrors_Unwrap(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Oops")
	err := Trace(stdErr)
	assert.Equal(t, stdErr, Unwrap(err))
}

func TestErrors_TraceFull(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Oops")
	err := Trace(stdErr)
	errUp0 := TraceUp(stdErr, 0)
	errUp1 := TraceUp(stdErr, 1)
	errFull := TraceFull(stdErr, 0)

	assert.Len(t, err.(*TracedError).stack, 1)

	assert.Len(t, errUp0.(*TracedError).stack, 1)
	assert.Len(t, errUp1.(*TracedError).stack, 1)
	assert.NotEqual(t, errUp0.(*TracedError).stack[0].Function, errUp1.(*TracedError).stack[0].Function)

	assert.Greater(t, len(errFull.(*TracedError).stack), 1)
	assert.Equal(t, errUp0.(*TracedError).stack[0].Function, errFull.(*TracedError).stack[0].Function)
	assert.Equal(t, errUp1.(*TracedError).stack[0].Function, errFull.(*TracedError).stack[1].Function)
}
