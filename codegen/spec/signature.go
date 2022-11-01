package spec

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Signature is the spec of a function signature.
type Signature struct {
	Name       string
	InputArgs  []*Argument
	OutputArgs []*Argument
}

// Argument is an input or output argument of a signature.
type Argument struct {
	Name string
	Type string
}

// UnmarshalYAML parses the signature.
func (sig *Signature) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Custom unmarshaling from string
	str := ""
	if err := unmarshal(&str); err != nil {
		return err
	}

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
			nameValue := strings.Split(arg, " ")
			if len(nameValue) != 2 {
				return errors.Newf("invalid argument '%s'", arg)
			}
			sig.InputArgs = append(sig.InputArgs, &Argument{
				Name: strings.TrimSpace(nameValue[0]),
				Type: strings.TrimSpace(nameValue[1]),
			})
		}
	}

	str = str[closeParen+1:]
	openParen = strings.Index(str, "(")
	if openParen < 0 {
		return nil
	}
	str = str[openParen+1:]
	closeParen = strings.Index(str, ")")
	if closeParen < 0 {
		return errors.New("missing closing parenthesis")
	}

	args = strings.TrimSpace(str[:closeParen])
	if args != "" {
		for _, arg := range strings.Split(args, ",") {
			arg = strings.TrimSpace(arg)
			nameValue := strings.Split(arg, " ")
			if len(nameValue) != 2 {
				return errors.Newf("invalid argument '%s'", arg)
			}
			sig.OutputArgs = append(sig.OutputArgs, &Argument{
				Name: strings.TrimSpace(nameValue[0]),
				Type: strings.TrimSpace(nameValue[1]),
			})
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
	if !isUpperCaseIdentifier(sig.Name) {
		return errors.Newf("invalid signature '%s'", sig.Name)
	}

	allArgs := []*Argument{}
	allArgs = append(allArgs, sig.InputArgs...)
	allArgs = append(allArgs, sig.OutputArgs...)
	for _, arg := range allArgs {
		if arg.Name == "ctx" || arg.Type == "context.Context" ||
			arg.Name == "err" || arg.Type == "error" ||
			!isLowerCaseIdentifier(arg.Name) {
			return errors.Newf("invalid argument '%s'", arg.Name+" "+arg.Type)
		}
	}
	return nil
}
