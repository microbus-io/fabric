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
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// Signature is the spec of a function signature.
type Signature struct {
	OrigString string
	Name       string
	InputArgs  []*Argument
	OutputArgs []*Argument
}

// UnmarshalYAML parses the signature.
func (sig *Signature) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Custom unmarshaling from string
	str := ""
	if err := unmarshal(&str); err != nil {
		return err
	}
	sig.OrigString = str
	sig.InputArgs = []*Argument{}
	sig.OutputArgs = []*Argument{}

	openParen := strings.Index(str, "(")
	if openParen < 0 {
		sig.Name = strings.TrimSpace(str)
		return nil
	}
	sig.Name = strings.TrimSpace(str[:openParen])

	str = str[openParen+1:]
	closeParen := strings.Index(str, ")")
	if closeParen < 0 {
		return errors.New("missing closing parenthesis")
	}

	args := strings.TrimSpace(str[:closeParen])
	if args != "" {
		for _, arg := range strings.Split(args, ",") {
			arg = strings.TrimSpace(arg)
			space := strings.Index(arg, " ")
			if space < 0 {
				return errors.Newf("invalid argument '%s'", arg)
			}
			sig.InputArgs = append(sig.InputArgs, &Argument{
				Name: strings.TrimSpace(arg[:space]),
				Type: strings.TrimSpace(strings.TrimLeft(arg[space:], " :")),
			})
		}
	}

	str = str[closeParen+1:]
	openParen = strings.Index(str, "(")
	if openParen >= 0 {
		str = str[openParen+1:]
		closeParen = strings.Index(str, ")")
		if closeParen < 0 {
			return errors.New("missing closing parenthesis")
		}

		args = strings.TrimSpace(str[:closeParen])
		if args != "" {
			for _, arg := range strings.Split(args, ",") {
				arg = strings.TrimSpace(arg)
				space := strings.Index(arg, " ")
				if space < 0 {
					return errors.Newf("invalid argument '%s'", arg)
				}
				sig.OutputArgs = append(sig.OutputArgs, &Argument{
					Name: strings.TrimSpace(arg[:space]),
					Type: strings.TrimSpace(strings.TrimLeft(arg[space:], " :")),
				})
			}
		}
	}

	// Validate
	err := sig.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (sig *Signature) validate() error {
	if !utils.IsUpperCaseIdentifier(sig.Name) {
		return errors.Newf("handler '%s' must start with uppercase in '%s'", sig.Name, sig.OrigString)
	}
	if strings.HasPrefix(sig.Name, "Mock") {
		return errors.Newf("handler '%s' must not start with 'Mock' in '%s'", sig.Name, sig.OrigString)
	}

	allArgs := []*Argument{}
	allArgs = append(allArgs, sig.InputArgs...)
	allArgs = append(allArgs, sig.OutputArgs...)
	for _, arg := range allArgs {
		err := arg.validate()
		if err != nil {
			return errors.Newf("%s in '%s'", err.Error(), sig.OrigString)
		}
	}
	return nil
}

// TestingT returns "testingT" if "t" conflicts with an argument of the function.
// Otherwise, "t" is returned.
func (sig *Signature) TestingT() string {
	for _, arg := range sig.InputArgs {
		if arg.Name == "t" {
			return "testingT"
		}
	}
	for _, arg := range sig.OutputArgs {
		if arg.Name == "t" {
			return "testingT"
		}
	}
	return "t"
}
