package spec

import (
	"path/filepath"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Service is the spec of the microservice parsed from service.yaml.
type Service struct {
	Package string `yaml:"-"`

	General   *General `yaml:"general"`
	Configs   []*Handler
	Functions []*Handler
	Webs      []*Handler
	Tickers   []*Handler
	Types     []*Type

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

// FullyQualifyDefinedTypes prepends the API package name to complex types of function arguments.
func (s *Service) FullyQualifyDefinedTypes() {
	if s.fullyQualified {
		return
	}
	s.fullyQualified = true

	apiPkg := s.ShortPackage() + "api."
	for _, w := range s.Functions {
		for _, a := range w.Signature.InputArgs {
			endType := a.EndType()
			if isUpperCaseIdentifier(endType) {
				a.Type = strings.TrimSuffix(a.Type, endType) + apiPkg + endType
			}
		}
		for _, a := range w.Signature.OutputArgs {
			endType := a.EndType()
			if isUpperCaseIdentifier(endType) {
				a.Type = strings.TrimSuffix(a.Type, endType) + apiPkg + endType
			}
		}
	}
}

// ShorthandDefinedTypes removed the API package name from complex types of function arguments.
func (s *Service) ShorthandDefinedTypes() {
	if !s.fullyQualified {
		return
	}
	s.fullyQualified = false

	apiPkg := s.ShortPackage() + "api."
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
			if isUpperCaseIdentifier(fldType) && !typeNames[fldType] {
				return errors.Newf("undeclared field type '%s' in type '%s'", fldType, t.Name)
			}
		}
	}
	for _, fn := range s.Functions {
		for _, a := range fn.Signature.InputArgs {
			if isUpperCaseIdentifier(a.EndType()) && !typeNames[a.EndType()] {
				return errors.Newf("undeclared type '%s' in '%s'", a.EndType(), fn.Signature.OrigString)
			}
		}
		for _, a := range fn.Signature.OutputArgs {
			if isUpperCaseIdentifier(a.EndType()) && !typeNames[a.EndType()] {
				return errors.Newf("undeclared type '%s' in '%s'", a.EndType(), fn.Signature.OrigString)
			}
		}
	}

	return nil
}

// ShortPackage returns only the last portion of the full package path.
func (s *Service) ShortPackage() string {
	return strings.TrimPrefix(s.Package, filepath.Dir(s.Package)+"/")
}

// AllHandlers returns an array holding all handlers of all types.
func (s *Service) AllHandlers() []*Handler {
	var result []*Handler
	result = append(result, s.Configs...)
	result = append(result, s.Functions...)
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
