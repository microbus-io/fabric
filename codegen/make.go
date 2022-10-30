package main

import (
	"os"
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
)

// makeIntermediate creates the intermediate directory and files.
func makeIntermediate(specs *spec.Service) error {
	printer.Printf("Generating intermediate")
	printer.Indent()
	defer printer.Unindent()

	// Create the directories
	_, err := os.Stat("intermediate")
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir("intermediate", os.ModePerm)
		printer.Printf("mkdir intermediate")
	} else if err != nil {
		return errors.Trace(err)
	}
	_, err = os.Stat("resources")
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir("resources", os.ModePerm)
		printer.Printf("mkdir resources")
	} else if err != nil {
		return errors.Trace(err)
	}

	// Generate intermediate source files
	templateNames := []string{
		"resources/embed-gen",
		"intermediate/todo-gen",
		"intermediate/intermediate-gen",
		"intermediate/configs-gen",
		"intermediate/functions-gen",
		"intermediate/resources-gen",
	}
	for _, n := range templateNames {
		tt, err := LoadTemplate(n + ".txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(n+".go", specs)
		if err != nil {
			return errors.Trace(err)
		}
		printer.Printf(n + ".go")
	}

	return nil
}

// makeImplementation generates service.go and service-gen.go.
func makeImplementation(specs *spec.Service) error {
	printer.Printf("Generating implementation")
	printer.Indent()
	defer printer.Unindent()

	// Overwrite service-gen.go
	tt, err := LoadTemplate("service-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite("service-gen.go", specs)
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf("service-gen.go")

	// Create service.go
	tt, err = LoadTemplate("service.txt")
	if err != nil {
		return errors.Trace(err)
	}
	created, err := tt.Create("service.go", specs)
	if err != nil {
		return errors.Trace(err)
	}
	if created {
		printer.Printf("service.go")
	}

	// Scan .go files for exiting endpoints
	printer.Printf("Scanning for existing endpoints")
	existingEndpoints, err := scanFiles(".go", `func \(svc \*Service\) ([A-Z][a-zA-Z0-9]*)\(`) // func (svc *Service) XXX(
	if err != nil {
		return errors.Trace(err)
	}
	printer.Indent()
	for k := range existingEndpoints {
		printer.Printf(k)
	}
	printer.Unindent()

	// Mark existing endpoints in the specs
	newEndpoints := false
	for _, h := range specs.AllHandlers() {
		if h.Type == "config" && !h.Callback {
			continue
		}
		if existingEndpoints[h.Name()] || existingEndpoints["OnChanged"+h.Name()] {
			h.Exists = true
		} else {
			newEndpoints = true
		}
	}

	// Append new endpoints
	if newEndpoints {
		printer.Printf("Creating new endpoints")
		printer.Indent()
		for _, h := range specs.AllHandlers() {
			if h.Type == "config" && !h.Callback {
				continue
			}
			if !h.Exists {
				if h.Type == "config" {
					printer.Printf("OnChanged%s", h.Name())
				} else {
					printer.Printf("%s", h.Name())
				}
			}
		}
		printer.Unindent()

		tt, err = LoadTemplate("service-append.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.AppendTo("service.go", specs)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// scanFiles scans all files with the indicated suffix for all sub-matches of the regular expression.
func scanFiles(fileSuffix string, regExpression string) (map[string]bool, error) {
	result := map[string]bool{}
	re, err := regexp.Compile(regExpression)
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), fileSuffix) {
			continue
		}

		body, err := os.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
		allSubMatches := re.FindAllStringSubmatch(string(body), -1)
		for _, subMatches := range allSubMatches {
			if len(subMatches) == 2 {
				result[subMatches[1]] = true
			}
		}
	}
	return result, nil
}
