package spec

import (
	"path/filepath"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Service is the spec of the microservice parsed from service.yaml.
type Service struct {
	Package string `yaml:"-"`

	General   General `yaml:"general"`
	Configs   []*Handler
	Functions []*Handler
	Webs      []*Handler
	Tickers   []*Handler
	Types     []*Type
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

// Validate indicates if the specs are valid.
func (s *Service) Validate() error {
	err := s.General.Validate()
	if err != nil {
		return errors.Trace(err)
	}
	for _, w := range s.Configs {
		w.Type = "config"
		err := w.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, w := range s.Functions {
		w.Type = "func"
		err := w.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, w := range s.Webs {
		w.Type = "web"
		err := w.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, w := range s.Tickers {
		w.Type = "ticker"
		err := w.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, w := range s.Types {
		err := w.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
