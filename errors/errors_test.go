/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	stderrors "errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestErrors_RuntimeTrace(t *testing.T) {
	t.Parallel()

	file, function, line1, _ := RuntimeTrace(0)
	_, _, line2, _ := RuntimeTrace(0)
	testarossa.Contains(t, file, "errors_test.go")
	testarossa.Equal(t, "errors.TestErrors_RuntimeTrace", function)
	testarossa.Equal(t, line1+1, line2)
}

func TestErrors_New(t *testing.T) {
	t.Parallel()

	tracedErr := New("This is a new error!")
	testarossa.Error(t, tracedErr)
	testarossa.Equal(t, "This is a new error!", tracedErr.Error())
	testarossa.SliceLen(t, tracedErr.(*TracedError).stack, 1)
	testarossa.Contains(t, tracedErr.(*TracedError).stack[0].Function, "TestErrors_New")
}

func TestErrors_Newf(t *testing.T) {
	t.Parallel()

	err := Newf("Error %s", "Error")
	testarossa.Error(t, err)
	testarossa.Equal(t, "Error Error", err.Error())
	testarossa.SliceLen(t, err.(*TracedError).stack, 1)
	testarossa.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newf")
}

func TestErrors_Newc(t *testing.T) {
	t.Parallel()

	err := Newc(400, "User Error")
	testarossa.Error(t, err)
	testarossa.Equal(t, "User Error", err.Error())
	testarossa.Equal(t, 400, err.(*TracedError).StatusCode)
	testarossa.Equal(t, 400, StatusCode(err))
	testarossa.SliceLen(t, err.(*TracedError).stack, 1)
	testarossa.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newc")
}

func TestErrors_Newcf(t *testing.T) {
	t.Parallel()

	err := Newcf(400, "User %s", "Error")
	testarossa.Error(t, err)
	testarossa.Equal(t, "User Error", err.Error())
	testarossa.Equal(t, 400, err.(*TracedError).StatusCode)
	testarossa.Equal(t, 400, StatusCode(err))
	testarossa.SliceLen(t, err.(*TracedError).stack, 1)
	testarossa.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Newcf")
}

func TestErrors_Trace(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Standard Error")
	testarossa.Error(t, stdErr)

	err := Trace(stdErr)
	testarossa.Error(t, err)
	testarossa.SliceLen(t, err.(*TracedError).stack, 1)
	testarossa.Contains(t, err.(*TracedError).stack[0].Function, "TestErrors_Trace")

	err = Trace(err)
	testarossa.SliceLen(t, err.(*TracedError).stack, 2)
	testarossa.NotEqual(t, "", err.(*TracedError).String())

	err = Trace(err)
	testarossa.SliceLen(t, err.(*TracedError).stack, 3)
	testarossa.NotEqual(t, "", err.(*TracedError).String())

	stdErr = Trace(nil)
	testarossa.NoError(t, stdErr)
	testarossa.Nil(t, stdErr)
}

func TestErrors_Convert(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Other standard error")
	testarossa.Error(t, stdErr)

	err := Convert(stdErr)
	testarossa.Error(t, err)
	testarossa.SliceLen(t, err.stack, 0)

	stdErr = Trace(err)
	err = Convert(stdErr)
	testarossa.Error(t, err)
	testarossa.SliceLen(t, err.stack, 1)

	err = Convert(nil)
	testarossa.Nil(t, err)
}

func TestErrors_JSON(t *testing.T) {
	t.Parallel()

	err := New("Error!")

	b, jsonErr := err.(*TracedError).MarshalJSON()
	testarossa.NoError(t, jsonErr)

	var unmarshal TracedError
	jsonErr = unmarshal.UnmarshalJSON(b)
	testarossa.NoError(t, jsonErr)

	testarossa.Equal(t, err.Error(), unmarshal.Error())
	testarossa.Equal(t, err.(*TracedError).String(), unmarshal.String())
}

func TestErrors_Format(t *testing.T) {
	t.Parallel()

	err := New("my error")

	s := fmt.Sprintf("%s", err)
	testarossa.Equal(t, "my error", s)

	v := fmt.Sprintf("%v", err)
	testarossa.Equal(t, "my error", v)

	vPlus := fmt.Sprintf("%+v", err)
	testarossa.Equal(t, err.(*TracedError).String(), vPlus)
	testarossa.Contains(t, vPlus, "errors.TestErrors_Format")
	testarossa.Contains(t, vPlus, "errors/errors_test.go:")

	vSharp := fmt.Sprintf("%#v", err)
	testarossa.Equal(t, err.(*TracedError).String(), vSharp)
	testarossa.Contains(t, vSharp, "errors.TestErrors_Format")
	testarossa.Contains(t, vSharp, "errors/errors_test.go:")
}

func TestErrors_Is(t *testing.T) {
	t.Parallel()

	err := Trace(os.ErrNotExist)
	testarossa.True(t, Is(err, os.ErrNotExist))
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
	testarossa.True(t, Is(j, e1))
	testarossa.True(t, Is(j, e2))
	testarossa.True(t, Is(j, e3))
	testarossa.True(t, Is(j, e4))
	testarossa.True(t, Is(j, e4a))
	testarossa.True(t, Is(j, e4b))
	jj, ok := j.(*TracedError)
	if testarossa.True(t, ok) {
		testarossa.SliceLen(t, jj.stack, 1)
		testarossa.Equal(t, 500, jj.StatusCode)
	}

	testarossa.Nil(t, Join(nil, nil))
	testarossa.Equal(t, e3, Join(e3, nil))
}

func TestErrors_String(t *testing.T) {
	t.Parallel()

	err := Newc(400, "Oops!")
	err = Trace(err)
	s := err.(*TracedError).String()
	testarossa.Contains(t, s, "Oops!")
	testarossa.Contains(t, s, "[400]")
	testarossa.Contains(t, s, "/fabric/errors/errors_test.go:")
	firstDash := strings.Index(s, "-")
	testarossa.True(t, firstDash > 0)
	secondDash := strings.Index(s[firstDash+1:], "-")
	testarossa.True(t, secondDash > 0)
}

func TestErrors_Unwrap(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Oops")
	err := Trace(stdErr)
	testarossa.Equal(t, stdErr, Unwrap(err))
}

func TestErrors_TraceFull(t *testing.T) {
	t.Parallel()

	stdErr := stderrors.New("Oops")
	err := Trace(stdErr)
	errUp0 := TraceUp(stdErr, 0)
	errUp1 := TraceUp(stdErr, 1)
	errFull := TraceFull(stdErr, 0)

	testarossa.SliceLen(t, err.(*TracedError).stack, 1)

	testarossa.SliceLen(t, errUp0.(*TracedError).stack, 1)
	testarossa.SliceLen(t, errUp1.(*TracedError).stack, 1)
	testarossa.NotEqual(t, errUp0.(*TracedError).stack[0].Function, errUp1.(*TracedError).stack[0].Function)

	testarossa.True(t, len(errFull.(*TracedError).stack) > 1)
	testarossa.Equal(t, errUp0.(*TracedError).stack[0].Function, errFull.(*TracedError).stack[0].Function)
	testarossa.Equal(t, errUp1.(*TracedError).stack[0].Function, errFull.(*TracedError).stack[1].Function)
}
