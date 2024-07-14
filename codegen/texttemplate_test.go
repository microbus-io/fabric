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
	"os"
	"testing"

	"github.com/microbus-io/fabric/codegen/spec"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestCodegen_CapitalizeIdentifier(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"fooBar":     "FooBar",
		"fooBAR":     "FooBAR",
		"urlEncoder": "URLEncoder",
		"URLEncoder": "URLEncoder",
		"a":          "A",
		"A":          "A",
		"":           "",
		"id":         "ID",
		"xId":        "XId",
	}
	for id, expected := range testCases {
		testarossa.Equal(t, expected, capitalizeIdentifier(id))
	}
}

func TestCodegen_TextTemplate(t *testing.T) {
	t.Parallel()

	_, err := LoadTemplate("doesn't.exist")
	testarossa.Error(t, err)

	tt, err := LoadTemplate("service.txt")
	testarossa.NoError(t, err)

	var x struct{}
	_, err = tt.Execute(&x)
	testarossa.Error(t, err)

	specs := &spec.Service{
		Package: "testing/text/template",
		General: spec.General{
			Host:        "example.com",
			Description: "Example",
		},
	}
	rendered, err := tt.Execute(specs)
	n := len(rendered)
	testarossa.NoError(t, err)
	testarossa.Contains(t, string(rendered), specs.PackageSuffix())
	testarossa.Contains(t, string(rendered), specs.General.Host)

	fileName := "testing-" + rand.AlphaNum32(12)
	defer os.Remove(fileName)

	err = tt.AppendTo(fileName, specs)
	testarossa.NoError(t, err)
	onDisk, err := os.ReadFile(fileName)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, rendered, onDisk)

	err = tt.AppendTo(fileName, specs)
	testarossa.NoError(t, err)
	onDisk, err = os.ReadFile(fileName)
	testarossa.NoError(t, err)
	testarossa.Equal(t, n*2, len(onDisk))

	err = tt.Overwrite(fileName, specs)
	testarossa.NoError(t, err)
	onDisk, err = os.ReadFile(fileName)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, rendered, onDisk)
}
