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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/configurator/configuratorapi"
)

var (
	_ context.Context
	_ *testing.T
	_ application.Application
	_ connector.Connector
	_ pub.Option
	_ testarossa.Asserter
	_ configuratorapi.Client
)

// MARKER: Values

// MARKER: Refresh

// MARKER: SyncRepo

// TestConfigurator_NoConfigsOfItsOwn guards the invariant that the configurator declares no config
// property of its own. A microservice with a config fetches it from configurator.core during startup,
// before its own subscriptions answer requests; if the configurator did that it would fetch from
// itself and deadlock. The service under test is run under a renamed hostname alongside a stand-in
// configurator.core, so a config-of-its-own would surface as a fetch against the stand-in.
func TestConfigurator_NoConfigsOfItsOwn(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Stand-in configurator.core that records whether anyone fetched config from it. A Mock cannot be
	// used here: mocks are disallowed outside TESTING, and TESTING disables the config fetch entirely.
	var fetched atomic.Bool
	stand := connector.New("configurator.core")
	stand.SetDeployment(connector.LAB) // Configs are disabled in TESTING
	stand.SetPlane(plane)
	stand.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			fetched.Store(true)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)
	err := stand.Startup(ctx)
	assert.NoError(err)
	defer stand.Shutdown(ctx)

	// The configurator under test, renamed so that a config of its own would fetch from the stand-in
	// (in production, unrenamed, it would fetch from itself and deadlock).
	svc := NewService()
	assert.NoError(svc.SetHostname("renamed.configurator.core"))
	svc.SetDeployment(connector.LAB)
	svc.SetPlane(plane)

	err = svc.Startup(ctx)
	assert.NoError(err)
	defer svc.Shutdown(ctx)

	assert.False(fetched.Load(),
		"the configurator fetched config from configurator.core during startup, which means it declares a config property of its own - adding one deadlocks its startup in production")
}

func TestConfigurator_ManyMicroservices(t *testing.T) {
	// No parallel - Setting envars
	ctx := t.Context()
	env.Push("MICROBUS_PLANE", utils.RandomIdentifier(12))
	defer env.Pop("MICROBUS_PLANE")
	env.Push("MICROBUS_DEPLOYMENT", connector.LAB)
	defer env.Pop("MICROBUS_DEPLOYMENT")

	assert := testarossa.For(t)

	configSvc := NewService()
	services := []service.Service{}
	n := 16
	var wg sync.WaitGroup
	for range n {
		con := connector.New("many.microservices.configurator")
		con.DefineConfig("foo", cfg.DefaultValue("bar"))
		con.DefineConfig("moo")
		con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
			con.LogDebug(ctx, "Config changed",
				"foo", con.Config("foo"),
			)
			wg.Done()
			return nil
		})
		services = append(services, con)
	}

	app := application.New()
	app.Add(configSvc)
	app.Add(services...)
	err := app.Startup(ctx)
	assert.NoError(err)
	defer app.Shutdown(ctx)

	for i := 1; i < len(services); i++ {
		assert.Equal("bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal("", services[i].(*connector.Connector).Config("moo"))
	}

	// Load new values
	err = configSvc.loadYAML(`
many.microservices.configurator:
  foo: baz
  moo: cow
`)
	assert.NoError(err)

	wg.Add(n)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(err)
	wg.Wait()

	for i := range services {
		assert.Equal("baz", services[i].(*connector.Connector).Config("foo"))
		assert.Equal("cow", services[i].(*connector.Connector).Config("moo"))
	}

	// Restore foo to use the default value
	err = configSvc.loadYAML(`
many.microservices.configurator:
  foo:
  moo: cow
`)
	assert.NoError(err)

	wg.Add(n)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(err)
	wg.Wait()

	for i := range services {
		assert.Equal("bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal("cow", services[i].(*connector.Connector).Config("moo"))
	}
}

func TestConfigurator_Callback(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	configSvc := NewService()
	configSvc.SetDeployment(connector.LAB)
	configSvc.SetPlane(plane)

	con := connector.New("callback.configurator")
	con.SetDeployment(connector.LAB)
	con.SetPlane(plane)
	con.DefineConfig("foo", cfg.DefaultValue("bar"))
	var wg sync.WaitGroup
	err := con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(changed("foo"))
		wg.Done()
		return nil
	})
	assert.NoError(err)

	err = configSvc.Startup(ctx)
	assert.NoError(err)
	defer configSvc.Shutdown(ctx)
	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal("bar", con.Config("foo"))

	configSvc.loadYAML(`
callback.configurator:
  foo: baz
`)

	// Force a refresh
	wg.Add(1)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(err)
	wg.Wait()

	assert.Equal("baz", con.Config("foo"))
}

