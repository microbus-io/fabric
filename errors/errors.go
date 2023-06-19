/*
Copyright 2023 Microbus LLC and various contributors

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

// Unwrap delegates to the standard Go's errors.Wrap function.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// New creates a new error, capturing the current stack location.
// Optionally annotations may be attached
func New(text string, annotations ...any) error {
	return TraceUp(stderrors.New(text), 1, annotations...)
}

// Newc creates a new error with an HTTP status code, capturing the current stack location.
// Optionally annotations may be attached
func Newc(statusCode int, text string, annotations ...any) error {
	if text == "" {
		text = statusText[statusCode]
	}
	err := TraceUp(stderrors.New(text), 1, annotations...)
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
// Optional annotations may be attached
func Trace(err error, annotations ...any) error {
	return TraceUp(err, 1, annotations...)
}

// TraceUp appends the level above the current stack location to the error's stack trace.
// Level 0 captures the location of the caller.
// Optional annotations may be attached.
func TraceUp(err error, level int, annotations ...any) error {
	if err == nil {
		return nil
	}
	if level < 0 {
		level = 0
	}
	tracedErr := Convert(err)
	file, function, line, ok := RuntimeTrace(1 + level)
	if ok {
		var strAnnotations []string
		if len(annotations) > 0 {
			strAnnotations = make([]string, len(annotations))
			for i := range annotations {
				strAnnotations[i] = fmt.Sprintf("%v", annotations[i])
			}
		}
		tracedErr.stack = append(tracedErr.stack, &trace{
			File:        file,
			Function:    function,
			Line:        line,
			Annotations: strAnnotations,
		})
	}
	return tracedErr
}

// TraceFull appends the full stack to the error's stack trace,
// starting at the indicated level.
// Level 0 captures the location of the caller.
// Optional annotations may be attached to the first level captured.
func TraceFull(err error, level int, annotations ...any) error {
	if err == nil {
		return nil
	}
	if level < 0 {
		level = 0
	}
	tracedErr := Convert(err)

	var strAnnotations []string
	if len(annotations) > 0 {
		strAnnotations = make([]string, len(annotations))
		for i := range annotations {
			strAnnotations[i] = fmt.Sprintf("%v", annotations[i])
		}
	}

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
		tracedErr.stack = append(tracedErr.stack, &trace{
			File:        file,
			Function:    function,
			Line:        line,
			Annotations: strAnnotations,
		})
		strAnnotations = nil // Only add to first level
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
