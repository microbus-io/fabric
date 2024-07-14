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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestCodegen_YAMLFile(t *testing.T) {
	t.Parallel()

	// Create a temp directory
	dir := "testing-" + rand.AlphaNum32(12)
	os.Mkdir(dir, os.ModePerm)
	defer os.RemoveAll(dir)

	gen := NewGenerator()
	gen.WorkDir = dir

	// Run on an empty directory should do nothing
	err := gen.Run()
	testarossa.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "service.yaml"))
	testarossa.True(t, errors.Is(err, os.ErrNotExist))

	// Create doc.go
	file, err := os.Create(filepath.Join(dir, "doc.go"))
	testarossa.NoError(t, err)
	file.Close()

	// Running now should create service.yaml
	err = gen.Run()
	testarossa.NoError(t, err)
	onDisk, err := os.ReadFile(filepath.Join(dir, "service.yaml"))
	testarossa.NoError(t, err)
	template, err := bundle.ReadFile("bundle/service.yaml.txt")
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, template, onDisk)

	// Delete service.yaml
	os.Remove(filepath.Join(dir, "service.yaml"))
	_, err = os.Stat(filepath.Join(dir, "service.yaml"))
	testarossa.True(t, errors.Is(err, os.ErrNotExist))

	// Create empty service.yaml
	file, err = os.Create(filepath.Join(dir, "service.yaml"))
	testarossa.NoError(t, err)
	file.Close()

	// Running now should create service.yaml
	err = gen.Run()
	testarossa.NoError(t, err)
	onDisk, err = os.ReadFile(filepath.Join(dir, "service.yaml"))
	testarossa.NoError(t, err)
	template, err = bundle.ReadFile("bundle/service.yaml.txt")
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, template, onDisk)

	// Change/remove the comments on disk
	newLines := [][]byte{}
	lines := bytes.Split(onDisk, []byte("\n"))
	for i := range lines {
		if bytes.HasPrefix(lines[i], []byte("#")) {
			if rand.Intn(2) == 0 {
				newLines = append(newLines, []byte("#"+rand.AlphaNum64(8)))
			}
		} else {
			newLines = append(newLines, lines[i])
		}
	}
	err = os.WriteFile(filepath.Join(dir, "service.yaml"), bytes.Join(newLines, []byte("\n")), 0666)
	testarossa.NoError(t, err)

	// Verify that the file changed
	onDisk, err = os.ReadFile(filepath.Join(dir, "service.yaml"))
	testarossa.NoError(t, err)
	template, err = bundle.ReadFile("bundle/service.yaml.txt")
	testarossa.NoError(t, err)
	testarossa.SliceNotEqual(t, template, onDisk)

	// Running now should fix the comments in service.yaml
	err = gen.Run()
	testarossa.Error(t, err) // Missing hostname
	onDisk, err = os.ReadFile(filepath.Join(dir, "service.yaml"))
	testarossa.NoError(t, err)
	template, err = bundle.ReadFile("bundle/service.yaml.txt")
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, template, onDisk)
}
