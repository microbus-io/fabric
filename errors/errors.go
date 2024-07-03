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
	"runtime"
	"strings"
)

var statusText = map[int]string{
	// 1xx
	100: "continue",
	// 2xx
	200: "ok",
	206: "partial content",
	// 3xx
	301: "moved permanently",
	302: "found",
	304: "not modified",
	307: "temporary redirect",
	308: "permanent redirect",
	// 4xx
	400: "bad request",
	401: "unauthorized",
	403: "forbidden",
	404: "not found",
	405: "method not allowed",
	408: "request timeout",
	413: "payload too large",
	// 5xx
	500: "internal server error",
	501: "not implemented",
	503: "service unavailable",
	508: "loop detected",
}

// As delegates to the standard Go's errors.As function.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// Is delegates to the standard Go's errors.Is function.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// Join aggregates multiple errors into one.
// The stack traces of the original errors are discarded and a new stack trace is captured.
func Join(errs ...error) error {
	var err error
	var n int
	for _, e := range errs {
		if e != nil {
			err = e
			n++
		}
	}
	if n == 0 {
		return nil
	}
	if n == 1 {
		return TraceUp(err, 1)
	}
	return TraceUp(stderrors.Join(errs...), 1)
}

// Unwrap delegates to the standard Go's errors.Wrap function.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// New creates a new error, capturing the current stack location.
func New(text string) error {
	return TraceUp(stderrors.New(text), 1)
}

// Newc creates a new error with an HTTP status code, capturing the current stack location.
func Newc(statusCode int, text string) error {
	if text == "" {
		text = statusText[statusCode]
	}
	err := TraceUp(stderrors.New(text), 1)
	err.(*TracedError).StatusCode = statusCode
	return err
}

// Newcf creates a new formatted error with an HTTP status code, capturing the current stack location.
func Newcf(statusCode int, format string, a ...any) error {
	if format == "" {
		format = statusText[statusCode]
	}
	err := TraceUp(fmt.Errorf(format, a...), 1)
	err.(*TracedError).StatusCode = statusCode
	return err
}

// Newf formats a new error, capturing the current stack location.
func Newf(format string, a ...any) error {
	return TraceUp(fmt.Errorf(format, a...), 1)
}

// Trace appends the current stack location to the error's stack trace.
func Trace(err error) error {
	return TraceUp(err, 1)
}

// Tracec appends the current stack location to the error's stack trace and sets the status code.
func Tracec(statusCode int, err error) error {
	err = TraceUp(err, 1)
	Convert(err).StatusCode = statusCode
	return err
}

// TraceUp appends the level above the current stack location to the error's stack trace.
// Level 0 captures the location of the caller.
func TraceUp(err error, level int) error {
	if err == nil {
		return nil
	}
	if level < 0 {
		level = 0
	}
	tracedErr := Convert(err)
	file, function, line, ok := RuntimeTrace(1 + level)
	if ok {
		tracedErr.stack = append(tracedErr.stack, &trace{
			File:     file,
			Function: function,
			Line:     line,
		})
	}
	return tracedErr
}

// TraceFull appends the full stack to the error's stack trace,
// starting at the indicated level.
// Level 0 captures the location of the caller.
func TraceFull(err error, level int) error {
	if err == nil {
		return nil
	}
	if level < 0 {
		level = 0
	}
	tracedErr := Convert(err)

	levels := level - 1
	for {
		levels++
		file, function, line, ok := RuntimeTrace(1 + levels)
		if !ok {
			break
		}
		if strings.HasPrefix(function, "runtime.") {
			continue
		}
		if function == "utils.CatchPanic" {
			break
		}
		tracedErr.stack = append(tracedErr.stack, &trace{
			File:     file,
			Function: function,
			Line:     line,
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
		error:      err,
		StatusCode: 500,
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

// StatusCode returns the HTTP status code associated with an error.
// It is the equivalent of Convert(err).StatusCode.
// If not specified, the default status code is 500.
func StatusCode(err error) int {
	if err == nil {
		return 0
	}
	return Convert(err).StatusCode
}
