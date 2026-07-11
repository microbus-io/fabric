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
	"maps"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"
	"github.com/microbus-io/fabric/utils"
	"go.yaml.in/yaml/v3"
)

// SetOnConfigChanged sets the function to be called when a new config was received from the configurator.
func (c *Connector) SetOnConfigChanged(handler service.ConfigChangedHandler) error {
	if !c.isPhase(shutDown) {
		return c.captureInitErr(errors.New("already started"))
	}
	c.onConfigChanged = handler
	return nil
}

// DefineConfig defines a property used to configure the microservice.
// Properties must be defined before the service starts.
// Config property names are case sensitive.
func (c *Connector) DefineConfig(name string, options ...cfg.Option) error {
	if !c.isPhase(shutDown) {
		return c.captureInitErr(errors.New("already started"))
	}

	config, err := cfg.NewConfig(name, options...)
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	config.Value = config.DefaultValue

	c.configLock.Lock()
	defer c.configLock.Unlock()

	if _, ok := c.configs[name]; ok {
		return c.captureInitErr(errors.New("config '%s' is already defined", name))
	}
	c.configs[name] = config
	return nil
}

// Config returns the value of a previously defined property.
// The value of the property is available after the microservice has started
// after being obtained from the configurator microservice.
// Config property names are case sensitive.
func (c *Connector) Config(name string) (value string) {
	c.configLock.Lock()
	config, ok := c.configs[name]
	if ok {
		value = config.Value
	}
	c.configLock.Unlock()
	return value
}

// refuseInsecureSecrets returns an error when the connector holds secret configs but the transport
// is not secure in a deployed environment (PROD or LAB), and nil otherwise. Secrets travel from the
// configurator over the bus, so without an encrypted transport they would cross the network in the
// clear. The transport's Secure result is passed in so the policy is decidable without a live
// connection.
func (c *Connector) refuseInsecureSecrets(secure bool) error {
	if secure {
		return nil
	}
	if c.deployment != PROD && c.deployment != LAB {
		return nil
	}
	c.configLock.Lock()
	hasSecret := false
	for _, config := range c.configs {
		if config.Secret {
			hasSecret = true
			break
		}
	}
	c.configLock.Unlock()
	if !hasSecret {
		return nil
	}
	return errors.New("refusing to start with secret configs over an insecure transport", "deployment", c.deployment)
}

