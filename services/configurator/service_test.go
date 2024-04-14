/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
	services := []connector.Service{}
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

	app := application.New(configSvc, services)
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

	for i := 0; i < len(services); i++ {
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

	for i := 0; i < len(services); i++ {
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
