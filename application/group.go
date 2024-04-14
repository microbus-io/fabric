/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package application

import (
	"context"
	"sync"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
)

// Group is a collection of microservices that share the same lifecycle.
// Microservices in a group are started and shutdown in parallel.
type Group []connector.Service

// Startup starts up a group of microservices in parallel.
// The context deadline is used to limit the time allotted to the operation.
func (grp Group) Startup(ctx context.Context) error {
	// Start the microservices in parallel
	startErrs := make(chan error, len(grp))
	var wg sync.WaitGroup
	var offsettingDelay time.Duration
	for _, s := range grp {
		if s.IsStarted() {
			continue
		}
		s := s
		wg.Add(1)
		offsettingDelay += 2 * time.Millisecond
		go func() {
			time.Sleep(offsettingDelay)
			defer wg.Done()
			err := errors.Newf("'%s' failed to start", s.HostName())
			delay := time.Millisecond
			for {
				select {
				case <-ctx.Done():
					// Failed to start in allotted time, return the last error
					startErrs <- err
					return
				case <-time.After(delay):
					err = s.Startup()
					if err == nil {
						return
					}
					delay = time.Second // Try again a second later
				}
			}
		}()
	}
	wg.Wait()
	close(startErrs)
	var lastErr error
	for e := range startErrs {
		if e != nil {
			lastErr = e
		}
	}
	return lastErr
}

// Shutdown shuts down a group of microservices in parallel.
// The context deadline is used to limit the time allotted to the operation.
func (grp Group) Shutdown(ctx context.Context) error {
	// Shutdown the microservices in parallel
	shutdownErrs := make(chan error, len(grp))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range grp {
		if !s.IsStarted() {
			continue
		}
		s := s
		wg.Add(1)
		delay += 2 * time.Millisecond
		go func() {
			time.Sleep(delay)
			shutdownErrs <- s.Shutdown()
			wg.Done()
		}()
	}
	wg.Wait()
	close(shutdownErrs)
	var lastErr error
	for e := range shutdownErrs {
		if e != nil {
			lastErr = e
		}
	}
	return lastErr
}
