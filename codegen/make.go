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
func makeIntermediate(specs *spec.Service) error {
	printer.Printf("Generating intermediate")
	printer.Indent()
	defer printer.Unindent()

	// Create the directory
	_, err := os.Stat("intermediate")
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir("intermediate", os.ModePerm)
		printer.Printf("mkdir intermediate")
	} else if err != nil {
		return errors.Trace(err)
	}

	// intermediate.go
	tt, err := LoadTemplate(
		"intermediate/intermediate-gen.txt",
		"intermediate/intermediate-configs.txt",
		"intermediate/intermediate-functions.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite("intermediate/intermediate-gen.go", specs)
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf("intermediate/intermediate-gen.go")
	return nil
}

// makeResources creates the resources directory and files.
func makeResources(specs *spec.Service) error {
	printer.Printf("Generating resources")
	printer.Indent()
	defer printer.Unindent()

	// Create the directory
	_, err := os.Stat("resources")
	if errors.Is(err, os.ErrNotExist) {
		os.Mkdir("resources", os.ModePerm)
		printer.Printf("mkdir resources")
	} else if err != nil {
		return errors.Trace(err)
	}

	// embed-gen.go
	tt, err := LoadTemplate("resources/embed-gen.txt")
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite("resources/embed-gen.go", specs)
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf("resources/embed-gen.go")

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

	// types-gen.go
	if len(specs.Types) > 0 {
		tt, err := LoadTemplate("api/types-gen.txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(specs.ShortPackage()+"api/types-gen.go", specs)
		if err != nil {
			return errors.Trace(err)
		}
		printer.Printf(specs.ShortPackage() + "api/types-gen.go")
	} else {
		err := os.Remove(specs.ShortPackage() + "api/types-gen.go")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.Trace(err)
		}
	}

	// clients-gen.go
	tt, err := LoadTemplate(
		"api/clients-gen.txt",
		"api/clients-webs.txt",
		"api/clients-functions.txt",
	)
	if err != nil {
		return errors.Trace(err)
	}
	err = tt.Overwrite(specs.ShortPackage()+"api/clients-gen.go", specs)
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf(specs.ShortPackage() + "api/clients-gen.go")

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
	printer.Printf("Scanning for existing handlers")
	existingEndpoints, err := scanFiles(".go", `func \(svc \*Service\) ([A-Z][a-zA-Z0-9]*)\(`) // func (svc *Service) XXX(
	if err != nil {
		return errors.Trace(err)
	}
	printer.Indent()
	for k := range existingEndpoints {
		printer.Printf(k)
	}
	printer.Unindent()

	// Mark existing handlers in the specs
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

	// Append new handlers
	if newEndpoints {
		printer.Printf("Creating new handlers")
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
func makeTraceReturnedErrors() error {
	printer.Printf("Tracing returned errors")
	printer.Indent()
	defer printer.Unindent()

	err := makeTraceReturnedErrorsDir(".")
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func makeTraceReturnedErrorsDir(directory string) error {
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
			err = makeTraceReturnedErrorsDir(fileName)
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

func makeRefreshSignature(specs *spec.Service) error {
	printer.Printf("Refreshing signatures")
	printer.Indent()
	defer printer.Unindent()

	files, err := os.ReadDir(".")
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".go") ||
			strings.HasSuffix(file.Name(), "_test.go") || strings.HasSuffix(file.Name(), "-gen.go") {
			continue
		}

		buf, err := os.ReadFile(file.Name())
		if err != nil {
			return errors.Trace(err)
		}
		body := string(buf)
		alteredBody := findReplaceSignature(specs, body)
		if body != alteredBody {
			err = os.WriteFile(file.Name(), []byte(alteredBody), 0666)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

func makeRefreshComments(specs *spec.Service) error {
	printer.Printf("Refreshing comments")
	printer.Indent()
	defer printer.Unindent()

	files, err := os.ReadDir(".")
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".go") ||
			strings.HasSuffix(file.Name(), "_test.go") || strings.HasSuffix(file.Name(), "-gen.go") {
			continue
		}

		buf, err := os.ReadFile(file.Name())
		if err != nil {
			return errors.Trace(err)
		}
		body := string(buf)
		alteredBody := findReplaceDescription(specs, body)
		if body != alteredBody {
			err = os.WriteFile(file.Name(), []byte(alteredBody), 0666)
			if err != nil {
				return errors.Trace(err)
			}
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
				printer.Printf("%s", fn.Name())
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
				printer.Printf("%s", h.Name())
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
		printer.Printf("%s", h.Name())
	}
	return source
}

// makeVersion generates the versioning files.
func makeVersion(pkg string, version int) error {
	printer.Printf("Versioning")
	printer.Indent()
	defer printer.Unindent()

	hash, err := utils.SourceCodeSHA256(".")
	if err != nil {
		return errors.Trace(err)
	}

	v := &spec.Version{
		Package:   pkg,
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SHA256:    hash,
	}
	printer.Printf("Version %d", v.Version)
	printer.Printf("SHA256 %s", v.SHA256)
	printer.Printf("Timestamp %v", v.Timestamp)

	templateNames := []string{
		"version-gen",
		"version-gen_test",
	}
	for _, n := range templateNames {
		tt, err := LoadTemplate(n + ".txt")
		if err != nil {
			return errors.Trace(err)
		}
		err = tt.Overwrite(n+".go", &v)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
