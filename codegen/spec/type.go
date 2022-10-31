package spec

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Type is a complex type used in a function.
type Type struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Define      []*Field `yaml:"define"`
	Import      string   `yaml:"import"`
}

// ImportSuffix returns the last piece of the import definition,
// which is expected to point to a microservice.
func (t *Type) ImportSuffix() string {
	p := strings.LastIndex(t.Import, "/")
	if p < 0 {
		p = -1
	}
	return t.Import[p+1:]
}

// Validate indicates if the specs are valid.
func (t *Type) Validate() error {
	match, _ := regexp.MatchString(`^[A-Z][a-zA-Z0-9]*$`, t.Name)
	if !match {
		return errors.Newf("invalid type name '%s'", t.Name)
	}
	if t.Import == "" && len(t.Define) == 0 {
		return errors.Newf("missing type specification '%s'", t.Name)
	}
	if t.Import != "" && len(t.Define) > 0 {
		return errors.Newf("ambiguous type specification '%s'", t.Name)
	}
	for _, f := range t.Define {
		err := f.Validate()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
