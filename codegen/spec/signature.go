package spec

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
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
				Type: strings.TrimSpace(arg[space:]),
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
			space := strings.Index(arg, " ")
			if space < 0 {
				return errors.Newf("invalid argument '%s'", arg)
			}
			sig.OutputArgs = append(sig.OutputArgs, &Argument{
				Name: strings.TrimSpace(arg[:space]),
				Type: strings.TrimSpace(arg[space:]),
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
		return errors.Newf("signature must start with uppercase '%s'", sig.OrigString)
	}

	allArgs := []*Argument{}
	allArgs = append(allArgs, sig.InputArgs...)
	allArgs = append(allArgs, sig.OutputArgs...)
	for _, arg := range allArgs {
		if arg.Name == "ctx" || arg.Type == "context.Context" {
			return errors.Newf("context argument not allowed '%s'", sig.OrigString)
		}
		if arg.Name == "err" || arg.Type == "error" {
			return errors.Newf("error argument not allowed '%s'", sig.OrigString)
		}
		if !isLowerCaseIdentifier(arg.Name) {
			return errors.Newf("argument must start with lowercase '%s'", sig.OrigString)
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
					return errors.Newf("map keys must be strings '%s'", sig.OrigString)
				}
				t = strings.TrimPrefix(t, "map[string]")
				continue
			}
			t = strings.TrimPrefix(t, "time.")
			if strings.Contains(t, ".") {
				return errors.Newf("dots not allowed in types '%s'", sig.OrigString)
			}

			switch {
			case t == "int" || t == "int64" || t == "int32" || t == "int16" || t == "int8":
				arg.Type = strings.TrimSuffix(arg.Type, t) + "int"
			case t == "uint" || t == "uint64" || t == "uint32" || t == "uint16" || t == "uint8":
				arg.Type = strings.TrimSuffix(arg.Type, t) + "int"
			case t == "float32" || t == "float64" || t == "float":
				arg.Type = strings.TrimSuffix(arg.Type, t) + "float64"
			case t == "Time":
				arg.Type = strings.TrimSuffix(arg.Type, t) + "time.Time"
			case t == "Duration":
				arg.Type = strings.TrimSuffix(arg.Type, t) + "time.Duration"
			case t == "byte" || t == "bool" || t == "string":
				// Nothing
			case isLowerCaseIdentifier(t):
				return errors.Newf("unknown primitive type '%s'", sig.OrigString)
			}

			break
		}
	}
	return nil
}
