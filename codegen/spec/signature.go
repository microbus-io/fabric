package spec

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// Signature is the spec of a function signature.
type Signature struct {
	Name       string
	InputArgs  []*Argument
	OutputArgs []*Argument
}

// UnmarshalYAML parses the signature.
func (sig *Signature) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
	return nil
}

// Validate indicates if the specs are valid.
func (sig *Signature) Validate() error {
	match, _ := regexp.MatchString(`^[A-Z][a-zA-Z0-9]*$`, sig.Name)
	if !match {
		return errors.Newf("invalid signature '%s'", sig.Name)
	}
	argNameRegexp, err := regexp.Compile(`^[a-z][a-zA-Z0-9]*$`)
	if err != nil {
		return errors.Trace(err)
	}

	allArgs := []*Argument{}
	allArgs = append(allArgs, sig.InputArgs...)
	allArgs = append(allArgs, sig.OutputArgs...)
	for _, arg := range allArgs {
		if arg.Name == "ctx" || arg.Type == "context.Context" ||
			arg.Name == "err" || arg.Type == "error" ||
			!argNameRegexp.MatchString(arg.Name) {
			return errors.Newf("invalid argument '%s'", arg.Name+" "+arg.Type)
		}
	}
	return nil
}

// String returns the signature as a string.
func (sig *Signature) String() string {
	var b strings.Builder
	b.WriteString(sig.Name)
	b.WriteString("(")
	for i, arg := range sig.InputArgs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
	}
	b.WriteString(")")

	if len(sig.OutputArgs) > 0 {
		b.WriteString(" (")
		for i, arg := range sig.OutputArgs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(arg.Name)
			b.WriteString(" ")
			b.WriteString(arg.Type)
		}
		b.WriteString(")")
	}
	return b.String()
}
