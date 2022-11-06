package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

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

	// intermediate.go
	fileName := filepath.Join(gen.WorkDir, "intermediate", "intermediate-gen.go")
	tt, err := LoadTemplate(
		"intermediate/intermediate-gen.txt",
		"intermediate/intermediate-configs.txt",
		"intermediate/intermediate-functions.txt",
		"intermediate/intermediate-sinks.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("intermediate/intermediate-gen.go")
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

	hyphenated := strings.ReplaceAll(gen.specs.General.Host, ".", "-")
	dir = filepath.Join(gen.WorkDir, "app", hyphenated)
	_, err = os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir app/%s", hyphenated)
	} else if err != nil {
		return errors.Trace(err)
	}

	// main-gen.go
	fileName := filepath.Join(gen.WorkDir, "app", hyphenated, "main-gen.go")
	tt, err := LoadTemplate("app/main-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("app/%s/main-gen.go", hyphenated)

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
	dir := filepath.Join(gen.WorkDir, gen.specs.ShortPackage()+"api")
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir(dir, os.ModePerm)
		gen.Printer.Debug("mkdir %sapi", gen.specs.ShortPackage())
	} else if err != nil {
		return errors.Trace(err)
	}

	// types-gen.go
	fileName := filepath.Join(gen.WorkDir, gen.specs.ShortPackage()+"api", "types-gen.go")
	if len(gen.specs.Types) > 0 {
		tt, err := LoadTemplate("api/types-gen.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(fileName, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}
		gen.Printer.Debug("%sapi/types-gen.go", gen.specs.ShortPackage())
	} else {
		err := os.Remove(fileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.Trace(err)
		}
	}

	// clients-gen.go
	fileName = filepath.Join(gen.WorkDir, gen.specs.ShortPackage()+"api", "clients-gen.go")
	tt, err := LoadTemplate(
		"api/clients-gen.txt",
		"api/clients-webs.txt",
		"api/clients-functions.txt",
		"api/clients-events.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(fileName, gen.specs)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("%sapi/clients-gen.go", gen.specs.ShortPackage())

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
	existingEndpoints, err := scanFiles(gen.WorkDir, ".go", `func \(svc \*Service\) ([A-Z][a-zA-Z0-9]*)\(`) // func (svc *Service) XXX(
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
			newEndpoints = true
		}
	}

	// Append new handlers
	fileName = filepath.Join(gen.WorkDir, "service.go")
	if newEndpoints {
		tt, err = LoadTemplate("service-append.txt")
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
func scanFiles(workDir string, fileSuffix string, regExpression string) (map[string]bool, error) {
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
		if file.IsDir() || !strings.HasSuffix(file.Name(), fileSuffix) {
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

// makeTraceReturnedErrors adds errors.Trace to returned errors.
func (gen *Generator) makeTraceReturnedErrors() error {
	gen.Printer.Debug("Adding tracing to errors")
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
	for _, fn := range specs.Functions {
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
	for _, h := range specs.AllHandlers() {
		pos := strings.Index(source, "\nfunc (svc *Service) "+h.Name()+"(")
		if pos < 0 {
			continue
		}

		newComment := "/*\n" + h.Description + "\n*/"

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
			continue
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
	}
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
