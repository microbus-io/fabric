/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package configurator

import (
	"context"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/configurator/configuratorapi"
	"github.com/microbus-io/fabric/coreservices/control/controlapi"
	"github.com/microbus-io/fabric/frame"
)

var (
	_ errors.TracedError
	_ http.Request
)

// refreshRound tracks one execution of the config refresh fan-out.
type refreshRound struct {
	done chan struct{}
	err  error
}

/*
Service implements the configurator.core microservice.

The Configurator is a core microservice that centralizes the dissemination of configuration values to other microservices.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	repo           *repository
	repoTimestamp  time.Time
	lock           sync.RWMutex
	refreshLock    sync.Mutex
	refreshCurrent *refreshRound
	refreshNext    *refreshRound
	refreshWork    func(ctx context.Context) error // Defaults to PeriodicRefresh; overridable in tests
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	if svc.repo == nil {
		svc.repo = &repository{}
	}

	// Load values from config.yaml or config.local.yaml, in current working directory or ancestor directory
	exists := func(fileName string) bool {
		_, err := os.Stat(fileName)
		return err == nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return errors.Trace(err)
	}
	dir := ""
	split := strings.Split(wd, string(os.PathSeparator))
	for p := range split {
		dir = string(os.PathSeparator) + path.Join(split[:p+1]...)
		for _, fileName := range []string{
			path.Join(dir, "config.yaml"),
			path.Join(dir, "config.local.yaml"),
		} {
			if !exists(fileName) {
				continue
			}
			data, err := os.ReadFile(fileName)
			if err != nil {
				return errors.Trace(err)
			}
			svc.lock.Lock()
			err = svc.repo.LoadYAML(data)
			svc.repoTimestamp = time.Now()
			svc.lock.Unlock()
			if err != nil {
				svc.LogError(ctx, "Parsing config file",
					"error", err,
					"file", fileName,
				)
			} else {
				svc.LogDebug(ctx, "Read config file",
					"file", fileName,
				)
			}
		}
	}

	// Sync the current repo to peers before microservices pull the new config
	err = svc.publishSync(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	// Tell all microservices to refresh their config
	err = svc.Refresh(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Values returns the values associated with the specified config property names for the caller microservice.
*/
func (svc *Service) Values(ctx context.Context, names []string) (values map[string]string, err error) {
	host := frame.Of(ctx).FromHost()
	values = map[string]string{}
	svc.lock.RLock()
	for _, name := range names {
		val, ok := svc.repo.Value(host, name)
		if ok {
			values[name] = val
		}
	}
	svc.lock.RUnlock()
	return values, nil
}

/*
Refresh tells all microservices to contact the configurator and refresh their configs.
An error is returned if any of the values sent to the microservices fails validation.
*/
func (svc *Service) Refresh(ctx context.Context) (err error) {
	// Coalesce concurrent calls, but never silently drop one: a caller arriving while a refresh is
	// already running waits for a *subsequent* round guaranteed to start after this call, so its
	// change is propagated rather than piggybacking on a round that may predate it.
	svc.refreshLock.Lock()
	if svc.refreshCurrent != nil {
		if svc.refreshNext == nil {
			svc.refreshNext = &refreshRound{done: make(chan struct{})}
		}
		round := svc.refreshNext
		svc.refreshLock.Unlock()
		<-round.done
		return round.err
	}
	// No refresh is running; this caller becomes the runner of its own round.
	mine := &refreshRound{done: make(chan struct{})}
	svc.refreshCurrent = mine
	svc.refreshLock.Unlock()

	work := svc.refreshWork // For injection during test
	if work == nil {
		work = svc.PeriodicRefresh
	}

	round := mine
	for {
		roundErr := errors.CatchPanic(func() error {
			return work(ctx)
		})

		svc.refreshLock.Lock()
		round.err = roundErr
		close(round.done)
		next := svc.refreshNext
		svc.refreshNext = nil
		svc.refreshCurrent = next
		svc.refreshLock.Unlock()

		if next == nil {
			// Return this runner's own round result - the first round, which started after this call.
			return mine.err
		}
		// Callers arrived while this round ran; drive one more round so their change propagates.
		round = next
	}
}

