package connector

import (
	"context"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

func TestConnector_SetConfig(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Mock config service
	mockCfg := New("configurator.sys")
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return nil
	})

	err := mockCfg.Startup()
	assert.NoError(t, err)
	defer mockCfg.Shutdown()

	// Connector
	con := New("set.config.connector")
	con.SetPlane(plane)

	err = con.DefineConfig("s", cfg.DefaultValue("default"))
	assert.NoError(t, err)

	assert.Equal(t, "default", con.Config("s"))
	err = con.SetConfig("s", "string")
	assert.NoError(t, err)
	assert.Equal(t, "string", con.Config("s"))

	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, "default", con.Config("s")) // Reset after fetching from configurator

	err = con.SetConfig("s", "something")
	assert.NoError(t, err)
	assert.Equal(t, "something", con.Config("s"))

	err = con.ResetConfig("s")
	assert.NoError(t, err)
	assert.Equal(t, "default", con.Config("s"))
}

func TestConnector_FetchConfig(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Mock a config service
	mockCfg := New("configurator.sys")
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"values":{"foo":"baz","int":"$$$"}}`))
		return nil
	})

	err := mockCfg.Startup()
	assert.NoError(t, err)
	defer mockCfg.Shutdown()

	// Connector
	con := New("fetch.config.connector")
	con.SetPlane(plane)
	err = con.DefineConfig("foo", cfg.DefaultValue("bar"))
	assert.NoError(t, err)
	err = con.DefineConfig("int", cfg.Validation("int"), cfg.DefaultValue("5"))
	assert.NoError(t, err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(t, changed("FOO"))
		assert.False(t, changed("int"))
		callbackCalled = true
		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, "bar", con.Config("foo"))
	assert.Equal(t, "5", con.Config("int"))

	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, "baz", con.Config("foo"), "New value should be read from configurator")
	assert.Equal(t, "5", con.Config("int"), "Invalid value should not be accepted")
	assert.True(t, callbackCalled)
}