func TestConfigurator_CoalesceRefresh(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	configSvc := NewService()

	// Each round signals it has started and blocks until the test sends its result on release.
	started := make(chan struct{})
	release := make(chan error)
	var rounds int32
	configSvc.refreshWork = func(ctx context.Context) error {
		atomic.AddInt32(&rounds, 1)
		started <- struct{}{}
		return <-release
	}

	// Caller A starts round 1 and becomes the runner.
	aErr := make(chan error, 1)
	go func() { aErr <- configSvc.Refresh(context.Background()) }()
	<-started // Round 1 is running and blocked

	// Callers B and C arrive while round 1 is in flight. Their change may postdate round 1, so
	// coalescing must not let them piggyback on it - a later round must run and carry their result.
	bErr := make(chan error, 1)
	cErr := make(chan error, 1)
	go func() { bErr <- configSvc.Refresh(context.Background()) }()
	go func() { cErr <- configSvc.Refresh(context.Background()) }()

	// Wait until at least one of B/C has registered as a waiter on a subsequent round, so releasing
	// round 1 is guaranteed to trigger round 2 rather than the callers starting from scratch.
	for {
		configSvc.refreshLock.Lock()
		registered := configSvc.refreshNext != nil
		configSvc.refreshLock.Unlock()
		if registered {
			break
		}
		time.Sleep(time.Millisecond)
	}

	// Every round after the first fails validation; feed that error to each subsequent round so
	// whichever round B and C land on, they observe the same failure.
	valErr := errors.New("validation failed")
	feederDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-started:
				release <- valErr
			case <-feederDone:
				return
			}
		}
	}()

	// Round 1 succeeds; A's own change propagated in it, so A must see nil.
	release <- nil

	assert.NoError(<-aErr)
	// B and C waited for a round that started after they arrived, and both see that round's error -
	// never a false nil, and identical across the two callers.
	assert.Equal(valErr, <-bErr)
	assert.Equal(valErr, <-cErr)
	// At least a second round ran: the concurrent requests were not coalesced away.
	assert.True(atomic.LoadInt32(&rounds) >= 2)
	close(feederDone)
}

func TestConfigurator_PeerSync(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Start the first peer
	config1 := NewService()
	config1.SetDeployment(connector.LAB)
	config1.SetPlane(plane)
	config1.loadYAML(`
www.example.com:
  Foo: Bar
`)
	err := config1.Startup(ctx)
	assert.NoError(err)
	defer config1.Shutdown(ctx)

	val, ok := config1.repo.Value("www.example.com", "Foo")
	assert.True(ok)
	assert.Equal("Bar", val)

	// Start the microservice
	con := connector.New("www.example.com")
	con.SetDeployment(connector.LAB)
	con.SetPlane(plane)
	con.DefineConfig("Foo")

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal("Bar", con.Config("Foo"))

	// Start the second peer
	config2 := NewService()
	config2.SetDeployment(connector.LAB)
	config2.SetPlane(plane)
	config2.loadYAML(`
www.example.com:
  Foo: Baz
`)
	err = config2.Startup(ctx)
	assert.NoError(err)
	defer config2.Shutdown(ctx)

	val, ok = config2.repo.Value("www.example.com", "Foo")
	assert.True(ok)
	assert.Equal("Baz", val)

	val, ok = config1.repo.Value("www.example.com", "Foo")
	assert.True(ok)
	assert.Equal("Baz", val, "First peer should have been updated")

	assert.Equal("Baz", con.Config("Foo"), "Microservice should have been updated")
}
