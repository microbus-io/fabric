/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"runtime"
	"strings"
)

// As delegates to the standard Go's errors.As function
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// Is delegates to the standard Go's errors.Is function
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// Unwrap delegates to the standard Go's errors.Wrap function
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// New creates a new error, capturing the current stack location.
// Optionally annotations may be attached
func New(text string, annotations ...string) error {
	return TraceUp(stderrors.New(text), 1, annotations...)
}

// Newc creates a new error with an HTTP status code, capturing the current stack location.
// Optionally annotations may be attached
func Newc(statusCode int, text string, annotations ...string) error {
	err := TraceUp(stderrors.New(text), 1, annotations...)
	err.(*TracedError).StatusCode = statusCode
	return err
}

// Newf formats a new error, capturing the current stack location
func Newf(format string, a ...any) error {
	return TraceUp(fmt.Errorf(format, a...), 1)
}

// Trace appends the current stack location to the error's stack trace.
// Optional annotations may be attached
func Trace(err error, annotations ...string) error {
	return TraceUp(err, 1, annotations...)
}

// TraceUp appends the levels above the current stack location to the error's stack trace.
// Optional annotations may be attached
func TraceUp(err error, levels int, annotations ...string) error {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err)
	file, function, line, ok := RuntimeTrace(1 + levels)
	if ok {
		tracedErr.stack = append(tracedErr.stack, &trace{
			File:        file,
			Function:    function,
			Line:        line,
			Annotations: annotations,
		})
	}
	return tracedErr
}

// Convert converts an error to one that supports stack tracing.
// If the error already supports this, it is returned as it is.
// Note: Trace should be called to include the error's trace in the stack.
func Convert(err error) *TracedError {
	if err == nil {
		return nil
	}
	if tracedErr, ok := err.(*TracedError); ok {
		return tracedErr
	}
	return &TracedError{
		error: err,
	}
}

// RuntimeTrace traces back by the amount of levels
// to retrieve the runtime information used for tracing.
func RuntimeTrace(levels int) (file string, function string, line int, ok bool) {
	pc, file, line, ok := runtime.Caller(levels + 1)
	if !ok {
		return "", "", 0, false
	}
	function = "?"
	runtimeFunc := runtime.FuncForPC(pc)
	if runtimeFunc != nil {
		function = runtimeFunc.Name()
		p := strings.LastIndex(function, "/")
		if p >= 0 {
			function = function[p+1:]
		}
	}
	return file, function, line, ok
}
