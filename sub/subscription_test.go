package sub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSub_NewSub(t *testing.T) {
	t.Parallel()

	type testCase struct {
		spec         string
		expectedHost string
		expectedPort int
		expectedPath string
	}
	testCases := []testCase{
		{"", "www.example.com", 443, ""},
		{"/", "www.example.com", 443, "/"},
		{":555", "www.example.com", 555, ""},
		{":555/", "www.example.com", 555, "/"},
		{"/path/with/slash", "www.example.com", 443, "/path/with/slash"},
		{"path/with/no/slash", "www.example.com", 443, "/path/with/no/slash"},
		{"https://good.example.com", "good.example.com", 443, ""},
		{"https://good.example.com/", "good.example.com", 443, "/"},
		{"https://good.example.com:555", "good.example.com", 555, ""},
		{"https://good.example.com:555/", "good.example.com", 555, "/"},
		{"https://good.example.com:555/path", "good.example.com", 555, "/path"},
	}

	for _, tc := range testCases {
		s, err := NewSub("www.example.com", tc.spec, nil)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedHost, s.Host)
		assert.Equal(t, tc.expectedPort, s.Port)
		assert.Equal(t, tc.expectedPath, s.Path)
	}
}

func TestSub_InvalidPort(t *testing.T) {
	t.Parallel()

	badSpecs := []string{
		":x/path",
		":-5/path",
		":1000000/path",
		"https://bad.example.com:1000000/path",
	}
	for _, s := range badSpecs {
		_, err := NewSub("www.example.com", s, nil)
		assert.Error(t, err)
	}
}

func TestSub_Apply(t *testing.T) {
	t.Parallel()

	s, err := NewSub("www.example.com", "/path", nil)
	assert.NoError(t, err)

	assert.Equal(t, "www.example.com", s.Queue)
	s.Apply(NoQueue())
	assert.Equal(t, "", s.Queue)
	s.Apply(Queue("foo"))
	assert.Equal(t, "foo", s.Queue)
	s.Apply(DefaultQueue())
	assert.Equal(t, "www.example.com", s.Queue)
	s.Apply(Pervasive())
	assert.Equal(t, "", s.Queue)
	s.Apply(LoadBalanced())
	assert.Equal(t, "www.example.com", s.Queue)

	err = s.Apply(Queue("$$$"))
	assert.Error(t, err)
}

func TestSub_Canonical(t *testing.T) {
	t.Parallel()

	s, err := NewSub("www.example.com", ":567/path", nil)
	assert.NoError(t, err)
	assert.Equal(t, "www.example.com:567/path", s.Canonical())

	s, err = NewSub("www.example.com", "/path", nil)
	assert.NoError(t, err)
	assert.Equal(t, "www.example.com:443/path", s.Canonical()) // default port 443

	s, err = NewSub("www.example.com", "http://zzz.example.com/path", nil) // http
	assert.NoError(t, err)
	assert.Equal(t, "zzz.example.com:80/path", s.Canonical()) // default port 80 for http
}
