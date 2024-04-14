/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

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
			err = errors.TraceFull(err, 1)
		}
	}()
	err = f()
	return
}
