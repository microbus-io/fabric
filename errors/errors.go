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

// New creates a new TracedError. It replaces Go's errors.New function,
// to enable stack tracing and annotation of an error.
func New(text string, annotations ...string) TracedError {
	tracedErr := &tracedErrorImpl{
		error: stderrors.New(text),
		stack: []trace{},
	}

	file, function, line := runtimeTrace(1)

	return tracedErr.Push(trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
}

// Newf creates a new TracedError. It replaces Go's fmt.Errorf function
// but enables stack tracing of the error.
func Newf(format string, a ...any) TracedError {
	tracedErr := &tracedErrorImpl{
		error: fmt.Errorf(format, a...),
		stack: []trace{},
	}

	file, function, line := runtimeTrace(1)

	return tracedErr.Push(trace{
		File:     file,
		Function: function,
		Line:     line,
	})
}

// Trace adds a trace to an existing stack trace of TracedError.
// If the error is not a TracedError, it is converted and starts a new stack trace
// and establishes the current error as the root of the stack trace.
func Trace(err error, annotations ...string) TracedError {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err).(*tracedErrorImpl)

	file, function, line := runtimeTrace(1)

	return tracedErr.Push(trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
}

// Convert converts a Go error into a TracedError without stack tracing.
// If the error is already a TracedError, it is returned as is.
func Convert(err error) TracedError {
	if err == nil {
		return nil
	}

	if tracedErr, ok := err.(*tracedErrorImpl); ok {
		return tracedErr
	}

	return &tracedErrorImpl{
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
