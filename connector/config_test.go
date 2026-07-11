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

package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
	"github.com/microbus-io/testarossa"
)

func TestConnector_SetConfig(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("set.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)

	err = con.DefineConfig("s", cfg.DefaultValue("default"))
	assert.NoError(err)
	assert.Equal("default", con.Config("s"))

	err = con.SetConfig("s", "changed")
	assert.NoError(err)
	assert.Equal("changed", con.Config("s"))

	err = con.ResetConfig("s")
	assert.NoError(err)
	assert.Equal("default", con.Config("s"))

	err = con.SetConfig("s", "changed")
	assert.NoError(err)
	assert.Equal("changed", con.Config("s"))

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal("default", con.Config("s")) // Gets reset after fetching from configurator

	err = con.SetConfig("s", "something")
	assert.Error(err)
	assert.Equal("default", con.Config("s"))

	err = con.ResetConfig("s")
	assert.Error(err)
}

func TestConnector_FetchConfig(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	fooValue := "baz"
	intValue := "$$$"
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"values":{"foo":"` + fooValue + `","int":"` + intValue + `"}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("fetch.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)
	err = con.DefineConfig("foo", cfg.DefaultValue("bar"))
	assert.NoError(err)
	err = con.DefineConfig("int", cfg.Validation("int"), cfg.DefaultValue("5"))
	assert.NoError(err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(changed("foo"))
		assert.True(changed("int"))
		callbackCalled = true
		return nil
	})
	assert.NoError(err)

	assert.Equal("bar", con.Config("foo"))
	assert.Equal("5", con.Config("int"))

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal("baz", con.Config("foo"), "New value should be read from configurator")
	assert.Equal("5", con.Config("int"), "Invalid value should not be accepted")
	assert.False(callbackCalled)

	fooValue = "bam"
	intValue = "8"
	_, err = mockCfg.GET(ctx, "https://fetch.config.connector:888/config-refresh")
	assert.NoError(err)

	assert.Equal("bam", con.Config("foo"))
	assert.Equal("8", con.Config("int"))
	assert.True(callbackCalled)
}

func TestConnector_ValidatorConfig(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	jValue := `{"n":5}`
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"values":{"j":` + strconv.Quote(jValue) + `}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("validator.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)
	err = con.DefineConfig("j",
		cfg.Validation("json"),
		cfg.DefaultValue(`{"n":1}`),
		cfg.Validator(func(ctx context.Context, value string) error {
			var v struct {
				N int `json:"n"`
			}
			err := json.Unmarshal([]byte(value), &v)
			if err != nil {
				return err
			}
			if v.N >= 10 {
				return errors.New("n must be less than 10")
			}
			return nil
		}),
	)
	assert.NoError(err)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal(`{"n":5}`, con.Config("j"), "Valid value should be accepted")

	// A value passing the rule but failing the validator falls back to the default
	jValue = `{"n":50}`
	_, err = mockCfg.GET(ctx, "https://validator.config.connector:888/config-refresh")
	assert.NoError(err)
	assert.Equal(`{"n":1}`, con.Config("j"))

	jValue = `{"n":8}`
	_, err = mockCfg.GET(ctx, "https://validator.config.connector:888/config-refresh")
	assert.NoError(err)
	assert.Equal(`{"n":8}`, con.Config("j"))
}

func TestConnector_ValidatorSetConfig(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	con := New("validator.set.config.connector")
	err := con.DefineConfig("j",
		cfg.Validation("json"),
		cfg.DefaultValue(`{"n":1}`),
		cfg.Validator(func(ctx context.Context, value string) error {
			var v struct {
				N int `json:"n"`
			}
			err := json.Unmarshal([]byte(value), &v)
			if err != nil {
				return err
			}
			if v.N >= 10 {
				return errors.New("n must be less than 10")
			}
			return nil
		}),
	)
	assert.NoError(err)

	err = con.SetConfig("j", `{"n":5}`)
	assert.NoError(err)
	assert.Equal(`{"n":5}`, con.Config("j"))

	// A rejected value leaves the current value in place
	err = con.SetConfig("j", `{"n":50}`)
	assert.Error(err)
	assert.Equal(`{"n":5}`, con.Config("j"))
	con.initErr = nil // Clear the captured init error

	// A panicking validator is captured as an error
	err = con.DefineConfig("p",
		cfg.Validator(func(ctx context.Context, value string) error {
			panic("boom")
		}),
	)
	assert.NoError(err)
	err = con.SetConfig("p", "x")
	assert.Error(err)
	con.initErr = nil // Clear the captured init error
}

func TestConnector_NoFetchInTestingApp(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"values":{"foo":"baz"}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("no.fetch.in.testing.app.config.connector")
	con.SetPlane(plane)
	err = con.DefineConfig("foo", cfg.DefaultValue("bar"))
	assert.NoError(err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		callbackCalled = true
		return nil
	})
	assert.NoError(err)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Equal("bar", con.Config("foo"))
	assert.False(callbackCalled)
}

func TestConnector_CallbackWhenStarted(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Connector
	con := New("callback.when.started.config.connector")
	err := con.DefineConfig("foo", cfg.DefaultValue("bar"))
	assert.NoError(err)
	callbackCalled := 0
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		callbackCalled++
		assert.True(changed("foo"))
		return nil
	})
	assert.NoError(err)

	con.SetConfig("foo", "baz")
	assert.Equal("baz", con.Config("foo"))
	assert.Zero(callbackCalled)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)
	assert.Zero(callbackCalled)

	con.SetConfig("foo", "bam")
	assert.Equal("bam", con.Config("foo"))
	assert.Equal(1, callbackCalled)
}

