package configurator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepository_LoadYAML(t *testing.T) {
	t.Parallel()

	y := `
# Comments should be ok
www.example.com:
  aaa: 111
  multiline: |-
    Line1
    Line2
example.com:
  aaa: xxx
  bbb: 222
  override: 2
com:
  CCC: 333
  override: 1
www.another.com:
  aaa: xxx
empty:
all:
  ddd: 444
  override: 0
`

	var r repository
	err := r.LoadYAML([]byte(y), "")
	assert.NoError(t, err)

	cases := map[string]string{
		"aaa":       "111",
		"bbb":       "222",
		"CCC":       "333",
		"ddd":       "444",
		"override":  "2",
		"multiline": "Line1\nLine2",
	}
	for name, expected := range cases {
		value, ok := r.Value("www.example.com", name)
		assert.True(t, ok)
		assert.Equal(t, expected, value)
	}

	cases = map[string]string{
		"aaa":      "xxx",
		"bbb":      "222",
		"CCC":      "333",
		"ddd":      "444",
		"override": "2",
	}
	for name, expected := range cases {
		value, ok := r.Value("EXAMPLE.com", name)
		assert.True(t, ok)
		assert.Equal(t, expected, value)
	}

	_, ok := r.Value("www.EXAMPLE.com", "foo")
	assert.False(t, ok)
	_, ok = r.Value("example.com", "multiLINE")
	assert.False(t, ok)
}

func TestRepository_LoadYAMLWithScope(t *testing.T) {
	t.Parallel()

	y := `
www.example.com:
  aaa: 111
example.com:
  bbb: 222
com:
  ccc: 333
all:
  ddd: 444
`

	// Load only www.example.com
	r := &repository{}
	err := r.LoadYAML([]byte(y), "www.example.com")
	assert.NoError(t, err)
	cases := map[string]bool{
		"aaa": true,
		"bbb": false,
		"ccc": false,
		"ddd": false,
	}
	for name := range cases {
		_, ok := r.Value("www.example.com", name)
		assert.Equal(t, cases[name], ok)
	}

	// Load only example.com and subdomains
	r = &repository{}
	err = r.LoadYAML([]byte(y), "example.com")
	assert.NoError(t, err)
	cases = map[string]bool{
		"aaa": true,
		"bbb": true,
		"ccc": false,
		"ddd": false,
	}
	for name := range cases {
		_, ok := r.Value("www.example.com", name)
		assert.Equal(t, cases[name], ok)
	}

	// Load only com and subdomains
	r = &repository{}
	err = r.LoadYAML([]byte(y), "com")
	assert.NoError(t, err)
	cases = map[string]bool{
		"aaa": true,
		"bbb": true,
		"ccc": true,
		"ddd": false,
	}
	for name := range cases {
		_, ok := r.Value("www.example.com", name)
		assert.Equal(t, cases[name], ok)
	}

	// Load all
	r = &repository{}
	err = r.LoadYAML([]byte(y), "")
	assert.NoError(t, err)
	cases = map[string]bool{
		"aaa": true,
		"bbb": true,
		"ccc": true,
		"ddd": true,
	}
	for name := range cases {
		_, ok := r.Value("www.example.com", name)
		assert.Equal(t, cases[name], ok)
	}

	// Load all
	r = &repository{}
	err = r.LoadYAML([]byte(y), "all")
	assert.NoError(t, err)
	cases = map[string]bool{
		"aaa": true,
		"bbb": true,
		"ccc": true,
		"ddd": true,
	}
	for name := range cases {
		_, ok := r.Value("www.example.com", name)
		assert.Equal(t, cases[name], ok)
	}
}

func TestRepository_Quals(t *testing.T) {
	t.Parallel()

	var r repository
	err := r.LoadYAML([]byte(`
www.example.com:
  aaa: 111
example.com:
  bbb: 222
  bbbb: 2222
com:
  ccc: 333
all:
  ddd: 444
`), "")
	assert.NoError(t, err)

	var rr repository
	err = rr.LoadYAML([]byte(`
# Comment
example.com:
  bbbb: 2222
  bbb: 222
com:
  CCC: 333
all:
  ddd: 444
www.example.com:
  aaa: 111
`), "")
	assert.NoError(t, err)

	assert.True(t, r.Equals(&rr))
	assert.True(t, rr.Equals(&r))

	var rrr repository
	err = rrr.LoadYAML([]byte(`
example.com:
  b: 2
  bbb: 222
com:
  CCC: 333
all:
  ddd: 444
www.example.com:
  aaa: 111
`), "")
	assert.NoError(t, err)

	assert.False(t, r.Equals(&rrr))
	assert.False(t, rrr.Equals(&r))
}
