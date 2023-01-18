/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
		ifEmpty = t.Name + " refers to " + t.Import + "."
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
		match, _ = regexp.MatchString(`^([a-z][a-zA-Z0-9\.\-]*)(/[a-z][a-zA-Z0-9\.\-]*)*(/[A-Z][a-zA-Z0-9\.\-]*)$`, t.Import)
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
		t.Define[fName] = arg.Type // Type may have been sanitized
	}
	return nil
}

// ImportType returns the last piece of the import path, which is the name of the type.
// "path/to/a/remote/Type" returns "Type".
func (t *Type) ImportType() string {
	p := strings.LastIndex(t.Import, "/")
	if p < 0 {
		return t.Import
	}
	return t.Import[p+1:]
}

// ImportPackage returns the import path, excluding the type name.
// "path/to/a/remote/Type" returns "path/to/a/remote".
func (t *Type) ImportPackage() string {
	p := strings.LastIndex(t.Import, "/")
	if p < 0 {
		return ""
	}
	return t.Import[:p]
}

// ImportPackageSuffix returns the last portion of the package path.
// "path/to/a/remote/Type" returns "remote".
func (t *Type) ImportPackageSuffix() string {
	pkg := t.ImportPackage()
	p := strings.LastIndex(pkg, "/")
	if p < 0 {
		return pkg
	}
	return pkg[p+1:]
}
