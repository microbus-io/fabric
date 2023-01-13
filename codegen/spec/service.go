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
	Databases Databases  `yaml:"databases"`
	Configs   []*Handler `yaml:"configs"`
	Metrics   []*Handler `yaml:"metrics"`
	Types     []*Type    `yaml:"types"`
	Functions []*Handler `yaml:"functions"`
	Events    []*Handler `yaml:"events"`
	Sinks     []*Handler `yaml:"sinks"`
	Webs      []*Handler `yaml:"webs"`
	Tickers   []*Handler `yaml:"tickers"`

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
	*s = Service(x)

	// Validate
	err = s.validate()
	if err != nil {
		return errors.Trace(err)
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
	for _, w := range s.Functions {
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
	for _, w := range s.Functions {
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
	for _, h := range s.AllHandlers() {
		err := h.validate()
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Check that all complex types are declared
	typeNames := map[string]bool{}
	for _, t := range s.Types {
		typeNames[t.Name] = true
	}
	for _, t := range s.Types {
		for _, fldType := range t.Define {
			if utils.IsUpperCaseIdentifier(fldType) && !typeNames[fldType] {
				return errors.Newf("undeclared field type '%s' in type '%s'", fldType, t.Name)
			}
		}
	}
	typedHandlers := []*Handler{}
	typedHandlers = append(typedHandlers, s.Functions...)
	typedHandlers = append(typedHandlers, s.Events...)
	typedHandlers = append(typedHandlers, s.Sinks...)
	for _, fn := range typedHandlers {
		for _, a := range fn.Signature.InputArgs {
			if utils.IsUpperCaseIdentifier(a.EndType()) && !typeNames[a.EndType()] {
				return errors.Newf("undeclared type '%s' in '%s'", a.EndType(), fn.Signature.OrigString)
			}
		}
		for _, a := range fn.Signature.OutputArgs {
			if utils.IsUpperCaseIdentifier(a.EndType()) && !typeNames[a.EndType()] {
				return errors.Newf("undeclared type '%s' in '%s'", a.EndType(), fn.Signature.OrigString)
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

// ImportedTypes returns only types that are imported.
func (s *Service) ImportedTypes() []*Type {
	var result []*Type
	for _, t := range s.Types {
		if t.Import != "" {
			result = append(result, t)
		}
	}
	return result
}

// DefinedTypes returns only types that are defined.
func (s *Service) DefinedTypes() []*Type {
	var result []*Type
	for _, t := range s.Types {
		if len(t.Define) > 0 {
			result = append(result, t)
		}
	}
	return result
}
