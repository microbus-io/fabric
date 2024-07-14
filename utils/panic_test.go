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

package utils

import (
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/testarossa"
)

func TestUtils_CatchPanic(t *testing.T) {
	t.Parallel()

	// String
	err := CatchPanic(func() error {
		panic("message")
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "message", err.Error())

	// Error
	err = CatchPanic(func() error {
		panic(errors.New("panic"))
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "panic", err.Error())

	// Number
	err = CatchPanic(func() error {
		panic(5)
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "5", err.Error())

	// Division by zero
	err = CatchPanic(func() error {
		j := 1
		j--
		i := 5 / j
		i++
		return nil
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "runtime error: integer divide by zero", err.Error())

	// Nil map
	err = CatchPanic(func() error {
		x := map[int]int{}
		if true {
			x = nil
		}
		x[5] = 6
		return nil
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "assignment to entry in nil map", err.Error())

	// Standard error
	err = CatchPanic(func() error {
		return errors.New("standard")
	})
	testarossa.Error(t, err)
	testarossa.Equal(t, "standard", err.Error())
}
