package configurator

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/connector"
	"github.com/stretchr/testify/assert"
)

func TestConfigurator_ManyMicroservices(t *testing.T) {
	t.Parallel()

	configSvc := NewService()
	services := []connector.Service{
		configSvc,
	}
	for i := 0; i < 16; i++ {
		con := connector.New("many.microservices.configurator")
		con.DefineConfig("foo", cfg.DefaultValue("bar"))
		con.DefineConfig("moo")
		services = append(services, con)
	}

	app := application.NewTesting(services...)
	err := app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "", services[i].(*connector.Connector).Config("moo"))
	}

	// Inject a new values
	err = configSvc.mockConfigYAML(`
many.microservices.configurator:
  foo: baz
  moo: cow
`)
	assert.NoError(t, err)

	err = configSvc.fetchValues(configSvc.Lifetime())
	assert.NoError(t, err)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(time.Second)

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "baz", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "cow", services[i].(*connector.Connector).Config("moo"))
	}

	// Restore foo to use the default value
	err = configSvc.mockConfigYAML(`
many.microservices.configurator:
  moo: cow
`)
	assert.NoError(t, err)

	err = configSvc.fetchValues(configSvc.Lifetime())
	assert.NoError(t, err)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(time.Second)

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "cow", services[i].(*connector.Connector).Config("moo"))
	}
}

func TestConfigurator_Import(t *testing.T) {
	t.Parallel()

	// Mock the config service
	configSvc := NewService()
	configSvc.mockConfigYAML(`
import.configurator:
  foo1: baz1
`)
	configSvc.mockConfigImportYAML(`
- from: https://www.example.com/allowed/config.yaml
  import: configurator
- from: https://www.example.com/disallowed/config.yaml
  import: xxx
`)
	configSvc.mockRemoteConfigYAML("https://www.example.com/allowed/config.yaml", `
import.configurator:
  foo2: baz2
`)
	configSvc.mockRemoteConfigYAML("https://www.example.com/disallowed/config.yaml", `
import.configurator:
  foo3: baz3
`)

	con := connector.New("import.configurator")
	con.DefineConfig("foo1", cfg.DefaultValue("bar1"))
	con.DefineConfig("foo2", cfg.DefaultValue("bar2"))
	con.DefineConfig("foo3", cfg.DefaultValue("bar3"))

	app := application.NewTesting(configSvc, con)
	err := app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	assert.Equal(t, "baz1", con.Config("foo1"))
	assert.Equal(t, "baz2", con.Config("foo2"))
	assert.Equal(t, "bar3", con.Config("foo3"))
}

func TestConfigurator_Ticker(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMock()
	configSvc := NewService()

	con := connector.New("ticker.configurator")
	con.DefineConfig("foo", cfg.DefaultValue("bar"))
	callbackCalled := false
	err := con.SetOnConfigChanged(func(ctx context.Context, changed map[string]bool) error {
		assert.True(t, changed["foo"])
		callbackCalled = true
		return nil
	})
	assert.NoError(t, err)

	app := application.NewTesting(configSvc, con)
	err = app.SetClock(mockClock)
	assert.NoError(t, err)
	err = app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	assert.Equal(t, "bar", con.Config("foo"))

	configSvc.mockConfigYAML(`
ticker.configurator:
  foo: baz
`)

	// Should trigger a refresh
	mockClock.Add(21 * time.Minute)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(time.Second)

	assert.Equal(t, "baz", con.Config("foo"))
	assert.True(t, callbackCalled)
}