/*
SyncRepo is used to synchronize values among replica peers of the configurator.
*/
func (svc *Service) SyncRepo(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error) {
	// Only respond to peers, and not to self
	if frame.Of(ctx).FromHost() != svc.Hostname() || frame.Of(ctx).FromID() == svc.ID() {
		return nil
	}

	// Compare incoming and current repos
	localRepo := &repository{
		values: values,
	}
	svc.lock.RLock()
	same := localRepo.Equals(svc.repo)
	newness := svc.repoTimestamp.Sub(timestamp)
	svc.lock.RUnlock()

	// If repos are the same, do nothing
	if same {
		return nil
	}

	// If incoming repo is newer, override the current one
	if newness <= 0 {
		svc.lock.Lock()
		svc.repo = localRepo
		svc.repoTimestamp = timestamp
		svc.lock.Unlock()
		return nil
	}

	// Sync the current repo to peers
	err = svc.publishSync(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
PeriodicRefresh tells all microservices to contact the configurator and refresh their configs. An error
is returned if any of the values sent to the microservices fails validation.
*/
func (svc *Service) PeriodicRefresh(ctx context.Context) (err error) {
	var lastErr error
	ch := controlapi.NewMulticastClient(svc).ForHost("all").ConfigRefresh(ctx)
	for i := range ch {
		err := i.Get()
		if err != nil && errors.StatusCode(err) != http.StatusNotFound {
			lastErr = errors.Trace(err)
			svc.LogError(ctx, "Updating config", "error", lastErr)
		}
	}
	return lastErr
}

// publishSync syncs the current repo with peers.
func (svc *Service) publishSync(ctx context.Context) error {
	svc.lock.RLock()
	timestamp := svc.repoTimestamp
	values := svc.repo.values
	svc.lock.RUnlock()

	// Broadcast to peers
	ch := configuratorapi.NewMulticastClient(svc).SyncRepo(ctx, timestamp, values)
	for range ch {
		// Ignore results
	}
	return nil
}

// loadYAML loads a config.yaml into the repo. For testing purposes only.
func (svc *Service) loadYAML(configYAML string) error {
	if svc.Deployment() == connector.PROD {
		return errors.New("disallowed in %s deployment", connector.PROD)
	}
	svc.lock.Lock()
	if svc.repo == nil {
		svc.repo = &repository{}
	}
	svc.repo.LoadYAML([]byte(configYAML))
	svc.repoTimestamp = time.Now()
	svc.lock.Unlock()
	return nil
}

/*
Deprecated.
*/
func (svc *Service) Values443(ctx context.Context, names []string) (values map[string]string, err error) {
	if frame.Of(ctx).XForwardedBaseURL() != "" {
		// Disallow external requests
		return nil, errors.New("", http.StatusNotFound)
	}
	svc.LogWarn(ctx, "Port 443 is deprecated")
	return svc.Values(ctx, names)
}

/*
Deprecated.
*/
func (svc *Service) Refresh443(ctx context.Context) (err error) {
	if frame.Of(ctx).XForwardedBaseURL() != "" {
		// Disallow external requests
		return errors.New("", http.StatusNotFound)
	}
	svc.LogWarn(ctx, "Port 443 is deprecated")
	return svc.Refresh(ctx)
}

/*
Deprecated.
*/
func (svc *Service) Sync443(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error) {
	if frame.Of(ctx).XForwardedBaseURL() != "" {
		// Disallow external requests
		return errors.New("", http.StatusNotFound)
	}
	svc.LogWarn(ctx, "Port 443 is deprecated")
	return svc.SyncRepo(ctx, timestamp, values)
}
