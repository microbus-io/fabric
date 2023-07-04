/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
