/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
