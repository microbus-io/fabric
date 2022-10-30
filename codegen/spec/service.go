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
	return nil
}