// validateConfigValue checks a raw config value against the config's validation rule
// and validator function, if set. It must be called without holding configLock because
// the validator function may execute user code.
func (c *Connector) validateConfigValue(ctx context.Context, config *cfg.Config, value string) error {
	if !cfg.Validate(config.Validation, value) {
		return errors.New("value doesn't validate against rule '%s'", config.Validation)
	}
	if config.Validator != nil {
		err := errors.CatchPanic(func() error {
			return config.Validator(ctx, value)
		})
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// SetConfig sets the value of a previously defined configuration property.
// This action is restricted to the TESTING deployment in which the fetching of values from the configurator is disabled.
// Config property names are case sensitive.
func (c *Connector) SetConfig(name string, value any) error {
	if !c.isPhase(shutDown) && c.Deployment() != TESTING {
		return errors.New("setting value of config property '%s' is not allowed outside %s deployment", name, TESTING)
	}
	c.configLock.Lock()
	config, ok := c.configs[name]
	c.configLock.Unlock()
	if !ok {
		return nil
	}
	v := utils.AnyToString(value)
	err := c.validateConfigValue(context.Background(), config, v)
	if err != nil {
		return c.captureInitErr(errors.New("invalid value '%s' for config property '%s'", v, name, err))
	}
	c.configLock.Lock()
	changed := config.Value != v
	config.Value = v
	config.Set = true
	c.configLock.Unlock()

	// Call the callback function, if provided
	if c.isPhase(startedUp) && changed && c.onConfigChanged != nil {
		startTime := time.Now()
		err := errors.CatchPanic(func() error {
			return c.onConfigChanged(
				c.Lifetime(),
				func(n string) bool {
					return strings.EqualFold(n, name)
				},
			)
		})
		_ = c.RecordHistogram(
			c.Lifetime(),
			"microbus_callback_duration_seconds",
			time.Since(startTime).Seconds(),
			"name", "OnConfigChanged",
			"type", "config",
			"error", func() string {
				if err != nil {
					return "ERROR"
				}
				return "OK"
			}(),
		)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// ResetConfig resets the value of a previously defined configuration property to its default value.
// This action is restricted to the TESTING deployment in which the fetching of values from the configurator is disabled.
// Config property names are case sensitive.
func (c *Connector) ResetConfig(name string) error {
	c.configLock.Lock()
	config, ok := c.configs[name]
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
			"value", c.printableConfigValue(config.Value, config.Secret),
		)
	}
}

// refreshConfig contacts the configurator microservices to fetch values for the config properties.
func (c *Connector) refreshConfig(ctx context.Context, callback bool) (err error) {
	if !c.isPhase(startedUp, startingUp) {
		return errors.New("not started")
	}
	var fetchedValues map[string]string
	if c.deployment == TESTING {
		fetchedValues = map[string]string{} // Do not read configs from files
		c.LogDebug(ctx, "Configurator disabled while testing")
		c.configLock.Lock()
		for _, config := range c.configs {
			if config.Set {
				fetchedValues[config.Name] = config.Value
			}
			if _, ok := fetchedValues[config.Name]; !ok {
				fetchedValues[config.Name] = config.Value
			}
		}
		count := len(c.configs)
		c.configLock.Unlock()
		if count == 0 {
			return nil
		}
	} else {
		fetchedValues, err = c.readConfigFile(ctx)
		if err != nil {
			return errors.Trace(err)
		}
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
			pub.POST("https://configurator.core:888/values"),
			pub.Body(req),
		)
		if err != nil && errors.StatusCode(err) == http.StatusNotFound {
			// Backward compatibility
			response, err = c.Request(
				ctx,
				pub.POST("https://configurator.core/values"),
				pub.Body(req),
			)
		}
		if err != nil {
			return errors.Trace(err)
		}
		var responseObj struct {
			Values map[string]string `json:"values"`
		}
		err = json.NewDecoder(response.Body).Decode(&responseObj)
		if err != nil {
			return errors.Trace(err)
		}
		maps.Copy(fetchedValues, responseObj.Values)
	}

	// Snapshot the configs; all fields except Value and Set are immutable once the connector starts.
	// Validation runs outside the lock because the validator function may execute user code.
	c.configLock.Lock()
	configs := make([]*cfg.Config, 0, len(c.configs))
	for _, config := range c.configs {
		configs = append(configs, config)
	}
	c.configLock.Unlock()

	changed := map[string]bool{}
	for _, config := range configs {
		valueToSet := config.DefaultValue
		if fetchedValue, ok := fetchedValues[config.Name]; ok {
			err := c.validateConfigValue(ctx, config, fetchedValue)
			if err == nil {
				valueToSet = fetchedValue
			} else {
				c.LogWarn(ctx, "Invalid config value",
					"name", config.Name,
					"value", c.printableConfigValue(fetchedValue, config.Secret),
					"error", err,
				)
			}
		}
		err := c.validateConfigValue(ctx, config, valueToSet)
		if err != nil {
			return errors.New("invalid value '%s' of config '%s'", c.printableConfigValue(valueToSet, config.Secret), config.Name, err)
		}
		c.configLock.Lock()
		if valueToSet != config.Value {
			changed[config.Name] = true
			config.Value = valueToSet
			c.LogInfo(ctx, "Config updated",
				"name", config.Name,
				"value", c.printableConfigValue(valueToSet, config.Secret),
			)
		}
		c.configLock.Unlock()
	}

	// Call the callback function, if provided
	if callback && len(changed) > 0 && c.onConfigChanged != nil {
		startTime := time.Now()
		err := errors.CatchPanic(func() error {
			return c.onConfigChanged(
				ctx,
				func(name string) bool {
					return changed[name]
				},
			)
		})
		_ = c.RecordHistogram(
			ctx,
			"microbus_callback_duration_seconds",
			time.Since(startTime).Seconds(),
			"name", "OnConfigChanged",
			"type", "config",
			"error", func() string {
				if err != nil {
					return "ERROR"
				}
				return "OK"
			}(),
		)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// printableConfigValue prints up to 40 returns up to 40 characters of the value of the config.
// Secret config values are replaced with asterisks.
func (c *Connector) printableConfigValue(value string, secret bool) string {
	if secret {
		n := len(value)
		if n > 16 {
			n = 16
		}
		value = strings.Repeat("*", n)
	}
	if len([]rune(value)) > 40 {
		value = string([]rune(value)[:39]) + "\u2026"
	}
	return value
}

// readConfigFile looks for configuration properties in config.yaml and config.local.yaml in ancestor directories.
// Properties in nested directories take precedence over ancestor directories.
// Within each directory, config.local.yaml takes precedence over config.yaml.
func (c *Connector) readConfigFile(ctx context.Context) (values map[string]string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Trace(err)
	}
	amalgamated := map[string]map[string]string{}
	dir := ""
	split := strings.Split(wd, string(os.PathSeparator))
	for p := range split {
		dir = string(os.PathSeparator) + path.Join(split[:p+1]...) // Subdirectories take priority over ancestors
		for _, fileName := range []string{
			path.Join(dir, "config.yaml"),
			path.Join(dir, "config.local.yaml"), // Local file takes priority
		} {
			data, err := os.ReadFile(fileName)
			if err == nil {
				var single map[string]map[string]string
				err = yaml.Unmarshal(data, &single)
				if err != nil {
					c.LogError(ctx, "Parsing config file",
						"error", err,
						"file", fileName,
					)
				} else {
					for d := range single {
						if amalgamated[d] == nil {
							amalgamated[d] = map[string]string{}
						}
						maps.Copy(amalgamated[d], single[d])
					}
					c.LogDebug(ctx, "Read config file",
						"file", fileName,
					)
				}
			}
		}
	}
	values = map[string]string{}
	hostname := strings.ToLower(c.Hostname())
	c.configLock.Lock()
	for _, config := range c.configs {
		name := config.Name
		var value string
		var ok bool
		if amalgamated["all"] != nil {
			value, ok = amalgamated["all"][name]
		}
		segments := strings.Split(hostname, ".")
		for i := len(segments) - 1; i >= 0; i-- {
			domain := strings.Join(segments[i:], ".")
			if amalgamated[domain] != nil {
				if v, found := amalgamated[domain][name]; found {
					value, ok = v, true
				}
			}
		}
		if ok {
			values[name] = value
		}
	}
	c.configLock.Unlock()
	return values, nil
}
