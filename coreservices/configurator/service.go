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

package configurator

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/coreservices/configurator/configuratorapi"
	"github.com/microbus-io/fabric/coreservices/configurator/intermediate"
)

var (
	_ errors.TracedError
	_ http.Request
)

/*
Service implements the configurator.core microservice.

The Configurator is a core microservice that centralizes the dissemination of configuration values to other microservices.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	repo          *repository
	repoTimestamp time.Time
	lock          sync.RWMutex
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	if svc.repo == nil {
		svc.repo = &repository{}
	}

	// Load values from config.yaml if present in current working directory
	exists := func(fileName string) bool {
		_, err := os.Stat(fileName)
		return err == nil
	}
	if exists("config.yaml") {
		y, err := os.ReadFile("config.yaml")
		if err != nil {
			return errors.Trace(err)
		}
		svc.lock.Lock()
		err = svc.repo.LoadYAML(y)
		svc.repoTimestamp = time.Now()
		svc.lock.Unlock()
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		svc.LogWarn(ctx, "config.yaml not found in CWD")
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
	err = svc.PeriodicRefresh(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
Sync is used to synchronize values among replica peers of the configurator.
*/
func (svc *Service) Sync(ctx context.Context, timestamp time.Time, values map[string]map[string]string) (err error) {
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
PeriodicRefresh tells all microservices to contact the configurator and refresh their configs.
An error is returned if any of the values sent to the microservices fails validation.
*/
func (svc *Service) PeriodicRefresh(ctx context.Context) (err error) {
	var lastErr error
	ch := svc.Publish(ctx, pub.GET("https://all:888/config-refresh"))
	for i := range ch {
		_, err := i.Get()
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
	ch := configuratorapi.NewMulticastClient(svc).Sync(ctx, timestamp, values)
	for range ch {
		// Ignore results
	}
	return nil
}

// loadYAML loads a config.yaml into the repo. For testing purposes only.
func (svc *Service) loadYAML(configYAML string) error {
	if svc.Deployment() == connector.PROD {
		return errors.Newf("disallowed in %s deployment", connector.PROD)
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
