package errors

import (
	stderrors "errors"
	"fmt"
	"runtime"
	"strings"
)

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

// New creates a new error with tracing and annotation support.
func New(text string, annotations ...string) error {
	tracedErr := &tracedError{
		error: stderrors.New(text),
		stack: []trace{},
	}

	file, function, line := runtimeTrace(1)

	tracedErr.Push(trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
	return tracedErr
}

// Newf formats, according to format specifiers, a new error with
// tracing support.
func Newf(format string, a ...any) error {
	tracedErr := &tracedError{
		error: fmt.Errorf(format, a...),
		stack: []trace{},
	}

	file, function, line := runtimeTrace(1)

	tracedErr.Push(trace{
		File:     file,
		Function: function,
		Line:     line,
	})
	return tracedErr
}

// Trace adds a trace to an existing stack trace.
// If the error does not support tracing, it first establishes
// a new stack trace for the error before pushing the trace.
func Trace(err error, annotations ...string) error {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err).(*tracedError)

	file, function, line := runtimeTrace(1)

	tracedErr.Push(trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
	return tracedErr
}

// Convert converts an error to one that supports stack tracing.
// If the error already supports this, it is returned as it is.
// Note: Trace should be called to include the error's trace in the stack.
func Convert(err error) error {
	if err == nil {
		return nil
	}

	if tracedErr, ok := err.(*tracedError); ok {
		return tracedErr
	}

	return &tracedError{
		error: err,
		stack: []trace{},
	}
}

// runtimeTrace traces back by the amount of levels
// to retrieve the runtime information used for tracing.
func runtimeTrace(levels int) (file string, function string, line int) {
	pc, file, line, ok := runtime.Caller(levels + 1)
	function = "?"
	runtimeFunc := runtime.FuncForPC(pc)
	if ok && runtimeFunc != nil {
		function = runtimeFunc.Name()
		p := strings.LastIndex(function, "/")
		if p >= 0 {
			function = function[p+1:]
		}
	}
	return file, function, line
}
