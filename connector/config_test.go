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

package connector

import (
	"context"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/coreservices/control/controlapi"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestConnector_SetConfig(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Mock config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return nil
	})

	err := mockCfg.Startup()
	testarossa.NoError(t, err)
	defer mockCfg.Shutdown()

	// Connector
	con := New("set.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)

	err = con.DefineConfig("s", cfg.DefaultValue("default"))
	testarossa.NoError(t, err)

	testarossa.Equal(t, "default", con.Config("s"))
	err = con.SetConfig("s", "string")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "string", con.Config("s"))

	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Equal(t, "default", con.Config("s")) // Reset after fetching from configurator

	err = con.SetConfig("s", "something")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "something", con.Config("s"))

	err = con.ResetConfig("s")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "default", con.Config("s"))
}

func TestConnector_FetchConfig(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"values":{"foo":"baz","int":"$$$"}}`))
		return nil
	})

	err := mockCfg.Startup()
	testarossa.NoError(t, err)
	defer mockCfg.Shutdown()

	// Connector
	con := New("fetch.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)
	err = con.DefineConfig("foo", cfg.DefaultValue("bar"))
	testarossa.NoError(t, err)
	err = con.DefineConfig("int", cfg.Validation("int"), cfg.DefaultValue("5"))
	testarossa.NoError(t, err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		testarossa.True(t, changed("FOO"))
		testarossa.True(t, changed("int"))
		callbackCalled = true
		return nil
	})
	testarossa.NoError(t, err)

	testarossa.Equal(t, "bar", con.Config("foo"))
	testarossa.Equal(t, "5", con.Config("int"))

	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Equal(t, "baz", con.Config("foo"), "New value should be read from configurator")
	testarossa.Equal(t, "5", con.Config("int"), "Invalid value should not be accepted")
	testarossa.False(t, callbackCalled)

	mockCfg.Unsubscribe("POST", "/values")
	mockCfg.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"values":{"foo":"bam","int":"8"}}`))
		return nil
	})

	ctx := context.Background()
	controlapi.NewClient(mockCfg).ForHost("fetch.config.connector").ConfigRefresh(ctx)

	testarossa.Equal(t, "bam", con.Config("foo"))
	testarossa.Equal(t, "8", con.Config("int"))
	testarossa.True(t, callbackCalled)
}

func TestConnector_NoFetchInTestingApp(t *testing.T) {
	t.Parallel()

	plane := rand.AlphaNum64(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("POST", "/values", func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"values":{"foo":"baz"}}`))
		return nil
	})

	err := mockCfg.Startup()
	testarossa.NoError(t, err)
	defer mockCfg.Shutdown()

	// Connector
	con := New("no.fetch.in.testing.app.config.connector")
	con.SetPlane(plane)
	err = con.DefineConfig("foo", cfg.DefaultValue("bar"))
	testarossa.NoError(t, err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		callbackCalled = true
		return nil
	})
	testarossa.NoError(t, err)

	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Equal(t, "bar", con.Config("foo"))
	testarossa.False(t, callbackCalled)
}

func TestConnector_CallbackWhenStarted(t *testing.T) {
	t.Parallel()

	// Connector
	con := New("callback.when.started.config.connector")
	err := con.DefineConfig("foo", cfg.DefaultValue("bar"))
	testarossa.NoError(t, err)
	callbackCalled := 0
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		callbackCalled++
		return nil
	})
	testarossa.NoError(t, err)

	con.SetConfig("foo", "baz")
	testarossa.Equal(t, "baz", con.Config("foo"))
	testarossa.Zero(t, callbackCalled)

	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()
	testarossa.Zero(t, callbackCalled)

	con.SetConfig("foo", "bam")
	testarossa.Equal(t, "bam", con.Config("foo"))
	testarossa.Equal(t, 1, callbackCalled)
}
