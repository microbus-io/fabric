package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadEnvars(t *testing.T) {
	environ := []string{
		"Path=...",
		"MICROBUS_WWWEXAMPLECOM_AAA=111",
		"MICROBUS_EXAMPLECOM_AAA=XXX",    // Less specific key, should not override
		"MICROBUS_WWWANOTHERCOM_AAA=XXX", // Property of another service, should not override
		"WWWEXAMPLECOM_AAA=XXX",          // Must have microbus prefix
		"MICROBUS_EXAMPLECOM_BBB=222",
		"microbus_com_ccc=333", // Lowercase should work
		"MICROBUS_DDD=XXX",
		"MICROBUS=XXX",
	}

	configs := map[string]string{}
	readEnvars("www.example.com", environ, configs)
	assert.Len(t, configs, 3)
	assert.Equal(t, "111", configs["aaa"])
	assert.Equal(t, "222", configs["bbb"])
	assert.Equal(t, "333", configs["ccc"])
}

func TestReadEnvYaml(t *testing.T) {
	envYaml := []byte(`
# Comments should be ok
www.example.com:
  aaa: 111
  multiline: |-
    Line1
    Line2
example.com:
  aaa: xxx
  bbb: 222

com:
  ccc: 333

www.another.com:
  aaa: xxx
empty:
`)

	configs := map[string]string{}
	readEnvYamlFile("www.example.com", envYaml, configs)
	assert.Len(t, configs, 4)
	assert.Equal(t, "111", configs["aaa"])
	assert.Equal(t, "222", configs["bbb"])
	assert.Equal(t, "333", configs["ccc"])
	assert.Equal(t, "Line1\nLine2", configs["multiline"])
}
