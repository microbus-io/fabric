package errors

import (
	stderrors "errors"
	"fmt"
)

type ErrorsExtended interface {
	error
}

type ErrorsExtendedImpl struct {
	error
}

// Wrap wraps an error to accumulate and capture the stack trace.
// This can be unwrapped with standard Go's errors.Unwrap method.
func Wrap(err error) ErrorsExtended {
	// TODO: Get caller function name and line number
	// for stack tracing
	return fmt.Errorf("%w", err)
}

// Unwrap is equivalent to the standard Go errors.Unwrap method.
func Unwrap(err error) ErrorsExtended {
	return stderrors.Unwrap(err)
}

// New is equivalent to the standard Go errors.New method.
func New(text string) ErrorsExtended {
	return stderrors.New(text)
}
