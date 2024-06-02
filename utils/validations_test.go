/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_ValidateHostname(t *testing.T) {
	valid := []string{
		"hello",
		"hello.WORLD",
		"123.456",
		"1",
		"hello_world",
		"hello-world",
	}
	invalid := []string{
		"hello world",
		"hello..world",
		"hello.",
		".hello",
		"~hello",
		"$",
		"",
	}

	for _, x := range valid {
		assert.NoError(t, ValidateHostname(x), "%s", x)
	}
	for _, x := range invalid {
		assert.Error(t, ValidateHostname(x), "%s", x)
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
