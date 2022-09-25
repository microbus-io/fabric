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
	tracedErr := &TracedError{
		error: stderrors.New(text),
	}
	file, function, line := runtimeTrace(1)
	tracedErr.stack = append(tracedErr.stack, trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
	return tracedErr
}

// Newf formats a new error, capturing the current stack location
func Newf(format string, a ...any) error {
	tracedErr := &TracedError{
		error: fmt.Errorf(format, a...),
	}
	file, function, line := runtimeTrace(1)
	tracedErr.stack = append(tracedErr.stack, trace{
		File:     file,
		Function: function,
		Line:     line,
	})
	return tracedErr
}

// Trace appends the current stack location to the error's stack trace.
// Optionally annotations may be attached
func Trace(err error, annotations ...string) error {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err).(*TracedError)
	file, function, line := runtimeTrace(1)
	tracedErr.stack = append(tracedErr.stack, trace{
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
	if tracedErr, ok := err.(*TracedError); ok {
		return tracedErr
	}
	return &TracedError{
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
