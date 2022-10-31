package spec

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Field is a fields in a complex type.
type Field struct {
	Name string
	Type string
}

// UnmarshalYAML parses the field definition.
func (f *Field) UnmarshalYAML(unmarshal func(interface{}) error) error {
	str := ""
	if err := unmarshal(&str); err != nil {
		return errors.Trace(err)
	}

	str = strings.TrimSpace(str)
	parts := strings.Split(str, " ")
	if len(parts) != 2 {
		return errors.New("invalid field '%s'", str)
	}

	f.Name = strings.TrimSpace(parts[0])
	f.Type = strings.TrimSpace(parts[1])

	return nil
}

// Validate indicates if the specs are valid.
func (f *Field) Validate() error {
	match, _ := regexp.MatchString(`^[a-z][a-zA-Z0-9]*$`, f.Name)
	if !match {
		return errors.Newf("invalid field name '%s'", f.Name)
	}
	match, _ = regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9]*$`, f.Type)
	if !match {
		return errors.Newf("invalid field type '%s'", f.Type)
	}
	return nil
}
