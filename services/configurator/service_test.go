package configurator

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestConfigurator_ManyMicroservices(t *testing.T) {
	t.Parallel()

	configSvc := NewService().(*Service)
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

	// Load new values
	err = configSvc.loadYAML(`
many.microservices.configurator:
  foo: baz
  moo: cow
`)
	assert.NoError(t, err)

	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(200 * time.Millisecond)

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

	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(200 * time.Millisecond)

	for i := 1; i < len(services); i++ {
		assert.Equal(t, "bar", services[i].(*connector.Connector).Config("foo"))
		assert.Equal(t, "cow", services[i].(*connector.Connector).Config("moo"))
	}
}

func TestConfigurator_Callback(t *testing.T) {
	t.Parallel()

	configSvc := NewService().(*Service)

	con := connector.New("callback.configurator")
	con.DefineConfig("foo", cfg.DefaultValue("bar"))
	callbackCalled := false
	err := con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(t, changed("foo"))
		callbackCalled = true
		return nil
	})
	assert.NoError(t, err)

	app := application.NewTesting(configSvc, con)
	err = app.Startup()
	assert.NoError(t, err)
	defer app.Shutdown()

	assert.Equal(t, "bar", con.Config("foo"))

	configSvc.loadYAML(`
callback.configurator:
  foo: baz
`)

	// Force a refresh
	err = configSvc.Refresh(configSvc.Lifetime())
	assert.NoError(t, err)

	// Known responders optimization might cause some of the microservices to be missed
	// and not return synchronously, but they still get the request
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, "baz", con.Config("foo"))
	assert.True(t, callbackCalled)
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
