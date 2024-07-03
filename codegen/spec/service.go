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
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Service is the spec of the microservice parsed from service.yaml.
type Service struct {
	Package string `yaml:"-"`

	General   General    `yaml:"general"`
	Configs   []*Handler `yaml:"configs"`
	Metrics   []*Handler `yaml:"metrics"`
	Functions []*Handler `yaml:"functions"`
	Events    []*Handler `yaml:"events"`
	Sinks     []*Handler `yaml:"sinks"`
	Webs      []*Handler `yaml:"webs"`
	Tickers   []*Handler `yaml:"tickers"`

	Types []*Type `yaml:"-"`

	fullyQualified bool
}

// UnmarshalYAML parses and validates the YAML.
func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Service
	var x different
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	pkg := s.Package
	*s = Service(x)
	s.Package = pkg

	// Validate
	err = s.validate()
	if err != nil {
		return errors.Trace(err)
	}

	// Default alias for metrics (requires the package name)
	for _, metric := range s.Metrics {
		if metric.Alias == "" {
			metric.Alias = utils.ToSnakeCase(s.PackageSuffix() + metric.Name())
		}
	}

	return nil
}

// FullyQualifyTypes prepends the API package name to complex types of function arguments.
func (s *Service) FullyQualifyTypes() {
	if s.fullyQualified {
		return
	}
	s.fullyQualified = true

	apiPkg := s.PackageSuffix() + "api."
	for _, w := range s.AllHandlers() {
		for _, a := range w.Signature.InputArgs {
			endType := a.EndType()
			if utils.IsUpperCaseIdentifier(endType) {
				a.Type = strings.TrimSuffix(a.Type, endType) + apiPkg + endType
			}
		}
		for _, a := range w.Signature.OutputArgs {
			endType := a.EndType()
			if utils.IsUpperCaseIdentifier(endType) {
				a.Type = strings.TrimSuffix(a.Type, endType) + apiPkg + endType
			}
		}
	}
}

// ShorthandTypes removed the API package name from complex types of function arguments.
func (s *Service) ShorthandTypes() {
	if !s.fullyQualified {
		return
	}
	s.fullyQualified = false

	apiPkg := s.PackageSuffix() + "api."
	for _, w := range s.AllHandlers() {
		for _, a := range w.Signature.InputArgs {
			endType := a.EndType()
			if strings.HasPrefix(endType, apiPkg) {
				shorthand := strings.TrimPrefix(endType, apiPkg)
				a.Type = strings.TrimSuffix(a.Type, endType) + shorthand
			}
		}
		for _, a := range w.Signature.OutputArgs {
			endType := a.EndType()
			if strings.HasPrefix(endType, apiPkg) {
				shorthand := strings.TrimPrefix(endType, apiPkg)
				a.Type = strings.TrimSuffix(a.Type, endType) + shorthand
			}
		}
	}
}

// validate validates the data after unmarshaling.
func (s *Service) validate() error {
	// Need to validate when YAML does not include a general section
	err := s.General.validate()
	if err != nil {
		return errors.Trace(err)
	}

	// Disallow duplicate handler names
	handlerNames := map[string]bool{}
	for _, h := range s.AllHandlers() {
		if handlerNames[h.Name()] {
			return errors.Newf("duplicate handler name '%s'", h.Name())
		}
		handlerNames[h.Name()] = true
	}

	// Has to repeat validation after setting the types because
	// the handlers don't know their type during parsing.
	for _, w := range s.Configs {
		w.Type = "config"
	}
	for _, w := range s.Functions {
		w.Type = "function"
	}
	for _, w := range s.Webs {
		w.Type = "web"
	}
	for _, w := range s.Tickers {
		w.Type = "ticker"
	}
	for _, w := range s.Events {
		w.Type = "event"
	}
	for _, w := range s.Sinks {
		w.Type = "sink"
	}
	for _, w := range s.Metrics {
		w.Type = "metric"
	}
	for _, h := range s.AllHandlers() {
		err := h.validate()
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Gather complex types
	typedHandlers := []*Handler{}
	typedHandlers = append(typedHandlers, s.Functions...)
	typedHandlers = append(typedHandlers, s.Events...)
	typedHandlers = append(typedHandlers, s.Sinks...)
	complexTypes := map[string]bool{}
	for _, fn := range typedHandlers {
		for _, a := range fn.Signature.InputArgs {
			endType := a.EndType()
			if utils.IsUpperCaseIdentifier(endType) && !complexTypes[endType] {
				s.Types = append(s.Types, &Type{
					Name: endType},
				)
				complexTypes[endType] = true
			}
		}
		for _, a := range fn.Signature.OutputArgs {
			endType := a.EndType()
			if utils.IsUpperCaseIdentifier(endType) && !complexTypes[endType] {
				s.Types = append(s.Types, &Type{
					Name: endType},
				)
				complexTypes[endType] = true
			}
		}
	}

	return nil
}

// PackageSuffix returns only the last portion of the full package path.
func (s *Service) PackageSuffix() string {
	p := strings.LastIndex(s.Package, "/")
	if p < 0 {
		return s.Package
	}
	return s.Package[p+1:]
}

// AllHandlers returns an array holding all handlers of all types.
func (s *Service) AllHandlers() []*Handler {
	var result []*Handler
	result = append(result, s.Configs...)
	result = append(result, s.Functions...)
	result = append(result, s.Events...)
	result = append(result, s.Sinks...)
	result = append(result, s.Webs...)
	result = append(result, s.Tickers...)
	return result
}
