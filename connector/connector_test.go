package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_HostAndID(t *testing.T) {
	c := NewConnector()
	assert.Empty(t, c.HostName())
	assert.NotEmpty(t, c.ID())
	c.SetHostName("example.com")
	assert.Equal(t, "example.com", c.HostName())
}

func TestConnector_BadHostName(t *testing.T) {
	c := NewConnector()
	badHosts := []string{
		"$.example.com",
		"my-example.com",
		"my_example.com",
		"example..com",
		"example.com.",
		".example.com",
		".",
		"",
	}
	for _, s := range badHosts {
		err := c.SetHostName(s)
		assert.Error(t, err)
	}
}
