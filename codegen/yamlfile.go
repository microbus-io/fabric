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
	"bytes"
	"os"
	"path/filepath"

	"github.com/microbus-io/fabric/errors"
)

func (gen *Generator) prepareServiceYAML() (found bool, err error) {
	// Create a new service.yaml if the directory only contains doc.go or
	// if there's an empty service.yaml file
	createNew := false
	fs, err := os.Stat(filepath.Join(gen.WorkDir, "service.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		files, err := os.ReadDir(gen.WorkDir)
		if err != nil {
			return false, errors.Trace(err)
		}
		if len(files) != 1 || files[0].Name() != "doc.go" {
			return false, nil
		}
		createNew = true
	} else if err != nil {
		return false, errors.Trace(err)
	} else if fs.Size() == 0 {
		createNew = true
	}

	if createNew {
		_, err = gen.createServiceYAML()
		if err != nil {
			return false, errors.Trace(err)
		}
		return false, nil // Avoids error processing an empty file
	}

	err = gen.updateServiceYAML()
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// createServiceYAML generates a new service.yaml file.
func (gen *Generator) createServiceYAML() (found bool, err error) {
	tt, err := LoadTemplate("service.yaml.txt")
	if err != nil {
		return false, errors.Trace(err)
	}
	err = tt.Overwrite(filepath.Join(gen.WorkDir, "service.yaml"), nil)
	if err != nil {
		return false, errors.Trace(err)
	}
	gen.Printer.Debug("Service.yaml created")
	return true, nil
}

// updateServiceYAML updates the comments and sections to the latest version.
func (gen *Generator) updateServiceYAML() error {
	// Read the latest version line by line
	tt, err := LoadTemplate("service.yaml.txt")
	if err != nil {
		return errors.Trace(err)
	}
	latest, err := tt.Execute(nil)
	if err != nil {
		return errors.Trace(err)
	}

	var commentBlock bytes.Buffer
	sectionComments := map[string]string{}
	sections := []string{}
	lines := bytes.Split(latest, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("#")) {
			commentBlock.Write(line)
			commentBlock.WriteString("\n")
		} else {
			if commentBlock.Len() > 0 && bytes.HasSuffix(line, []byte(":")) {
				// Map the comment block to the line that follows it
				sectionComments[string(line)] = commentBlock.String()
				sections = append(sections, string(line))
			}
			commentBlock.Reset()
		}
	}

	// Read the current version line by line
	current, err := os.ReadFile(filepath.Join(gen.WorkDir, "service.yaml"))
	if err != nil {
		return errors.Trace(err)
	}

	var content bytes.Buffer
	commentBlock.Reset()
	lines = bytes.Split(current, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("#")) {
			commentBlock.Write(line)
			commentBlock.WriteString("\n")
		} else {
			if bytes.HasSuffix(line, []byte(":")) && sectionComments[string(line)] != "" {
				// Print the latest comment instead of the current one
				content.WriteString(sectionComments[string(line)])
				delete(sectionComments, string(line))
			} else {
				content.Write(commentBlock.Bytes())
			}
			content.Write(line)
			if i < len(lines)-1 {
				content.WriteString("\n")
			}
			commentBlock.Reset()
		}
	}

	// Add new sections
	extraNewline := ""
	if len(lines[len(lines)-1]) > 0 {
		extraNewline = "\n"
	}
	for _, section := range sections {
		if sectionComments[string(section)] == "" {
			continue
		}
		content.WriteString(extraNewline)
		extraNewline = ""
		content.WriteString("\n")
		content.WriteString(sectionComments[string(section)])
		content.WriteString(section)
		content.WriteString("\n")
	}

	// Overwrite the original
	if !bytes.Equal(content.Bytes(), current) {
		file, err := os.Create(filepath.Join(gen.WorkDir, "service.yaml"))
		if err != nil {
			return err
		}
		_, err = file.Write(content.Bytes())
		file.Close()
		if err != nil {
			return err
		}
		gen.Printer.Debug("Service.yaml updated")
	}
	return nil
}
