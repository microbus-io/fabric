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
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
	"gopkg.in/yaml.v3"
)

// Generator is the main operator that operates to generate the code.
type Generator struct {
	Force       bool
	WorkDir     string
	ModulePath  string
	ProjectPath string
	Printer     IndentPrinter

	specs *spec.Service
}

// NewGenerator creates a new code generator set to run on
// the current working directory and output to stdout.
func NewGenerator() *Generator {
	return &Generator{
		Printer: &Printer{
			Verbose: true,
		},
	}
}

// Run performs code generation.
func (gen *Generator) Run() error {
	if !strings.HasPrefix(gen.WorkDir, string(os.PathSeparator)) {
		// Use current working directory if one is not explicitly specified
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Trace(err)
		}
		gen.WorkDir = filepath.Join(cwd, gen.WorkDir)
	}

	pkgPath, err := gen.identifyPackage()
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Info("%s", pkgPath)
	gen.Printer.Indent()
	defer gen.Printer.Unindent()
	gen.Printer.Debug("Directory %s", gen.WorkDir)

	// Generate hash
	hash, err := utils.SourceCodeSHA256(gen.WorkDir)
	if err != nil {
		return errors.Trace(err)
	}
	gen.Printer.Debug("SHA256 %s", hash)

	// Read current version
	v, err := gen.currentVersion()
	if err != nil {
		return errors.Trace(err)
	}
	if v != nil {
		gen.Printer.Debug("Version information parsed")
		gen.Printer.Indent()
		gen.Printer.Debug("Version %d", v.Version)
		gen.Printer.Debug("SHA256 %s", v.SHA256)
		gen.Printer.Debug("Timestamp %s", v.Timestamp)
		gen.Printer.Unindent()

		if v.SHA256 == hash {
			if !gen.Force {
				gen.Printer.Debug("No change detected, exiting")
				return nil
			} else {
				gen.Printer.Debug("No change detected, forcing execution")
			}
		} else {
			gen.Printer.Debug("Change detected, processing")
		}
	}

	// Prepare service.yaml
	ok, err := gen.prepareServiceYAML()
	if err != nil {
		return errors.Trace(err)
	}

	// Parse service.yaml
	if ok {
		b, err := os.ReadFile(filepath.Join(gen.WorkDir, "service.yaml"))
		if err != nil {
			return errors.Trace(err)
		}
		gen.specs = &spec.Service{
			Package: pkgPath, // Must be set before parsing
		}
		err = yaml.Unmarshal(b, gen.specs)
		if err != nil {
			return errors.Trace(err)
		}
		gen.Printer.Debug("Service.yaml parsed")
	}

	// Process specs
	if gen.specs != nil {
		err = gen.makeApp()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeAPI()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeResources()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeIntermediate()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeImplementation()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeIntegration()
		if err != nil {
			return errors.Trace(err)
		}
		err = gen.makeRefreshSignature()
		if err != nil {
			return errors.Trace(err)
		}
	}

	err = gen.makeTraceReturnedErrors()
	if err != nil {
		return errors.Trace(err)
	}

	if gen.specs != nil {
		verNum := 1
		if v != nil {
			verNum = v.Version + 1
			if verNum == 7357 { // Reserved to indicate TEST
				verNum++
			}
		}
		err := gen.makeVersion(verNum)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// identifyPackage identifies the full package path of the working directory.
// It scans for the go.mod and combines the module name with the relative path of
// the working directory.
func (gen *Generator) identifyPackage() (string, error) {
	// Locate module name in go.mod
	goModExists := func(path string) bool {
		_, err := os.Stat(filepath.Join(path, "go.mod"))
		return err == nil
	}
	d := gen.WorkDir
	for !goModExists(d) && d != string(os.PathSeparator) {
		d = filepath.Dir(d)
	}
	if d == string(os.PathSeparator) {
		return "", errors.New("unable to locate go.mod in ancestor directory")
	}
	gen.ProjectPath = d
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
	gen.ModulePath = string(subMatches[1])

	subPath := strings.TrimPrefix(gen.WorkDir, gen.ProjectPath)
	pkg := strings.ReplaceAll(filepath.Join(gen.ModulePath, subPath), "\\", "/")
	return pkg, nil
}

// currentVersion loads the version information.
func (gen *Generator) currentVersion() (*spec.Version, error) {
	buf, err := os.ReadFile(filepath.Join(gen.WorkDir, "version-gen.go"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	body := string(buf)
	p := strings.Index(body, "/* {")
	if p < 0 {
		return nil, errors.New("unable to parse version-gen.go")
	}
	q := strings.Index(body[p:], "} */")
	if q < 0 {
		return nil, errors.New("unable to parse version-gen.go")
	}
	j := body[p+3 : p+q+1]
	var v spec.Version
	err = json.Unmarshal([]byte(j), &v)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &v, nil
}
