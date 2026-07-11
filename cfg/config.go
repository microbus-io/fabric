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

package cfg

import (
	"context"

	"github.com/microbus-io/errors"
)

// Config is a property used to configure a microservice.
// Although technically public, it is used internally and should not be constructed by microservices directly.
type Config struct {
	Name         string
	Description  string
	DefaultValue string
	Validation   string
	Validator    func(ctx context.Context, value string) error
	Secret       bool

	Set   bool
	Value string
}

// NewConfig creates a new config property.
func NewConfig(name string, options ...Option) (*Config, error) {
	if name == "" {
		return nil, errors.New("config name is required")
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
