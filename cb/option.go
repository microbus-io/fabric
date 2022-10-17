package cb

import (
	"time"
)

// Option is used to set options for a callback.
type Option func(cb *Callback) error

// Timeout sets a timeout for the callback.
// The timeout is applied as a context deadline.
// A zero or negative timeout is considered no timeout at all.
func Timeout(timeout time.Duration) Option {
	return func(cb *Callback) error {
		cb.Timeout = timeout
		return nil
	}
}
