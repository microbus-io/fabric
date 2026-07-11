/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package openapi

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/microbus-io/dv8"
)

// applyDV8Type projects the dv8 directives declared on the fields of a type onto its reflected JSON schema,
// e.g. `dv8:"len<=64"` becomes `maxLength: 64`. The projection is best-effort: directives that mutate rather
// than constrain (trim, tolower, toupper) or that JSON Schema cannot express are skipped.
func applyDV8Type(schema *jsonschema.Schema, t reflect.Type) {
	w := &dv8walker{
		defs:    schema.Definitions,
		visited: map[reflect.Type]bool{},
	}
	w.walk(t, schema)
}

// applyDV8FieldTag projects the dv8 directives declared on a struct field onto a schema that was reflected
// from the field's type alone, which drops the field-level tag. Same rationale as fieldTagDescription:
// it keeps query/path parameters and magic HTTP body arguments constrainable by the same tags as body fields.
func applyDV8FieldTag(schema *jsonschema.Schema, field reflect.StructField) {
	applyDV8Directives(schema, field.Type, dv8.ParseTag(field.Tag.Get("dv8")), nil, "")
}

// dv8walker traverses a Go type in parallel with its reflected JSON schema.
type dv8walker struct {
	defs    jsonschema.Definitions
	visited map[reflect.Type]bool
}

// resolve follows a $defs reference to its definition.
func (w *dv8walker) resolve(s *jsonschema.Schema) *jsonschema.Schema {
	if s != nil && strings.HasPrefix(s.Ref, "#/$defs/") {
		return w.defs[strings.TrimPrefix(s.Ref, "#/$defs/")]
	}
	return s
}

// walk descends into the type and its schema in tandem, applying the dv8 tags of each struct field
// to the schema of that field. A named type's definition is mutated only by its own field tags,
// which are identical for all of its uses, and only once.
func (w *dv8walker) walk(t reflect.Type, s *jsonschema.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	s = w.resolve(s)
	if s == nil {
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		if t.Name() != "" {
			if w.visited[t] {
				return
			}
			w.visited[t] = true
		}
		for i := range t.NumField() {
			field := t.Field(i)
			if field.Anonymous && field.Tag.Get("json") == "" {
				// Embedded fields are flattened into the parent's properties
				w.walk(field.Type, s)
				continue
			}
			name := fieldName(field)
			if name == "" || s.Properties == nil {
				continue
			}
			prop, _ := s.Properties.Get(name)
			if prop == nil {
				continue
			}
			applyDV8Directives(prop, field.Type, dv8.ParseTag(field.Tag.Get("dv8")), s, name)
			w.walk(field.Type, prop)
		}
	case reflect.Slice, reflect.Array:
		w.walk(t.Elem(), s.Items)
	case reflect.Map:
		w.walk(t.Elem(), s.AdditionalProperties)
	}
}

// applyDV8Directives applies parsed dv8 directives to the schema of a single field, descending through
// `each` and `key` prefixes into the schemas of a container's elements and keys. A prefixed directive is
// skipped when descent hits a $defs reference: the definition of a named container type is shared by all
// of its uses and must not absorb the constraints of one field.
func applyDV8Directives(target *jsonschema.Schema, t reflect.Type, dirs []dv8.Directive, parent *jsonschema.Schema, name string) {
	for _, dir := range dirs {
		s := target
		ft := t
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		applicable := true
		for _, prefix := range dir.Prefixes {
			if s.Ref != "" {
				applicable = false
				break
			}
			switch {
			case prefix == "each" && (ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array):
				s = s.Items
				ft = ft.Elem()
			case prefix == "each" && ft.Kind() == reflect.Map:
				s = s.AdditionalProperties
				ft = ft.Elem()
			case prefix == "key" && ft.Kind() == reflect.Map:
				if s.PropertyNames == nil {
					s.PropertyNames = &jsonschema.Schema{Type: "string"}
				}
				s = s.PropertyNames
				ft = reflect.TypeOf("")
			default:
				applicable = false
			}
			if !applicable || s == nil {
				applicable = false
				break
			}
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
		}
		if !applicable || s == nil {
			continue
		}
		if len(dir.Prefixes) == 0 {
			applyDV8Directive(s, ft, dir, parent, name)
		} else {
			applyDV8Directive(s, ft, dir, nil, "")
		}
	}
}

// applyDV8Directive translates one dv8 directive into JSON Schema keywords on the field's schema.
func applyDV8Directive(s *jsonschema.Schema, t reflect.Type, dir dv8.Directive, parent *jsonschema.Schema, name string) {
	switch dir.Name {
	case "notzero":
		if parent != nil && !slices.Contains(parent.Required, name) {
			parent.Required = append(parent.Required, name)
		}
		one := uint64(1)
		switch t.Kind() {
		case reflect.String:
			if s.MinLength == nil {
				s.MinLength = &one
			}
		case reflect.Slice, reflect.Array:
			if s.MinItems == nil {
				s.MinItems = &one
			}
		case reflect.Map:
			if s.MinProperties == nil {
				s.MinProperties = &one
			}
		case reflect.Bool:
			s.Const = true
		}
	case "len":
		bound, err := strconv.ParseUint(dir.Value, 10, 64)
		if err != nil {
			return
		}
		var minBound, maxBound **uint64
		switch t.Kind() {
		case reflect.String:
			minBound, maxBound = &s.MinLength, &s.MaxLength
		case reflect.Slice, reflect.Array:
			minBound, maxBound = &s.MinItems, &s.MaxItems
		case reflect.Map:
			minBound, maxBound = &s.MinProperties, &s.MaxProperties
		default:
			return
		}
		switch dir.Operator {
		case "==":
			*minBound, *maxBound = &bound, &bound
		case ">=":
			*minBound = &bound
		case ">":
			b := bound + 1
			*minBound = &b
		case "<=":
			*maxBound = &bound
		case "<":
			if bound > 0 {
				b := bound - 1
				*maxBound = &b
			}
		}
	case "val":
		if !isNumericKind(t.Kind()) {
			return
		}
		_, err := strconv.ParseFloat(dir.Value, 64)
		if err != nil {
			// Duration and time bounds are not JSON numbers
			return
		}
		n := json.Number(dir.Value)
		switch dir.Operator {
		case "==":
			s.Minimum, s.Maximum = n, n
		case ">=":
			s.Minimum = n
		case ">":
			s.ExclusiveMinimum = n
		case "<=":
			s.Maximum = n
		case "<":
			s.ExclusiveMaximum = n
		}
	case "oneof":
		values := strings.Split(dir.Value, "|")
		enum := make([]any, 0, len(values))
		for _, v := range values {
			switch {
			case t.Kind() == reflect.String:
				enum = append(enum, v)
			case isNumericKind(t.Kind()):
				_, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return
				}
				enum = append(enum, json.Number(v))
			default:
				return
			}
		}
		s.Enum = enum
	case "regexp":
		if t.Kind() == reflect.String {
			s.Pattern = dir.Value
		}
	case "default":
		switch {
		case t.Kind() == reflect.String:
			s.Default = dir.Value
		case t.Kind() == reflect.Bool:
			b, err := strconv.ParseBool(dir.Value)
			if err == nil {
				s.Default = b
			}
		case isNumericKind(t.Kind()):
			_, err := strconv.ParseFloat(dir.Value, 64)
			if err == nil {
				s.Default = json.Number(dir.Value)
			}
		}
	}
}

// isNumericKind indicates if the kind is an integer or floating-point kind.
func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
