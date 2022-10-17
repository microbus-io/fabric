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
