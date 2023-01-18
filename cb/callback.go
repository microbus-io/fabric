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
