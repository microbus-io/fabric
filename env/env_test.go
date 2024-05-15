package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv_FileOverridesOS(t *testing.T) {
	// No parallel

	os.Chdir("testdata")
	defer os.Chdir("..")

	// File overrides OS
	os.Setenv("X5981245X", "InOS")
	defer os.Unsetenv("X5981245X")
	assert.Equal(t, "InOS", os.Getenv("X5981245X"))
	assert.Equal(t, "InFile", Get("X5981245X"))

	// Case sensitive keys
	assert.Equal(t, "infile", Get("x5981245x"))
	assert.NotEqual(t, Get("X5981245X"), os.Getenv("x5981245x"))

	// Push/pop
	Push("X5981245X", "Pushed")
	assert.Equal(t, "Pushed", Get("X5981245X"))
	Pop("X5981245X")
	assert.Equal(t, "InFile", Get("X5981245X"))
	assert.Panics(t, func() {
		Pop("X5981245X")
	})

	// Lookup
	_, ok := Lookup("X5981245X")
	assert.True(t, ok)
	_, ok = Lookup("x5981245x")
	assert.True(t, ok)
	_, ok = Lookup("Y5981245Y")
	assert.False(t, ok)
}
