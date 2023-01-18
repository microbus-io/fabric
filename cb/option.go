/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

package cb

import (
	"time"
)

// Option is used to set options for a callback.
type Option func(cb *Callback) error

// TimeBudget sets a timeout for the callback.
// The timeout is applied as a context deadline.
// The default time budget is 1 minute.
func TimeBudget(timeout time.Duration) Option {
	if timeout < 0 {
		timeout = 0
	}
	return func(cb *Callback) error {
		cb.TimeBudget = timeout
		return nil
	}
}
