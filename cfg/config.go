/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package cfg

import (
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Config is a property used to configure a microservice.
// Although technically public, it is used internally and should not be constructed by microservices directly.
type Config struct {
	Name         string
	Description  string
	DefaultValue string
	Validation   string
	Secret       bool

	Value string
}

// NewConfig creates a new config property.
func NewConfig(name string, options ...Option) (*Config, error) {
	if err := utils.ValidateConfigName(name); err != nil {
		return nil, errors.Trace(err)
	}

	c := &Config{
		Name:       name,
		Validation: "str",
	}
	err := c.Apply(options...)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Apply the provided options to the config.
func (c *Config) Apply(options ...Option) error {
	for _, opt := range options {
		err := opt(c)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
