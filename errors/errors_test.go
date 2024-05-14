/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
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

	tracedErr := Newf("Error %s", "Error")
	assert.Error(t, tracedErr)
	assert.Equal(t, "Error Error", tracedErr.Error())
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Newf")
}

func TestErrors_Newc(t *testing.T) {
	t.Parallel()

	tracedErr := Newc(400, "User Error")
	assert.Error(t, tracedErr)
	assert.Equal(t, "User Error", tracedErr.Error())
	assert.Equal(t, 400, tracedErr.(*TracedError).StatusCode)
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Newc")
}

func TestErrors_Presets(t *testing.T) {
	t.Parallel()

	e := BadRequest()
	assert.Equal(t, statusText[http.StatusBadRequest], e.Error())
	assert.Equal(t, http.StatusBadRequest, e.(*TracedError).StatusCode)

	e = Unauthorized()
	assert.Equal(t, statusText[http.StatusUnauthorized], e.Error())
	assert.Equal(t, http.StatusUnauthorized, e.(*TracedError).StatusCode)

	e = Forbidden()
	assert.Equal(t, statusText[http.StatusForbidden], e.Error())
	assert.Equal(t, http.StatusForbidden, e.(*TracedError).StatusCode)

	e = NotFound()
	assert.Equal(t, statusText[http.StatusNotFound], e.Error())
	assert.Equal(t, http.StatusNotFound, e.(*TracedError).StatusCode)

	e = RequestTimeout()
	assert.Equal(t, statusText[http.StatusRequestTimeout], e.Error())
	assert.Equal(t, http.StatusRequestTimeout, e.(*TracedError).StatusCode)

	e = NotImplemented()
	assert.Equal(t, statusText[http.StatusNotImplemented], e.Error())
	assert.Equal(t, http.StatusNotImplemented, e.(*TracedError).StatusCode)
}

func TestErrors_Newcf(t *testing.T) {
	t.Parallel()

	tracedErr := Newcf(400, "User %s", "Error")
	assert.Error(t, tracedErr)
	assert.Equal(t, "User Error", tracedErr.Error())
	assert.Equal(t, 400, tracedErr.(*TracedError).StatusCode)
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Newcf")
}

func TestErrors_Trace(t *testing.T) {
	t.Parallel()

	err := stderrors.New("Standard Error")
	assert.Error(t, err)

	tracedErr := Trace(err)
	assert.Error(t, tracedErr)
	assert.Len(t, tracedErr.(*TracedError).stack, 1)
	assert.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_Trace")

	tracedErr = Trace(tracedErr)
	assert.Len(t, tracedErr.(*TracedError).stack, 2)
	assert.NotEmpty(t, tracedErr.(*TracedError).String())

	tracedErr = Trace(tracedErr)
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

	err = Trace(tracedErr)
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

	err := stderrors.New("Oops")
	traced := Trace(err)
	assert.Equal(t, err, Unwrap(traced))
}

func TestErrors_TraceFull(t *testing.T) {
	t.Parallel()

	err := stderrors.New("Oops")
	traced := Trace(err)
	tracedUp0 := TraceUp(err, 0)
	tracedUp1 := TraceUp(err, 1)
	tracedFull := TraceFull(err, 0)

	assert.Len(t, traced.(*TracedError).stack, 1)

	assert.Len(t, tracedUp0.(*TracedError).stack, 1)
	assert.Len(t, tracedUp1.(*TracedError).stack, 1)
	assert.NotEqual(t, tracedUp0.(*TracedError).stack[0].Function, tracedUp1.(*TracedError).stack[0].Function)

	assert.Greater(t, len(tracedFull.(*TracedError).stack), 1)
	assert.Equal(t, tracedUp0.(*TracedError).stack[0].Function, tracedFull.(*TracedError).stack[0].Function)
	assert.Equal(t, tracedUp1.(*TracedError).stack[0].Function, tracedFull.(*TracedError).stack[1].Function)
}
