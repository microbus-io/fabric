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

package spec

import (
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// General are general properties of the microservice.
type General struct {
	Host             string `yaml:"host"`
	Description      string `yaml:"description"`
	IntegrationTests bool   `yaml:"integrationTests"`
	OpenAPI          bool   `yaml:"openApi"`
}

// UnmarshalYAML parses and validates the YAML.
func (g *General) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different General
	var x different
	x.IntegrationTests = true // Default
	x.OpenAPI = true          // Default
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	*g = General(x)

	// Post processing
	g.Description = conformDesc(
		g.Description,
		`The "`+g.Host+`" microservice.`,
	)

	// Validate
	err = g.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (g *General) validate() error {
	err := utils.ValidateHostname(g.Host)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
