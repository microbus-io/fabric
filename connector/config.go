/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
)

// StartupHandler handles the OnStartup callback.
type ConfigChangedHandler func(ctx context.Context, changed func(string) bool) error

// SetOnConfigChanged adds a function to be called when a new config was received from the configurator.
// Callbacks are called in the order they were added.
func (c *Connector) SetOnConfigChanged(handler ConfigChangedHandler) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onConfigChanged = append(c.onConfigChanged, &callback{
		Name:    "onconfigchanged",
		Handler: handler,
	})
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
func (c *Connector) logConfigs(ctx context.Context) {
	c.configLock.Lock()
	defer c.configLock.Unlock()
	for _, config := range c.configs {
		c.LogInfo(
			ctx,
			"Config",
			log.String("name", config.Name),
			log.String("value", printableConfigValue(config.Value, config.Secret)),
		)
	}
}

// refreshConfig contacts the configurator microservices to fetch values for the config properties.
func (c *Connector) refreshConfig(ctx context.Context, callback bool) error {
	if !c.started {
		return errors.New("not started")
	}
	var fetchedValues struct {
		Values map[string]string `json:"values"`
	}
	if c.deployment == TESTINGAPP {
		c.LogDebug(c.Lifetime(), "Configurator disabled while testing")
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
		c.LogDebug(ctx, "Requesting config values", log.String("names", strings.Join(req.Names, " ")))
		response, err := c.Request(
			ctx,
			pub.POST("https://configurator.sys/values"),
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
				c.LogWarn(ctx, "Invalid config value", log.String("name", config.Name), log.String("value", printableConfigValue(fetchedValue, config.Secret)), log.String("rule", config.Validation))
			}
		}
		if !cfg.Validate(config.Validation, valueToSet) {
			c.configLock.Unlock()
			return errors.Newf("value '%s' of config '%s' doesn't validate against rule '%s'", printableConfigValue(valueToSet, config.Secret), config.Name, config.Validation)
		}
		if valueToSet != config.Value {
			changed[config.Name] = true
			config.Value = valueToSet
			c.LogInfo(ctx, "Config updated", log.String("name", config.Name), log.String("value", printableConfigValue(valueToSet, config.Secret)))
		}
	}
	c.configLock.Unlock()

	// Call the callback function, if provided
	if callback && len(changed) > 0 {
		for i := 0; i < len(c.onConfigChanged); i++ {
			err := c.doCallback(
				c.lifetimeCtx,
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