func TestConnector_CaseSensitiveConfig(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	configValue := "bar"
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"values":{"foo-config_":"` + configValue + `"}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("case.sensitive.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)
	err = con.DefineConfig("foo-config_", cfg.DefaultValue("bar"))
	assert.NoError(err)
	callbackCalled := false
	err = con.SetOnConfigChanged(func(ctx context.Context, changed func(string) bool) error {
		assert.True(changed("foo-config_"))
		assert.False(changed("FOO-CONFIG_"))
		callbackCalled = true
		return nil
	})
	assert.Expect(
		err, nil,
		con.Config("foo-config_"), configValue,
		con.Config("FOO-CONFIG_"), "",
	)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.False(callbackCalled)

	configValue = "baz"
	_, err = mockCfg.GET(ctx, "https://case.sensitive.config.connector:888/config-refresh")
	assert.Expect(
		err, nil,
		callbackCalled, true,
		con.Config("foo-config_"), configValue,
		con.Config("FOO-CONFIG_"), "",
	)
}

func TestConnector_ReadFromFile(t *testing.T) {
	// No parallel
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	os.Chdir("testdata/subdir")
	defer os.Chdir("..")
	defer os.Chdir("..")

	// Mock a config service
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"values":{"Provider":"Configurator"}}`))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)

	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)

	// Connector
	con := New("read.from.file.config.connector")
	con.SetDeployment(LAB) // Configs are disabled in TESTING
	con.SetPlane(plane)
	err = con.DefineConfig("SubDir")
	assert.NoError(err)
	err = con.DefineConfig("case")
	assert.NoError(err)
	err = con.DefineConfig("CASE")
	assert.NoError(err)
	err = con.DefineConfig("Domain")
	assert.NoError(err)
	err = con.DefineConfig("Provider")
	assert.NoError(err)
	err = con.DefineConfig("Empty")
	assert.NoError(err)

	err = con.Startup(ctx)
	assert.NoError(err)
	defer con.Shutdown(ctx)

	assert.Expect(
		con.Config("SubDir"), "Child Subdomain",
		con.Config("case"), "lowercase",
		con.Config("CASE"), "UPPERCASE",
		con.Config("Domain"), "Subdomain",
		con.Config("Provider"), "Configurator",
		con.Config("Empty"), "",
		con.Config("Undefined"), "",
	)
}

// TestConnector_RefuseInsecureSecrets pins the fail-closed policy: a connector holding secret
// configs must refuse to start over an insecure transport in a deployed environment, and must be
// allowed in every other combination.
func TestConnector_RefuseInsecureSecrets(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	newCon := func(deployment string, secret bool) *Connector {
		c := New("refuse.insecure.secrets.connector")
		c.SetDeployment(deployment)
		if secret {
			c.DefineConfig("APIKey", cfg.Secret())
		} else {
			c.DefineConfig("Plain", cfg.DefaultValue("x"))
		}
		return c
	}

	// Deployed + secret + insecure transport: refuse.
	assert.Error(newCon(LAB, true).refuseInsecureSecrets(false))
	assert.Error(newCon(PROD, true).refuseInsecureSecrets(false))
	// Secure transport: allowed regardless of deployment or secrets.
	assert.NoError(newCon(LAB, true).refuseInsecureSecrets(true))
	assert.NoError(newCon(PROD, true).refuseInsecureSecrets(true))
	// No secret config: allowed even over an insecure transport.
	assert.NoError(newCon(LAB, false).refuseInsecureSecrets(false))
	assert.NoError(newCon(PROD, false).refuseInsecureSecrets(false))
	// Non-deployed environments are never gated.
	assert.NoError(newCon(TESTING, true).refuseInsecureSecrets(false))
	assert.NoError(newCon(LOCAL, true).refuseInsecureSecrets(false))
}

// TestConnector_SecretConfigTransportGate verifies the fail-closed check is wired into Startup and
// tracks the transport: a deployed connector with a secret config starts over a secure transport
// (short-circuit or TLS NATS) and is refused over an insecure one. It adapts to the CI matrix's
// transport mode rather than assuming one.
func TestConnector_SecretConfigTransportGate(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	plane := utils.RandomIdentifier(12)

	// Mock config service. It defines no secret configs, so it always starts, and its transport
	// reports the environment's security (short-circuit or NATS) that the connector below will see.
	mockCfg := New("configurator.core")
	mockCfg.SetDeployment(LAB) // Configs are disabled in TESTING
	mockCfg.SetPlane(plane)
	mockCfg.Subscribe("Values",
		func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return nil
		},
		sub.At("POST", ":888/values"),
		sub.Web(),
	)
	err := mockCfg.Startup(ctx)
	assert.NoError(err)
	defer mockCfg.Shutdown(ctx)
	secure := mockCfg.transportConn.Secure()

	// A deployed connector with a secret config.
	con := New("secret.transport.gate.connector")
	con.SetDeployment(LAB)
	con.SetPlane(plane)
	err = con.DefineConfig("APIKey", cfg.Secret())
	assert.NoError(err)

	err = con.Startup(ctx)
	if secure {
		assert.NoError(err, "secret config over a secure transport must start")
		con.Shutdown(ctx)
	} else {
		assert.Error(err, "secret config over an insecure transport must be refused")
		if err != nil {
			assert.Contains(err.Error(), "insecure transport")
		}
	}
}
