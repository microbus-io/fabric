package main

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
)

//go:embed bundle/*
var bundle embed.FS

// TextTemplate is a text template used to generate code.
type TextTemplate struct {
	content []byte
	name    string
}

// LoadTemplate loads a template from the embedded bundle.
func LoadTemplate(names ...string) (*TextTemplate, error) {
	var buf bytes.Buffer
	for _, name := range names {
		b, err := bundle.ReadFile("bundle/" + name)
		if err != nil {
			return nil, errors.Trace(err)
		}
		buf.Write(b)
	}
	return &TextTemplate{
		content: buf.Bytes(),
		name:    names[0],
	}, nil
}

// Execute the template given a data element.
func (tt *TextTemplate) Execute(data any) ([]byte, error) {
	var buf bytes.Buffer
	funcs := template.FuncMap{
		"CapitalizeIdentifier": capitalizeIdentifier,
		"JoinHandlers":         joinHandlers,
		"PackageSuffix":        packageSuffix,
		"TestingT":             testingT,
	}
	tmpl, err := template.New(tt.name).Funcs(funcs).Parse(string(tt.content))
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = tmpl.ExecuteTemplate(&buf, tt.name, data)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return buf.Bytes(), nil
}

// Overwrite writes the template to the named file, overwriting its content.
func (tt *TextTemplate) Overwrite(fileName string, data any) error {
	generated, err := tt.Execute(data)
	if err != nil {
		return errors.Trace(err)
	}
	file, err := os.Create(fileName) // Overwrite
	if err != nil {
		return errors.Trace(err)
	}
	_, err = file.Write(generated)
	file.Close()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// AppendTo writes the template to the named file.
func (tt *TextTemplate) AppendTo(fileName string, data any) error {
	generated, err := tt.Execute(data)
	if err != nil {
		return errors.Trace(err)
	}
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666) // Append or create
	if err != nil {
		return errors.Trace(err)
	}
	_, err = file.Write(generated)
	file.Close()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

var acronyms = map[string]bool{
	"http":  true,
	"https": true,
	"url":   true,
	"json":  true,
	"xml":   true,
	"html":  true,
	"api":   true,
	"css":   true,
}

// capitalizeIdentifier capitalizes a lowercase identifier.
// fooBar becomes FooBar, htmlPage becomes HTMLPage, etc.
func capitalizeIdentifier(identifier string) string {
	if identifier == "" {
		return identifier
	}
	lcPrefix := identifier
	suffix := ""
	for i, r := range identifier {
		if unicode.IsUpper(r) {
			if i == 0 {
				return identifier // Already uppercase
			}
			lcPrefix = identifier[:i]
			suffix = identifier[i:]
			break
		}
	}
	if acronyms[lcPrefix] {
		return strings.ToUpper(lcPrefix) + suffix
	}
	return strings.ToUpper(lcPrefix[:1]) + lcPrefix[1:] + suffix
}

// joinHandlers merges groups of handlers together.
func joinHandlers(handlers ...[]*spec.Handler) []*spec.Handler {
	result := []*spec.Handler{}
	for _, h := range handlers {
		result = append(result, h...)
	}
	return result
}

// packageSuffix returns the last segment of the path of a package.
func packageSuffix(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, filepath.Dir(pkgPath)+"/")
}

// testingT returns a name for the testing.T argument used in the test harness so that it doesn't clash
// with any arguments or return values defined by the function.
func testingT(sig *spec.Signature) string {
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
