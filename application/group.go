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

package application

import (
	"context"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/service"
)

// group is a collection of microservices that share the same lifecycle.
// Microservices in a group are started and shutdown in parallel.
type group []service.Service

// Startup starts up a group of microservices in parallel.
// The context deadline is used to limit the time allotted to the operation.
func (grp group) Startup(ctx context.Context) error {
	// Start the microservices in parallel
	startErrs := make(chan error, len(grp))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range grp {
		if s.IsStarted() {
			continue
		}
		wg.Add(1)
		go func(s service.Service, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay)
			err := errors.Newf("'%s' failed to start", s.Hostname())
			var tryAfter time.Duration
			for {
				select {
				case <-ctx.Done():
					// Failed to start in allotted time, return the last error
					startErrs <- err
					return
				case <-time.After(tryAfter):
					err = s.Startup()
					if err == nil {
						return
					}
					tryAfter = time.Second // Try again a second later
				}
			}
		}(s, delay)
		delay += time.Millisecond
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
func (grp group) Shutdown(ctx context.Context) error {
	// Shutdown the microservices in parallel
	shutdownErrs := make(chan error, len(grp))
	var wg sync.WaitGroup
	var delay time.Duration
	for _, s := range grp {
		if !s.IsStarted() {
			continue
		}
		wg.Add(1)
		go func(s service.Service, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay)
			shutdownErrs <- s.Shutdown()
		}(s, delay)
		delay += time.Millisecond
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
