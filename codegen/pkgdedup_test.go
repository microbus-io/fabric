package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodegen_ImportDedup(t *testing.T) {
	dd := PkgDedup{}
	dd.Add("path/to/some/package1")
	dd.Add("path/to/some/package")
	dd.Add("path/to/another/package")
	dd.Add("no/conflict/here")
	dd.Add("path/to/another/package1")
	dd.Add("package")
	assert.Equal(t, "package1x1", dd["path/to/some/package1"])
	assert.Equal(t, "package1x2", dd["path/to/another/package1"])
	assert.Equal(t, "package2", dd["path/to/some/package"])
	assert.Equal(t, "package1", dd["path/to/another/package"])
	assert.Equal(t, "package3", dd["package"])
	assert.Equal(t, "here", dd["no/conflict/here"])
}
