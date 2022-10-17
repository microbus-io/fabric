package cb

import (
	"context"
	"time"
)

// CallbackHandler handles callbacks such as OnStartup and OnShutdown.
type CallbackHandler func(context.Context) error

// Callback holds settings for a user callback handler,
// such as the OnStartup and OnShutdown callbacks.
type Callback struct {
	Name    string
	Timeout time.Duration
	Handler CallbackHandler
}

// NewCallback creates a new callback.
func NewCallback(name string) *Callback {
	return &Callback{
		Name: name,
	}
}
