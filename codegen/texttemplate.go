package main

import (
	"bytes"
	"embed"
	"os"
	"strings"
	"text/template"
	"unicode"

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
func LoadTemplate(name string) (*TextTemplate, error) {
	b, err := bundle.ReadFile("bundle/" + name)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &TextTemplate{
		content: b,
		name:    name,
	}, nil
}

// Execute the template given a data element.
func (tt *TextTemplate) Execute(data any) ([]byte, error) {
	var buf bytes.Buffer
	funcs := template.FuncMap{
		"CapitalizeIdentifier": capitalizeIdentifier,
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

// Create writes the template to the named file, but only if it doesn't exist.
func (tt *TextTemplate) Create(fileName string, data any) (ok bool, err error) {
	_, err = os.Stat(fileName)
	if errors.Is(err, os.ErrNotExist) {
		return true, tt.Overwrite(fileName, data)
	}
	return false, nil
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

// capitalizeIdentifier will capitalize a lowercase identifier.
// fooBar will become FooBar, htmlPage will become HTMLPage, etc.
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
