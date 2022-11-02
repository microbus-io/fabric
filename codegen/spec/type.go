package spec

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Type is a complex type used in a function.
type Type struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Define      map[string]string `yaml:"define"`
	Import      string            `yaml:"import"`
}

// UnmarshalYAML parses the handler.
func (t *Type) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Type
	var x different
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	*t = Type(x)

	// Post processing
	ifEmpty := t.Name + " is a complex type."
	if t.Import != "" {
		ifEmpty = t.Name + " refers to " + t.Import + "/" + t.ImportSuffix() + "api/" + t.Name + "."
	}
	t.Description = conformDesc(
		t.Description,
		ifEmpty,
		t.Name,
	)
	trimmed := map[string]string{}
	for k, v := range t.Define {
		trimmed[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	t.Define = trimmed

	// Validate
	err = t.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (t *Type) validate() error {
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

	if t.Import != "" {
		match, _ = regexp.MatchString(`^[a-z][a-zA-Z0-9\.\-]*(/[a-z][a-zA-Z0-9\.\-]*)*$`, t.Import)
		if !match {
			return errors.Newf("invalid import path '%s' in '%s'", t.Import, t.Name)
		}
	}

	for fName, fType := range t.Define {
		arg := &Argument{
			Name: fName,
			Type: fType,
		}
		err := arg.validate()
		if err != nil {
			return errors.Newf("%s in '%s'", err.Error(), t.Name)
		}
	}
	return nil
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
