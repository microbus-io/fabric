package utils

import (
	"fmt"

	"github.com/microbus-io/fabric/errors"
)

// CatchPanic calls the given function and returns any panic as a standard error
func CatchPanic(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}

			// Capture the stack trace
			annotations := []string{}
			for i := 4; ; i++ {
				_, function, line, ok := errors.RuntimeTrace(i)
				if !ok {
					break
				}
				annotations = append(annotations, fmt.Sprintf("%v:%d", function, line))
			}

			err = errors.TraceUp(err, 4, annotations...)
		}
	}()
	err = f()
	return
}
