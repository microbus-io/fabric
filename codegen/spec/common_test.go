package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_KebabCase(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"fooBar":     "foo-bar",
		"FooBar":     "foo-bar",
		"fooBAR":     "foo-bar",
		"FooBAR":     "foo-bar",
		"urlEncoder": "url-encoder",
		"URLEncoder": "url-encoder",
		"foobarX":    "foobar-x",
		"a":          "a",
		"A":          "a",
		"HTTP":       "http",
		"":           "",
	}
	for id, expected := range testCases {
		assert.Equal(t, expected, kebabCase(id), "%s", id)
	}
}
