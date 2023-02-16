/*
Copyright 2023 Microbus LLC and various contributors

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

package utils

import (
	"embed"
	"html"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/*
var testData embed.FS

func TestUtils_ResourceLoader(t *testing.T) {
	rl := ResourceLoader{testData}
	assert.Equal(t, "<html>{{ . }}</html>\n", string(rl.LoadFile("testdata/res.txt")))
	assert.Equal(t, "<html>{{ . }}</html>\n", rl.LoadText("testdata/res.txt"))

	assert.Nil(t, rl.LoadFile("testdata/nothing.txt"))
	assert.Equal(t, "", rl.LoadText("testdata/nothing.txt"))

	v, err := rl.LoadTemplate("testdata/res.txt", "<body></body>")
	assert.NoError(t, err)
	assert.Equal(t, "<html><body></body></html>\n", v)

	v, err = rl.LoadTemplate("testdata/res.html", "<body></body>")
	assert.NoError(t, err)
	assert.Equal(t, "<html>"+html.EscapeString("<body></body>")+"</html>\n", v)
}
