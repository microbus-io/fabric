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

package cfg

import "github.com/microbus-io/fabric/errors"

// Option is used to construct a request in Connector.Publish
type Option func(c *Config) error

// Description sets the description of the config property.
// The description is intended for humans and should explain the purpose of the config property
// and how it will impact the microservice.
func Description(description string) Option {
	return func(c *Config) error {
		c.Description = description
		return nil
	}
}

// DefaultValue sets the value to be used as default of the config property when no value is explicitly set.
// If validation is set, the default value must pass validation.
func DefaultValue(defaultValue string) Option {
	return func(c *Config) error {
		if c.Validation != "" && defaultValue != "" && !Validate(c.Validation, defaultValue) {
			return errors.Newf("default value '%s' of config '%s' doesn't validate against rule '%s'", defaultValue, c.Name, c.Validation)
		}
		c.DefaultValue = defaultValue
		return nil
	}
}

/*
Validation sets the validation rule of the config property.

Valid rules are:

	str ^[a-zA-Z0-9]+$
	bool
	int [0,60]
	float [0.0,1.0)
	dur (0s,24h]
	set Red|Green|Blue
	url
	email
	json

Whereas the following types are synonymous:

	str, string, text, (empty)
	bool, boolean
	int, integer, long
	float, double, decimal, number
	dur, duration
*/
func Validation(validation string) Option {
	return func(c *Config) error {
		if !checkRule(validation) {
			return errors.Newf("invalid validation rule '%s' for config '%s'", validation, c.Name)
		}
		if c.DefaultValue != "" && !Validate(validation, c.DefaultValue) {
			return errors.Newf("default value '%s' of config '%s' doesn't validate against rule '%s'", c.DefaultValue, c.Name, validation)
		}
		c.Validation = validation
		return nil
	}
}

// Secret indicates that the config property's value should be considered a secret.
func Secret() Option {
	return func(c *Config) error {
		c.Secret = true
		return nil
	}
}
