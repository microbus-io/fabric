package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_ValidateHostName(t *testing.T) {
	valid := []string{
		"hello",
		"hello.WORLD",
		"123.456",
		"1",
	}
	invalid := []string{
		"hello world",
		"hello_world",
		"hello-world",
		"hello..world",
		"hello.",
		".hello",
		"~hello",
		"$",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateHostName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateHostName(x), "%s", x)
	}
}

func TestUtils_ValidateConfigName(t *testing.T) {
	valid := []string{
		"hello",
		"WORLD",
		"hello123",
	}
	invalid := []string{
		"hello world",
		"hello-world",
		"hello_world",
		"hello.world",
		"1hello",
		"_hello",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateConfigName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateConfigName(x), "%s", x)
	}
}

func TestUtils_ValidateTickerName(t *testing.T) {
	valid := []string{
		"hello",
		"WORLD",
		"hello123",
	}
	invalid := []string{
		"hello world",
		"hello-world",
		"hello_world",
		"hello.world",
		"1hello",
		"_hello",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateTickerName(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateTickerName(x), "%s", x)
	}
}

func TestUtils_ValidateURL(t *testing.T) {
	valid := []string{
		"https://example.com:123/path",
		"https://example.com/path",
		"https://example.com/",
		"https://example.com",
		"https://example",
		"//example.com/path",
	}
	invalid := []string{
		"https://example.com:99999/path",
		"https://$.com:123/path",
		"https://example..com/path",
		"https://example.com:123:456/path",
		"example.com/path",
		"/example.com/path",
		"/path",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateURL(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateURL(x), "%s", x)
	}
}
