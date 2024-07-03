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

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// makeIntegration creates the integration tests.
func (gen *Generator) makeIntegration() error {
	gen.Printer.Debug("Generating integration tests")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	if !gen.specs.General.IntegrationTests {
		gen.Printer.Debug("Disabled in service.yaml")
		return nil
	}

	// Fully qualify the types outside of the API directory
	gen.specs.FullyQualifyTypes()

	// integration-gen_test.go
	fileName := filepath.Join(gen.WorkDir, "integration-gen_test.go")
	tt, err := LoadTemplate(
		"integration-gen_test.txt",
		"integration-gen_test.functions.txt",
		"integration-gen_test.events.txt",
		"integration-gen_test.webs.txt",
		"integration-gen_test.tickers.txt",
		"integration-gen_test.configs.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("integration-gen_test.go")

	// Create integration_test.go if it doesn't exist
	fileName = filepath.Join(gen.WorkDir, "integration_test.go")
	_, err = os.Stat(fileName)
	if errors.Is(err, os.ErrNotExist) {
		tt, err = LoadTemplate("integration_test.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err := tt.Overwrite(fileName, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}
		gen.Printer.Debug("integration_test.go")
	}

	// Scan .go files for existing endpoints
	gen.Printer.Debug("Scanning for existing tests")
	pkg := capitalizeIdentifier(gen.specs.PackageSuffix())
	existingTests, err := gen.scanFiles(
		gen.WorkDir,
		func(file fs.DirEntry) bool {
			return strings.HasSuffix(file.Name(), "_test.go") &&
				!strings.HasSuffix(file.Name(), "-gen_test.go")
		},
		`func Test`+pkg+`_([A-Z][a-zA-Z0-9]*)\(t `,
	) // func TestService_XXX(t *testing.T)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Indent()
	for k := range existingTests {
		gen.Printer.Debug(k)
	}
	gen.Printer.Unindent()

	// Mark existing tests in the specs
	newTests := false
	for _, h := range gen.specs.AllHandlers() {
		if h.Type != "function" && h.Type != "event" && h.Type != "sink" && h.Type != "web" && h.Type != "ticker" && h.Type != "config" {
			continue
		}
		if existingTests[h.Name()] || existingTests["OnChanged"+h.Name()] {
			h.Exists = true
		} else {
			h.Exists = false
			newTests = true
		}
	}

	// Append new handlers
	fileName = filepath.Join(gen.WorkDir, "integration_test.go")
	if newTests {
		tt, err = LoadTemplate("integration_test.append.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.AppendTo(fileName, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}

		gen.Printer.Debug("New tests created")
		gen.Printer.Indent()
		for _, h := range gen.specs.AllHandlers() {
			if h.Type != "function" && h.Type != "event" && h.Type != "sink" && h.Type != "web" && h.Type != "ticker" {
				continue
			}
			if !h.Exists {
				gen.Printer.Debug("%s", h.Name())
			}
		}
		gen.Printer.Unindent()
	}

	return nil
}

// makeIntermediate creates the intermediate directory and files.
func (gen *Generator) makeIntermediate() error {
	gen.Printer.Debug("Generating intermediate")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Fully qualify the types outside of the API directory
	gen.specs.FullyQualifyTypes()

	// Create the directory
	dir := filepath.Join(gen.WorkDir, "intermediate")
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir intermediate")
	} else if err != nil {
		return errors.Trace(err)
	}

	// intermediate-gen.go
	fileName := filepath.Join(gen.WorkDir, "intermediate", "intermediate-gen.go")
	tt, err := LoadTemplate(
		"intermediate/intermediate-gen.txt",
		"intermediate/intermediate-gen.configs.txt",
		"intermediate/intermediate-gen.functions.txt",
		"intermediate/intermediate-gen.metrics.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("intermediate/intermediate-gen.go")

	// mock-gen.go
	fileName = filepath.Join(gen.WorkDir, "intermediate", "mock-gen.go")
	tt, err = LoadTemplate(
		"intermediate/mock-gen.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("intermediate/mock-gen.go")
	return nil
}

// makeResources creates the resources directory and files.
func (gen *Generator) makeResources() error {
	gen.Printer.Debug("Generating resources")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Create the directory
	dir := filepath.Join(gen.WorkDir, "resources")
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir resources")
	} else if err != nil {
		return errors.Trace(err)
	}

	// embed-gen.go
	fileName := filepath.Join(gen.WorkDir, "resources", "embed-gen.go")
	tt, err := LoadTemplate("resources/embed-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("resources/embed-gen.go")

	return nil
}

// makeApp creates the app directory and main.
func (gen *Generator) makeApp() error {
	gen.Printer.Debug("Generating application")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Create the directories
	dir := filepath.Join(gen.WorkDir, "app")
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir app")
	} else if err != nil {
		return errors.Trace(err)
	}

	dir = filepath.Join(gen.WorkDir, "app", gen.specs.PackageSuffix())
	_, err = os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir app/%s", gen.specs.PackageSuffix())
	} else if err != nil {
		return errors.Trace(err)
	}

	// main-gen.go
	fileName := filepath.Join(gen.WorkDir, "app", gen.specs.PackageSuffix(), "main-gen.go")
	tt, err := LoadTemplate("app/main-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("app/%s/main-gen.go", gen.specs.PackageSuffix())

	return nil
}

// makeAPI creates the API directory and files.
func (gen *Generator) makeAPI() error {
	gen.Printer.Debug("Generating client API")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Should not fully qualify types when generating inside the API directory
	gen.specs.ShorthandTypes()

	// Create the directories
	dir := filepath.Join(gen.WorkDir, gen.specs.PackageSuffix()+"api")
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir %sapi", gen.specs.PackageSuffix())
	} else if err != nil {
		return errors.Trace(err)
	}

	// Types
	if len(gen.specs.Types) > 0 {
		// Scan .go files for existing types
		gen.Printer.Debug("Scanning for existing types")
		existingTypes, err := gen.scanFiles(
			dir,
			func(file fs.DirEntry) bool {
				return strings.HasSuffix(file.Name(), ".go") &&
					!strings.HasSuffix(file.Name(), "_test.go") &&
					!strings.HasSuffix(file.Name(), "-gen.go")
			},
			`type ([A-Z][a-zA-Z0-9]*) `, // type XXX
		)
		if err != nil {
			return errors.Trace(err)
		}
		gen.Printer.Indent()
		for k := range existingTypes {
			gen.Printer.Debug(k)
		}
		gen.Printer.Unindent()

		// Mark existing types in the specs
		newTypes := false
		for _, ct := range gen.specs.Types {
			ct.Exists = existingTypes[ct.Name]
			newTypes = newTypes || !ct.Exists
		}

		// Append new type definitions
		if newTypes {
			// Scan entire project type definitions and try to resolve new types
			typeDefs, err := gen.scanProjectTypeDefinitions()
			if err != nil {
				return errors.Trace(err)
			}
			hasImports := false
			for _, ct := range gen.specs.Types {
				if !ct.Exists && len(typeDefs[ct.Name]) == 1 {
					ct.Package = typeDefs[ct.Name][0]
					hasImports = true
				}
			}
			fileName := filepath.Join(dir, "imports-gen.go")
			if hasImports {
				// Create imports-gen.go
				tt, err := LoadTemplate("api/imports-gen.txt")
				if err != nil {
					return errors.Trace(err)
				}
				err = tt.Overwrite(fileName, gen.specs)
				if err != nil {
					return errors.Trace(err)
				}
				gen.Printer.Debug("%sapi/imports-gen.go", gen.specs.PackageSuffix())
			} else {
				os.Remove(fileName)
			}

			// Create a file for each new type
			for _, ct := range gen.specs.Types {
				if !ct.Exists && len(typeDefs[ct.Name]) != 1 {
					fileName := filepath.Join(dir, strings.ToLower(ct.Name)+".go")
					tt, err := LoadTemplate("api/type.txt")
					if err != nil {
						return errors.Trace(err)
					}
					ct.Package = gen.specs.Package // Hack
					err = tt.Overwrite(fileName, ct)
					if err != nil {
						return errors.Trace(err)
					}
					gen.Printer.Debug("%sapi/%s.go", gen.specs.PackageSuffix(), strings.ToLower(ct.Name))
				}
			}
		}
	}

	// clients-gen.go
	fileName := filepath.Join(gen.WorkDir, gen.specs.PackageSuffix()+"api", "clients-gen.go")
	tt, err := LoadTemplate(
		"api/clients-gen.txt",
		"api/clients-gen.webs.txt",
		"api/clients-gen.functions.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("%sapi/clients-gen.go", gen.specs.PackageSuffix())

	return nil
}

// makeImplementation generates service.go and service-gen.go.
func (gen *Generator) makeImplementation() error {
	gen.Printer.Debug("Generating implementation")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Fully qualify the types outside of the API directory
	gen.specs.FullyQualifyTypes()

	// Overwrite service-gen.go
	fileName := filepath.Join(gen.WorkDir, "service-gen.go")
	tt, err := LoadTemplate("service-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("service-gen.go")

	// Create service.go if it doesn't exist
	fileName = filepath.Join(gen.WorkDir, "service.go")
	_, err = os.Stat(fileName)
	if errors.Is(err, os.ErrNotExist) {
		tt, err = LoadTemplate("service.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err := tt.Overwrite(fileName, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}
		gen.Printer.Debug("service.go")
	}

	// Scan .go files for existing endpoints
	gen.Printer.Debug("Scanning for existing handlers")
	existingEndpoints, err := gen.scanFiles(
		gen.WorkDir,
		func(file fs.DirEntry) bool {
			return strings.HasSuffix(file.Name(), ".go") &&
				!strings.HasSuffix(file.Name(), "_test.go") &&
				!strings.HasSuffix(file.Name(), "-gen.go")
		},
		`func \(svc \*Service\) ([A-Z][a-zA-Z0-9]*)\(`, // func (svc *Service) XXX(
	)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Indent()
	for k := range existingEndpoints {
		gen.Printer.Debug(k)
	}
	gen.Printer.Unindent()

	// Mark existing handlers in the specs
	newEndpoints := false
	for _, h := range gen.specs.AllHandlers() {
		if h.Type == "config" && !h.Callback {
			continue
		}
		if existingEndpoints[h.Name()] || existingEndpoints["OnChanged"+h.Name()] {
			h.Exists = true
		} else {
			h.Exists = false
			newEndpoints = true
		}
	}

	// Append new handlers
	fileName = filepath.Join(gen.WorkDir, "service.go")
	if newEndpoints {
		tt, err = LoadTemplate("service.append.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.AppendTo(fileName, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}

		gen.Printer.Debug("New handlers created")
		gen.Printer.Indent()
		for _, h := range gen.specs.AllHandlers() {
			if h.Type == "config" && !h.Callback {
				continue
			}
			if !h.Exists {
				if h.Type == "config" {
					gen.Printer.Debug("OnChanged%s", h.Name())
				} else {
					gen.Printer.Debug("%s", h.Name())
				}
			}
		}
		gen.Printer.Unindent()
	}

	return nil
}

// scanFiles scans all files with the indicated suffix for all sub-matches of the regular expression.
func (gen *Generator) scanFiles(workDir string, filter func(file fs.DirEntry) bool, regExpression string) (map[string]bool, error) {
	result := map[string]bool{}
	re, err := regexp.Compile(regExpression)
	if err != nil {
		return nil, errors.Trace(err)
	}
	files, err := os.ReadDir(workDir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() || !filter(file) {
			continue
		}
		fileName := filepath.Join(workDir, file.Name())
		body, err := os.ReadFile(fileName)
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

// scanProjectTypeDefinitions scans for type definitions in the entire project tree.
func (gen *Generator) scanProjectTypeDefinitions() (map[string][]string, error) {
	found := map[string][]string{}
	gen.Printer.Debug("Scanning project type definitions")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()
	err := gen.scanDirTypeDefinitions(gen.ProjectPath, found)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return found, nil
}

// scanDirTypeDefinitions scans for type definitions in a directory tree.
func (gen *Generator) scanDirTypeDefinitions(workDir string, found map[string][]string) error {
	// Skip directories starting with .
	if strings.HasPrefix(filepath.Base(workDir), ".") {
		return nil
	}
	// Detect if processing a service directory
	_, err := os.Stat(filepath.Join(workDir, "service.yaml"))
	serviceDirectory := err == nil

	if !serviceDirectory {
		// Scan for type definitions in Go files
		typeDefs, err := gen.scanFiles(
			workDir,
			func(file fs.DirEntry) bool {
				return strings.HasSuffix(file.Name(), ".go") &&
					!strings.HasSuffix(file.Name(), "_test.go") &&
					!strings.HasSuffix(file.Name(), "-gen.go")
			},
			`type ([A-Z][a-zA-Z0-9]*) [^=]`, // type XXX struct, type XXX int, etc.
		)
		if err != nil {
			return errors.Trace(err)
		}
		if len(typeDefs) > 0 {
			subPath := strings.TrimPrefix(workDir, gen.ProjectPath)
			pkg := strings.ReplaceAll(filepath.Join(gen.ModulePath, subPath), "\\", "/")
			gen.Printer.Debug(pkg)
			gen.Printer.Indent()
			for k := range typeDefs {
				gen.Printer.Debug(k)
				found[k] = append(found[k], pkg)
			}
			gen.Printer.Unindent()
		}
	}

	// Recurse into sub directories
	files, err := os.ReadDir(workDir)
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() {
			if serviceDirectory &&
				(file.Name() == "intermediate" || file.Name() == "resources" || file.Name() == "app") {
				continue
			}
			err = gen.scanDirTypeDefinitions(filepath.Join(workDir, file.Name()), found)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

// makeTraceReturnedErrors adds errors.Trace to returned errors.
func (gen *Generator) makeTraceReturnedErrors() error {
	gen.Printer.Debug("Tracing returned errors")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	err := gen.makeTraceReturnedErrorsDir(gen.WorkDir)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (gen *Generator) makeTraceReturnedErrorsDir(directory string) error {
	files, err := os.ReadDir(directory)
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		fileName := filepath.Join(directory, file.Name())
		if file.IsDir() {
			if file.Name() == "intermediate" || file.Name() == "resources" {
				continue
			}
			err = gen.makeTraceReturnedErrorsDir(fileName)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if !strings.HasSuffix(file.Name(), ".go") ||
			strings.HasSuffix(file.Name(), "_test.go") ||
			strings.HasSuffix(file.Name(), "-gen.go") {
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
			err = os.WriteFile(fileName, []byte(alteredBody), 0666)
			if err != nil {
				return errors.Trace(err)
			}
			gen.Printer.Debug("%s", strings.TrimLeft(fileName, gen.WorkDir+"/"))
		}
	}

	return nil
}

var traceReturnErrRegexp = regexp.MustCompile(`\n(\s*)return ([^\n]+, )?(err)\n`)

func findReplaceReturnedErrors(body string) (modified string) {
	return traceReturnErrRegexp.ReplaceAllString(body, "\n${1}return ${2}errors.Trace(err)\n")
}

func findReplaceImportErrors(body string) (modified string) {
	hasTracing := strings.Contains(body, "errors.Trace(")

	modified = strings.ReplaceAll(body, "\n"+`import "errors"`+"\n", "\n"+`import "github.com/microbus-io/fabric/errors"`+"\n")

	start := strings.Index(modified, "\nimport (\n")
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
	lines := strings.Split(stmt, "\n")
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
				result.WriteString("\n")
			}
			result.WriteString(whitespace)
			result.WriteString(`"github.com/microbus-io/fabric/errors"`)
			result.WriteString("\n)\n")
		} else {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	result.WriteString(modified[start+end+2:])
	return result.String()
}

func (gen *Generator) makeRefreshSignature() error {
	gen.Printer.Debug("Refreshing handler signatures")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	// Fully qualify the types outside of the API directory
	gen.specs.FullyQualifyTypes()

	files, err := os.ReadDir(gen.WorkDir)
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".go") ||
			strings.HasSuffix(file.Name(), "_test.go") ||
			strings.HasSuffix(file.Name(), "-gen.go") {
			continue
		}
		fileName := filepath.Join(gen.WorkDir, file.Name())
		buf, err := os.ReadFile(fileName)
		if err != nil {
			return errors.Trace(err)
		}
		body := string(buf)
		alteredBody := findReplaceSignature(gen.specs, body)
		alteredBody = findReplaceDescription(gen.specs, alteredBody)
		if body != alteredBody {
			err = os.WriteFile(fileName, []byte(alteredBody), 0666)
			if err != nil {
				return errors.Trace(err)
			}
			gen.Printer.Debug("%s", file.Name())
		}
	}

	return nil
}

// findReplaceSignature updates the signature of functions.
func findReplaceSignature(specs *spec.Service, source string) (modified string) {
	endpoints := []*spec.Handler{}
	endpoints = append(endpoints, specs.Functions...)
	endpoints = append(endpoints, specs.Sinks...)
	for _, fn := range endpoints {
		p := strings.Index(source, "func (svc *Service) "+fn.Name()+"(")
		if p < 0 {
			continue
		}
		fnSig := "func (svc *Service) " + fn.Name() + "(" + fn.In() + ") (" + fn.Out() + ")"
		q := strings.Index(source[p:], fnSig)
		if q != 0 {
			// Signature changed
			nl := strings.Index(source[p:], " {")
			if nl >= 0 {
				source = strings.Replace(source, source[p:p+nl], fnSig, 1)
			}
		}
	}
	return source
}

// findReplaceDescription updates the description of handlers.
func findReplaceDescription(specs *spec.Service, source string) (modified string) {
	desc := fmt.Sprintf("Service implements the %s microservice.\n\n%s\n", specs.General.Host, specs.General.Description)
	desc = strings.TrimSpace(desc)
	source = findReplaceCommentBefore(source, "\ntype Service struct {", desc)

	for _, h := range specs.AllHandlers() {
		source = findReplaceCommentBefore(source, "\nfunc (svc *Service) "+h.Name()+"(", h.Description)
	}
	return source
}

// findReplaceCommentBefore updates the description of handlers.
func findReplaceCommentBefore(source string, searchTerm string, comment string) (modified string) {
	pos := strings.Index(source, searchTerm)
	if pos < 0 {
		return source
	}

	newComment := "/*\n" + comment + "\n*/"

	// /*
	// Comment
	// */
	// func (svc *Service) ...
	q := strings.LastIndex(source[:pos], "*/")
	if q == pos-len("*/") {
		q += len("*/")
		p := strings.LastIndex(source[:pos], "/*")
		if p > 0 && source[p:q] != newComment {
			source = source[:p] + newComment + source[q:]
		}
		return source
	}

	// // Comment
	// func (svc *Service) ...
	p := pos + 1
	q = pos
	for {
		q = strings.LastIndex(source[:q], "\n")
		if q < 0 || !strings.HasPrefix(source[q:], "\n//") {
			break
		}
		p = q + 1
	}
	source = source[:p] + newComment + source[pos:]
	return source
}

// makeVersion generates the versioning files.
func (gen *Generator) makeVersion(version int) error {
	gen.Printer.Debug("Versioning")
	gen.Printer.Indent()
	defer gen.Printer.Unindent()

	hash, err := utils.SourceCodeSHA256(gen.WorkDir)
	if err != nil {
		return errors.Trace(err)
	}

	v := &spec.Version{
		Package:   gen.specs.Package,
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SHA256:    hash,
	}
	gen.Printer.Debug("Version %d", v.Version)
	gen.Printer.Debug("SHA256 %s", v.SHA256)
	gen.Printer.Debug("Timestamp %v", v.Timestamp)

	templateNames := []string{
		"version-gen",
		"version-gen_test",
	}
	for _, n := range templateNames {
		fileName := filepath.Join(gen.WorkDir, n+".go")
		tt, err := LoadTemplate(n + ".txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(fileName, &v)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
