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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/utils"
)

// SetOnConfigChanged adds a function to be called when a new config was received from the configurator.
// Callbacks are called in the order they were added.
func (c *Connector) SetOnConfigChanged(handler service.ConfigChangedHandler) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onConfigChanged = append(c.onConfigChanged, handler)
	return nil
}

// DefineConfig defines a property used to configure the microservice.
// Properties must be defined before the service starts.
// Config property names are case-insensitive.
func (c *Connector) DefineConfig(name string, options ...cfg.Option) error {
	if c.IsStarted() {
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
	config, ok := c.configs[strings.ToLower(name)]
	if ok {
		value = config.Value
	}
	c.configLock.Unlock()
	return value
}

// SetConfig sets the value of a previously defined configuration property.
// This value will be overridden on the next fetch of values from the configurator core microservice,
// except in a TESTING deployment wherein the configurator is disabled.
// Configuration property names are case-insensitive.
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
	if c.IsStarted() && config.Value != origValue {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err := utils.CatchPanic(func() error {
				return c.onConfigChanged[i](
					c.lifetimeCtx,
					func(n string) bool {
						return strings.EqualFold(n, name)
					},
				)
			})
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

// ResetConfig resets the value of a previously defined configuration property to its default value.
// This value will be overridden on the next fetch of values from the configurator core microservice,
// except in a TESTING deployment wherein the configurator is disabled.
// Configuration property names are case-insensitive.
func (c *Connector) ResetConfig(name string) error {
	c.configLock.Lock()
	config, ok := c.configs[strings.ToLower(name)]
	c.configLock.Unlock()
	if !ok {
		return nil
	}
	err := c.SetConfig(name, config.DefaultValue)
	return errors.Trace(err)
}

// logConfigs prints the config properties to the log.
func (c *Connector) logConfigs(ctx context.Context) {
	c.configLock.Lock()
	defer c.configLock.Unlock()
	for _, config := range c.configs {
		c.LogInfo(ctx, "Config",
			"name", config.Name,
			"value", printableConfigValue(config.Value, config.Secret),
		)
	}
}

// refreshConfig contacts the configurator microservices to fetch values for the config properties.
func (c *Connector) refreshConfig(ctx context.Context, callback bool) error {
	if !c.IsStarted() {
		return errors.New("not started")
	}
	var fetchedValues struct {
		Values map[string]string `json:"values"`
	}
	if c.deployment == TESTING {
		c.LogDebug(ctx, "Configurator disabled while testing")
		fetchedValues.Values = map[string]string{}
		c.configLock.Lock()
		for _, config := range c.configs {
			fetchedValues.Values[config.Name] = config.Value
		}
		count := len(c.configs)
		c.configLock.Unlock()
		if count == 0 {
			return nil
		}
	} else {
		c.configLock.Lock()
		var req struct {
			Names []string `json:"names"`
		}
		for _, config := range c.configs {
			req.Names = append(req.Names, config.Name)
		}
		count := len(c.configs)
		c.configLock.Unlock()
		if count == 0 {
			return nil
		}
		c.LogDebug(ctx, "Requesting config values",
			"names", strings.Join(req.Names, " "),
		)
		response, err := c.Request(
			ctx,
			pub.POST("https://configurator.core/values"),
			pub.Body(req),
		)
		if err != nil {
			return errors.Trace(err)
		}
		err = json.NewDecoder(response.Body).Decode(&fetchedValues)
		if err != nil {
			return errors.Trace(err)
		}
	}

	c.configLock.Lock()
	changed := map[string]bool{}
	for _, config := range c.configs {
		valueToSet := config.DefaultValue
		if fetchedValue, ok := fetchedValues.Values[config.Name]; ok {
			if cfg.Validate(config.Validation, fetchedValue) {
				valueToSet = fetchedValue
			} else {
				c.LogWarn(ctx, "Invalid config value",
					"name", config.Name,
					"value", printableConfigValue(fetchedValue, config.Secret),
					"rule", config.Validation,
				)
			}
		}
		if !cfg.Validate(config.Validation, valueToSet) {
			c.configLock.Unlock()
			return errors.Newf("value '%s' of config '%s' doesn't validate against rule '%s'", printableConfigValue(valueToSet, config.Secret), config.Name, config.Validation)
		}
		if valueToSet != config.Value {
			changed[config.Name] = true
			config.Value = valueToSet
			c.LogInfo(ctx, "Config updated",
				"name", config.Name,
				"value", printableConfigValue(valueToSet, config.Secret),
			)
		}
	}
	c.configLock.Unlock()

	// Call the callback function, if provided
	if callback && len(changed) > 0 {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err := utils.CatchPanic(func() error {
				return c.onConfigChanged[i](
					ctx,
					func(name string) bool {
						return changed[strings.ToLower(name)]
					},
				)
			})
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

// printableConfigValue prints up to 40 returns up to 40 characters of the value of the config.
// Secret config values are replaced with asterisks.
func printableConfigValue(value string, secret bool) string {
	if secret {
		n := len(value)
		if n > 16 {
			n = 16
		}
		value = strings.Repeat("*", n)
	}
	if len([]rune(value)) > 40 {
		value = string([]rune(value)[:40]) + "..."
	}
	return value
}
