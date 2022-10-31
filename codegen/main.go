package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
	"gopkg.in/yaml.v2"
)

func main() {
	err := mainErr()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\r\n", err)
		os.Exit(-1)
	}
}

func mainErr() error {
	pkgPath, err := identifyPackage()
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf("Package %s", pkgPath)
	printer.Indent()
	defer printer.Unindent()

	dir, err := os.Getwd()
	if err != nil {
		return errors.Trace(err)
	}
	printer.Printf("Directory %s", dir)

	// Prepare service.yaml
	ok, err := prepareServiceYAML()
	if err != nil {
		return errors.Trace(err)
	}

	// Parse service.yaml
	var specs *spec.Service
	if ok {
		b, err := os.ReadFile("service.yaml")
		if err != nil {
			return errors.Trace(err)
		}
		err = yaml.Unmarshal(b, &specs)
		if err != nil {
			return errors.Trace(err)
		}
		printer.Printf("Service.yaml parsed")
	}

	// Process specs
	if specs != nil {
		specs.Package = pkgPath
		err = specs.Validate()
		if err != nil {
			return errors.Trace(err)
		}
		err = makeIntermediate(specs)
		if err != nil {
			return errors.Trace(err)
		}
		err = makeImplementation(specs)
		if err != nil {
			return errors.Trace(err)
		}
		err = makeAPI(specs)
		if err != nil {
			return errors.Trace(err)
		}
		err = makeRefreshSignature(specs)
		if err != nil {
			return errors.Trace(err)
		}
		err = makeRefreshDescription(specs)
		if err != nil {
			return errors.Trace(err)
		}
		err = makeTraceReturnedErrors(specs)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// identifyPackage identifies the full package path of the current working directory.
func identifyPackage() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Trace(err)
	}

	// Locate module name in go.mod
	goModExists := func(path string) bool {
		_, err := os.Stat(filepath.Join(path, "go.mod"))
		return err == nil
	}
	d := cwd
	for !goModExists(d) && d != "/" {
		d = filepath.Dir(d)
	}
	if d == "/" {
		return "", errors.New("unable to locate go.mod in ancestor directory")
	}
	goMod, err := os.ReadFile(filepath.Join(d, "go.mod"))
	if err != nil {
		return "", errors.Trace(err)
	}
	re, err := regexp.Compile(`module (.+)\n`)
	if err != nil {
		return "", errors.Trace(err)
	}
	subMatches := re.FindSubmatch(goMod)
	if len(subMatches) != 2 {
		return "", errors.New("unable to locate module in go.mod")
	}
	modulePath := string(subMatches[1])

	subPath := strings.TrimPrefix(cwd, d)
	return filepath.Join(modulePath, subPath), nil
}
