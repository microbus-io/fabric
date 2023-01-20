/*
Copyright 2023 Microbus LLC and various contributors

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
	"sync"
	"testing"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestConfigurator_ManyMicroservices(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	configSvc := NewService().(*Service)
	configSvc.SetPlane(plane)
	services := []connector.Service{
		configSvc,
	}
	n := 16
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		con := connector.New("many.microservices.configurator")
		con.SetPlane(plane)
		con.DefineConfig("foo", cfg.DefaultValue("bar"))
		con.DefineConfig("moo")
		con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
			con.LogDebug(ctx, "Config changed", log.String("foo", con.Config("foo")))
			wg.Done()
			return nil
		})
		services = append(services, con)
	}

	app := application.New(services...)
	err := app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "", services[i].(*connector.Connector).Config("moo"))
	}

	// Load new values
	err = configSvc.loadYAML(`
many.microservices.configurator:
  foo: baz
  moo: cow
`)
	assert.NoError(t, err)

	wg.Add(n)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)
	wg.Wait()

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "baz", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "cow", services[i].(*connector.Connector).Config("moo"))
	}

	// Restore foo to use the default value
	err = configSvc.loadYAML(`
many.microservices.configurator:
  foo:
  moo: cow
`)
	assert.NoError(t, err)

	wg.Add(n)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)
	wg.Wait()

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "cow", services[i].(*connector.Connector).Config("moo"))
	}
}

func TestConfigurator_Callback(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	configSvc := NewService().(*Service)
	configSvc.SetPlane(plane)

	con := connector.New("callback.configurator")
	con.SetPlane(plane)
	con.DefineConfig("foo", cfg.DefaultValue("bar"))
	var wg sync.WaitGroup
	err := con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(t, changed("foo"))
		wg.Done()
		return nil
	})
	assert.NoError(t, err)

	err = configSvc.Startup()
	assert.NoError(t, err)
	defer configSvc.Shutdown()
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, "bar", con.Config("foo"))

	configSvc.loadYAML(`
callback.configurator:
  foo: baz
`)

	// Force a refresh
	wg.Add(1)
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)
	wg.Wait()

	assert.Equal(t, "baz", con.Config("foo"))
}

func TestConfigurator_PeerSync(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Start the first peer
	config1 := NewService().(*Service)
	config1.SetPlane(plane)
	config1.loadYAML(`
www.example.com:
  Foo: Bar
`)
	err := config1.Startup()
	assert.NoError(t, err)
	defer config1.Shutdown()

	val, ok := config1.repo.Value("www.example.com", "Foo")
	assert.True(t, ok)
	assert.Equal(t, "Bar", val)

	// Start the microservice
	con := connector.New("www.example.com")
	con.SetPlane(plane)
	con.DefineConfig("Foo")

	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, "Bar", con.Config("Foo"))

	// Start the second peer
	config2 := NewService().(*Service)
	config2.SetPlane(plane)
	config2.loadYAML(`
www.example.com:
  Foo: Baz
`)
	err = config2.Startup()
	assert.NoError(t, err)
	defer config2.Shutdown()

	val, ok = config2.repo.Value("www.example.com", "Foo")
	assert.True(t, ok)
	assert.Equal(t, "Baz", val)

	val, ok = config1.repo.Value("www.example.com", "Foo")
	assert.True(t, ok)
	assert.Equal(t, "Baz", val, "First peer should have been updated")

	assert.Equal(t, "Baz", con.Config("Foo"), "Microservice should have been updated")
}
