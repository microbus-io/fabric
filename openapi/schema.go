/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

import (
	"reflect"
	"strings"
	"time"
	"unsafe"
)

// oapiSchema allows the definition of input and output data types. These types can be objects, but also primitives and arrays.
// https://spec.openapis.org/oas/v3.1.0#schema-object
type oapiSchema struct {
	Ref                  string                 `yaml:"$ref,omitempty"`
	Type                 string                 `yaml:"type,omitempty"`
	Format               string                 `yaml:"format,omitempty"`
	Items                *oapiSchema            `yaml:"items,omitempty"`
	AdditionalProperties *oapiSchema            `yaml:"additionalProperties,omitempty"`
	OneOf                []*oapiSchema          `yaml:"oneOf,omitempty"`
	Properties           map[string]*oapiSchema `yaml:"properties,omitempty"`
	Description          string                 `yaml:"description,omitempty"`
}

func schemaOfStruct(t reflect.Type) *oapiSchema {
	result := &oapiSchema{
		Type:       "object",
		Properties: map[string]*oapiSchema{},
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := fieldName(field)
		if name == "" {
			continue
		}
		schema := schemaOf(field.Type)
		result.Properties[name] = schema
	}
	return result
}

func fieldName(field reflect.StructField) string {
	name := field.Tag.Get("json")
	if comma := strings.Index(name, ","); comma >= 0 {
		name = name[:comma]
	}
	if name == "" {
		// No JSON tag, use field name
		name = field.Name
	}
	if name == "-" {
		// Omitted
		name = ""
	}
	return name
}

func schemaOf(t reflect.Type) *oapiSchema {
	// time.Time uses strings in the RFC3339 format
	if t == reflect.TypeOf(time.Time{}) {
		return &oapiSchema{
			Type:   "string",
			Format: "date-time",
		}
	}

	// time.Duration is int64
	if t == reflect.TypeOf(time.Duration(0)) {
		return &oapiSchema{
			Type:        "integer",
			Format:      "int64",
			Description: "Nanoseconds",
		}
	}

	// See https://swagger.io/docs/specification/data-models/data-types/
	switch t.Kind() {
	case reflect.Int:
		var i int
		n := unsafe.Sizeof(i)
		if n == 4 {
			return &oapiSchema{
				Type:   "integer",
				Format: "int32",
			}
		}
		return &oapiSchema{
			Type:   "integer",
			Format: "int64",
		}
	case reflect.Int64:
		return &oapiSchema{
			Type:   "integer",
			Format: "int64",
		}
	case reflect.Int32:
		return &oapiSchema{
			Type:   "integer",
			Format: "int32",
		}
	case reflect.Float64:
		return &oapiSchema{
			Type:   "number",
			Format: "double",
		}
	case reflect.Float32:
		return &oapiSchema{
			Type:   "number",
			Format: "float",
		}
	case reflect.Bool:
		return &oapiSchema{
			Type: "boolean",
		}
	case reflect.String:
		return &oapiSchema{
			Type: "string",
		}
	case reflect.Array, reflect.Slice:
		// []byte is base64 encoded
		if t.Elem() == reflect.TypeOf([]byte(nil)) {
			return &oapiSchema{
				Type:   "string",
				Format: "byte",
			}
		}
		return &oapiSchema{
			Type:  "array",
			Items: schemaOf(t.Elem()),
		}
	case reflect.Map:
		// See https://swagger.io/docs/specification/data-models/dictionaries/
		return &oapiSchema{
			Type:                 "object",
			AdditionalProperties: schemaOf(t.Elem()),
		}
	case reflect.Ptr:
		return &oapiSchema{
			OneOf: []*oapiSchema{
				schemaOf(t.Elem()),
				{Ref: "#/components/schemas/Nullable"},
			},
		}
	case reflect.Struct:
		return &oapiSchema{
			Ref: "#/components/schemas/" + typeFullName(t),
		}
	}

	// Unknown type
	return &oapiSchema{
		Type:        "object",
		Description: t.Name(),
	}
}

func typeFullName(t reflect.Type) string {
	x := t.PkgPath()
	if x != "" {
		s := strings.LastIndex(x, "/")
		if s > 0 {
			x = x[s+1:]
		}
		x += "/"
	}
	x += t.Name()
	x = strings.Replace(x, "/", "_", -1)
	x = strings.Replace(x, ".", "_", -1)
	return x
}
