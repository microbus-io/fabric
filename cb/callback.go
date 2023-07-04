/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package cb

import (
	"time"

	"github.com/microbus-io/fabric/errors"
)

// Callback holds settings for a user callback handler, such as the OnStartup and OnShutdown callbacks.
// Although technically public, it is used internally and should not be constructed by microservices directly.
type Callback struct {
	Name       string
	TimeBudget time.Duration
	Handler    any

	Interval time.Duration
	Ticker   *time.Ticker
}

// NewCallback creates a new callback.
func NewCallback(name string, handler any, options ...Option) (*Callback, error) {
	cb := &Callback{
		Name:       name,
		TimeBudget: time.Minute,
		Handler:    handler,
	}
	err := cb.Apply(options...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return cb, nil
}

// Apply the provided options to the callback.
func (cb *Callback) Apply(options ...Option) error {
	for _, opt := range options {
		err := opt(cb)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
