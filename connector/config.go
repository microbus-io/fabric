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

package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
)

// StartupHandler handles the OnStartup callback.
type ConfigChangedHandler func(ctx context.Context, changed func(string) bool) error

// SetOnConfigChanged adds a function to be called when a new config was received from the configurator.
// Callbacks are called in the order they were added.
func (c *Connector) SetOnConfigChanged(handler ConfigChangedHandler, options ...cb.Option) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	callback, err := cb.NewCallback("onconfigchanged", handler, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	c.onConfigChanged = append(c.onConfigChanged, callback)
	return nil
}

// DefineConfig defines a property used to configure the microservice.
// Properties must be defined before the service starts.
// Config property names are case-insensitive.
func (c *Connector) DefineConfig(name string, options ...cfg.Option) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}

	config, err := cfg.NewConfig(name, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	config.Value = config.DefaultValue

	c.configLock.Lock()
	defer c.configLock.Unlock()

	if _, ok := c.configs[strings.ToLower(name)]; ok {
		return c.captureInitErr(errors.Newf("config '%s' is already defined", name))
	}
	c.configs[strings.ToLower(name)] = config
	return nil
}

// Config returns the value of a previously defined property.
// The value of the property is available after the microservice has started
// after being obtained from the configurator microservice.
// Property names are case-insensitive.
func (c *Connector) Config(name string) (value string) {
	c.configLock.Lock()
	defer c.configLock.Unlock()
	config, ok := c.configs[strings.ToLower(name)]
	if ok {
		return config.Value
	}
	return ""
}

// SetConfig sets the value of a previously defined config property.
// This value will be overridden on the next fetch of configs from the configurator system microservice.
// Fetching configs is disabled in the TESTINGAPP environment.
// Config property names are case-insensitive.
func (c *Connector) SetConfig(name string, value any) error {
	c.configLock.Lock()
	config, ok := c.configs[strings.ToLower(name)]
	c.configLock.Unlock()
	if !ok {
		return nil
	}
	v := fmt.Sprintf("%v", value)
	if !cfg.Validate(config.Validation, v) {
		return c.captureInitErr(errors.Newf("invalid value '%s' for config property '%s'", v, name))
	}
	origValue := config.Value
	config.Value = v

	// Call the callback function, if provided
	if c.started && config.Value != origValue {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err := c.doCallback(
				c.lifetimeCtx,
				c.onConfigChanged[i].TimeBudget,
				c.onConfigChanged[i].Name,
				func(ctx context.Context) error {
					f := func(n string) bool {
						return strings.EqualFold(n, name)
					}
					return c.onConfigChanged[i].Handler.(ConfigChangedHandler)(ctx, f)
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

// ResetConfig resets the value of a previously defined config property to its default value.
// This value will be overridden on the next fetch of configs from the configurator system microservice.
// Fetching configs is disabled in the TESTINGAPP environment.
// Config property names are case-insensitive.
func (c *Connector) ResetConfig(name string) error {
	c.configLock.Lock()
	config, ok := c.configs[strings.ToLower(name)]
	c.configLock.Unlock()
	if !ok {
		return nil
	}
	origValue := config.Value
	config.Value = config.DefaultValue

	// Call the callback function, if provided
	if c.started && config.Value != origValue {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err := c.doCallback(
				c.lifetimeCtx,
				c.onConfigChanged[i].TimeBudget,
				c.onConfigChanged[i].Name,
				func(ctx context.Context) error {
					f := func(n string) bool {
						return strings.EqualFold(n, name)
					}
					return c.onConfigChanged[i].Handler.(ConfigChangedHandler)(ctx, f)
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

// logConfigs prints the config properties to the log.
func (c *Connector) logConfigs() {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	for _, config := range c.configs {
		value := config.Value
		if config.Secret {
			n := len(value)
			if n > 16 {
				n = 16
			}
			value = strings.Repeat("*", n)
		}
		if len([]rune(value)) > 40 {
			value = string([]rune(value)[:40]) + "..."
		}
		c.LogInfo(
			c.Lifetime(),
			"Config",
			log.String("name", config.Name),
			log.String("value", value),
		)
	}
}

// refreshConfig contacts the configurator microservices to fetch values for the config properties.
func (c *Connector) refreshConfig(ctx context.Context) error {
	if !c.started {
		return errors.New("not started")
	}
	if c.deployment == TESTINGAPP {
		c.LogDebug(c.Lifetime(), "Configurator disabled while testing")
		return nil
	}

	c.configLock.Lock()
	if len(c.configs) == 0 {
		c.configLock.Unlock()
		return nil
	}
	var req struct {
		Names []string `json:"names"`
	}
	for _, config := range c.configs {
		req.Names = append(req.Names, config.Name)
	}
	c.LogDebug(ctx, "Requesting config values", log.Any("names", req.Names))
	c.configLock.Unlock()

	response, err := c.Request(
		ctx,
		pub.POST("https://configurator.sys/values"),
		pub.Body(req),
	)
	if err != nil {
		return errors.Trace(err)
	}
	var res struct {
		Values map[string]string `json:"values"`
	}
	err = json.NewDecoder(response.Body).Decode(&res)
	if err != nil {
		return errors.Trace(err)
	}
	c.configLock.Lock()
	changed := map[string]bool{}
	for _, config := range c.configs {
		setValue := config.DefaultValue
		if value, ok := res.Values[config.Name]; ok {
			if cfg.Validate(config.Validation, value) {
				setValue = value
			} else {
				c.LogWarn(ctx, "Invalid config value", log.String("name", config.Name), log.String("value", value))
			}
		}
		if setValue != config.Value {
			config.Value = setValue
			changed[strings.ToLower(config.Name)] = true
			if config.Secret {
				setValue = strings.Repeat("*", len(setValue))
			}
			c.LogInfo(ctx, "Config value updated", log.String("name", config.Name), log.String("value", setValue))
		}
	}
	c.configLock.Unlock()

	// Call the callback function, if provided
	if len(changed) > 0 {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err = c.doCallback(
				c.lifetimeCtx,
				c.onConfigChanged[i].TimeBudget,
				c.onConfigChanged[i].Name,
				func(ctx context.Context) error {
					f := func(name string) bool {
						return changed[strings.ToLower(name)]
					}
					return c.onConfigChanged[i].Handler.(ConfigChangedHandler)(ctx, f)
				},
			)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

// With applies options to a connector during initialization and testing.
func (c *Connector) With(options ...func(Service) error) Service {
	for _, opt := range options {
		err := opt(c)
		if err != nil {
			c.captureInitErr(errors.Trace(err))
		}
	}
	return c
}
