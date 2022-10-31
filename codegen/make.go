package main

import (
	"os"
	"path/filepath"
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

// makeAPI creates the API directory and files.
func makeAPI(specs *spec.Service) error {
	printer.Printf("Generating client API")
	printer.Indent()
	defer printer.Unindent()

	// Create the directories
	dir := specs.ShortPackage() + "api"
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		printer.Printf("mkdir " + dir)
	} else if err != nil {
		return errors.Trace(err)
	}

	// Generate API source files
	templateNames := []string{
		"api/service-gen",
		"api/client-gen",
		"api/webs-gen",
		"api/functions-gen",
	}
	for _, n := range templateNames {
		tt, err := LoadTemplate(n + ".txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(specs.ShortPackage()+n+".go", specs)
		if err != nil {
			return errors.Trace(err)
		}
		printer.Printf(specs.ShortPackage() + n + ".go")
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
		return nil, errors.Trace(err)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), fileSuffix) {
			continue
		}

		body, err := os.ReadFile(file.Name())
		if err != nil {
			return nil, errors.Trace(err)
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

// makeTraceReturnedErrors adds errors.Trace to returned errors.
func makeTraceReturnedErrors(specs *spec.Service) error {
	printer.Printf("Tracing returned errors")
	printer.Indent()
	defer printer.Unindent()

	err := makeTraceReturnedErrorsDir(specs, ".")
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func makeTraceReturnedErrorsDir(specs *spec.Service, directory string) error {
	files, err := os.ReadDir(directory)
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		fileName := filepath.Join(directory, file.Name())
		if file.IsDir() {
			if file.Name() == "intermediate" || file.Name() == "resources" || file.Name() == specs.ShortPackage()+"api" {
				continue
			}
			err = makeTraceReturnedErrorsDir(specs, fileName)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if !strings.HasSuffix(file.Name(), ".go") ||
			strings.HasSuffix(file.Name(), "_test.go") || strings.HasSuffix(file.Name(), "-gen.go") {
			continue
		}

		buf, err := os.ReadFile(fileName)
		if err != nil {
			return errors.Trace(err)
		}
		body := string(buf)
		alteredBody := findReplaceReturnedErrors(body)
		alteredBody = findReplaceImportErrors(alteredBody)
		if body != alteredBody {
			printer.Printf("%s", fileName)
			err = os.WriteFile(fileName, []byte(alteredBody), 0666)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

var traceReturnErrRegexp = regexp.MustCompile(`\n(\s*)return ([^\n]+, )?(err)\n`)

func findReplaceReturnedErrors(body string) (modified string) {
	return traceReturnErrRegexp.ReplaceAllString(body, "\n${1}return ${2}errors.Trace(err)\n")
}

func findReplaceImportErrors(body string) (modified string) {
	newline := "\n"
	if strings.Contains(body, "\r\n") {
		newline = "\r\n"
	}
	hasTracing := strings.Contains(body, "errors.Trace(")

	modified = strings.ReplaceAll(body, newline+`import "errors"`+newline, newline+`import "github.com/microbus-io/fabric/errors"`+newline)

	start := strings.Index(modified, newline+"import ("+newline)
	if start < 0 {
		return modified
	}
	end := strings.Index(modified[start:], ")")
	if end < 0 {
		return modified
	}

	var result strings.Builder
	result.WriteString(modified[:start])

	stmt := modified[start : start+end+1]
	lines := strings.Split(stmt, newline)
	whitespace := "\t"
	goErrorsFound := false
	microbusErrorsFound := false
	for i, line := range lines {
		if strings.HasSuffix(line, `"errors"`) {
			whitespace = strings.TrimSuffix(line, `"errors"`)
			goErrorsFound = true
			continue
		}
		if strings.HasSuffix(line, `"github.com/microbus-io/fabric/errors"`) {
			microbusErrorsFound = true
		}
		if line == ")" && (goErrorsFound || hasTracing) && !microbusErrorsFound {
			if i >= 2 && lines[i-2] != "import (" {
				result.WriteString(newline)
			}
			result.WriteString(whitespace)
			result.WriteString(`"github.com/microbus-io/fabric/errors"`)
			result.WriteString(newline)
			result.WriteString(")")
			result.WriteString(newline)
		} else {
			result.WriteString(line)
			result.WriteString(newline)
		}
	}
	result.WriteString(modified[start+end+2:])
	return result.String()
}
