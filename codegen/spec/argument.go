/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Argument is an input or output argument of a signature.
type Argument struct {
	Name string
	Type string
}

// EndType returns the final part of the type, excluding map, array and pointer markers.
// map[string]int -> int; []*User -> User; *time.Time -> time.Time
func (arg *Argument) EndType() string {
	star := strings.LastIndex(arg.Type, "*")
	bracket := strings.LastIndex(arg.Type, "]")
	last := star
	if bracket > last {
		last = bracket
	}
	if last < 0 {
		last = -1
	}
	return arg.Type[last+1:]
}

// validate validates the data after unmarshaling.
func (arg *Argument) validate() error {
	if arg.Name == "ctx" || arg.Type == "context.Context" {
		return errors.Newf("context type not allowed")
	}
	if arg.Name == "err" || arg.Type == "error" {
		return errors.Newf("error type not allowed")
	}
	if !utils.IsLowerCaseIdentifier(arg.Name) {
		return errors.Newf("name '%s' must start with lowercase", arg.Name)
	}
	if arg.Name == "testingT" {
		return errors.New("name 'testingT' is reserved")
	}

	t := arg.Type
	for {
		if strings.HasPrefix(t, "[]") {
			t = strings.TrimPrefix(t, "[]")
			continue
		}
		if strings.HasPrefix(t, "*") {
			t = strings.TrimPrefix(t, "*")
			continue
		}
		if strings.HasPrefix(t, "map[") {
			if !strings.HasPrefix(t, "map[string]") {
				return errors.Newf("map keys must be strings in '%s'", arg.Type)
			}
			t = strings.TrimPrefix(t, "map[string]")
			continue
		}
		t = strings.TrimPrefix(t, "time.")
		if strings.Contains(t, ".") {
			return errors.Newf("dot notation not allowed in type '%s'", arg.Type)
		}

		switch {
		case t == "int" || t == "int64" || t == "int32" || t == "int16" || t == "int8" || t == "integer":
			arg.Type = strings.TrimSuffix(arg.Type, t) + "int"
		case t == "uint" || t == "uint64" || t == "uint32" || t == "uint16" || t == "uint8":
			arg.Type = strings.TrimSuffix(arg.Type, t) + "int"
		case t == "float32" || t == "float64" || t == "float" || t == "double":
			arg.Type = strings.TrimSuffix(arg.Type, t) + "float64"
		case t == "boolean" || t == "Boolean":
			arg.Type = strings.TrimSuffix(arg.Type, t) + "bool"
		case t == "Time" || t == "time":
			arg.Type = strings.TrimSuffix(arg.Type, t)
			arg.Type = strings.TrimSuffix(arg.Type, "time.")
			arg.Type += "time.Time"
		case t == "Duration" || t == "duration":
			arg.Type = strings.TrimSuffix(arg.Type, t)
			arg.Type = strings.TrimSuffix(arg.Type, "time.")
			arg.Type += "time.Duration"
		case t == "byte" || t == "bool" || t == "string" || t == "any":
			// Nothing
		case utils.IsLowerCaseIdentifier(t):
			return errors.Newf("unknown primitive type '%s'", t)
		}

		break
	}
	return nil
}
