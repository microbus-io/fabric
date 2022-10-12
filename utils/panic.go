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
			err = errors.TraceUp(err, 2)
		}
	}()
	err = f()
	return
}
