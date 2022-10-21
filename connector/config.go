package connector

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/utils"
)

// StartupHandler handles the OnStartup callback.
type ConfigChangedHandler func(ctx context.Context, changed map[string]bool) error

// SetOnConfigChanged sets a function to be called when a new config was received from the configurator.
func (c *Connector) SetOnConfigChanged(handler ConfigChangedHandler, options ...cb.Option) error {
	if c.started {
		return errors.New("already started")
	}

	callback, err := cb.NewCallback("onconfigchanged", handler, options...)
	if err != nil {
		return errors.Trace(err)
	}
	c.onConfigChanged = callback
	return nil
}

// DefineConfig defines a property used to configure the microservice.
// Properties must be defined before the service starts.
// Property names are case-insensitive.
func (c *Connector) DefineConfig(name string, options ...cfg.Option) error {
	if c.started {
		return errors.New("already started")
	}

	config, err := cfg.NewConfig(name, options...)
	if err != nil {
		return errors.Trace(err)
	}
	config.Value = config.DefaultValue

	c.configLock.Lock()
	defer c.configLock.Unlock()

	if _, ok := c.configs[strings.ToLower(name)]; ok {
		return errors.Newf("config '%s' is already defined", name)
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

// InitConfig sets the default value of a previously defined property.
// Properties can be initialized only before the microservice starts
// and therefore before values are fetched from the configurator microservice.
// Property names are case-insensitive.
func (c *Connector) InitConfig(name string, value string) error {
	if c.started {
		return errors.New("already started")
	}

	c.configLock.Lock()
	defer c.configLock.Unlock()

	config, ok := c.configs[strings.ToLower(name)]
	if !ok {
		return nil
	}
	if !cfg.Validate(config.Validation, value) {
		return errors.Newf("invalid value '%s' for config property '%s'", value, name)
	}
	config.DefaultValue = value
	config.Value = value
	return nil
}

// logConfigs prints the config properties to the log.
func (c *Connector) logConfigs() {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	for _, config := range c.configs {
		value := config.Value
		if config.Secret {
			value = strings.Repeat("*", len(value))
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
	if len(c.configs) == 0 {
		return nil
	}
	if !c.started {
		return errors.New("not started")
	}

	var req struct {
		Names []string `json:"names"`
	}
	c.configLock.Lock()
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
			changed[config.Name] = true
			if config.Secret {
				setValue = strings.Repeat("*", len(setValue))
			}
			c.LogInfo(ctx, "Config value updated", log.String("name", config.Name), log.String("value", setValue))
		}
	}
	c.configLock.Unlock()

	// Call the callback function, if provided
	if c.onConfigChanged != nil && len(changed) > 0 {
		callbackCtx := c.lifetimeCtx
		cancel := func() {}
		if c.onConfigChanged.TimeBudget > 0 {
			callbackCtx, cancel = context.WithTimeout(c.lifetimeCtx, c.onConfigChanged.TimeBudget)
		}
		err = utils.CatchPanic(func() error {
			return c.onConfigChanged.Handler.(ConfigChangedHandler)(callbackCtx, changed)
		})
		cancel()
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
