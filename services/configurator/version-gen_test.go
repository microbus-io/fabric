/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

// Code generated by Microbus. DO NOT EDIT.

package configurator

import (
	"os"
	"testing"

	"github.com/microbus-io/fabric/utils"
	"github.com/stretchr/testify/assert"
)

func TestConfigurator_Versioning(t *testing.T) {
	t.Parallel()
	
	hash, err := utils.SourceCodeSHA256(".")
	if assert.NoError(t, err) {
		assert.Equal(t, hash, SourceCodeSHA256, "SourceCodeSHA256 is not up to date")
	}
	buf, err := os.ReadFile("version-gen.go")
	if assert.NoError(t, err) {
		assert.Contains(t, string(buf), hash, "SHA256 in version-gen.go is not up to date")
	}
}
