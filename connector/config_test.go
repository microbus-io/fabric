package connector

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadEnvars(t *testing.T) {
	t.Parallel()

	environ := []string{
		"Path=...",
		"MICROBUS_WWWEXAMPLECOM_AAA=111",
		"MICROBUS_EXAMPLECOM_AAA=XXX",    // Less specific key, should not override
		"MICROBUS_WWWANOTHERCOM_AAA=XXX", // Property of another service, should not override
		"WWWEXAMPLECOM_AAA=XXX",          // Must have microbus prefix
		"MICROBUS_EXAMPLECOM_BBB=222",
		"microbus_com_ccc=333", // Lowercase should work
		"MICROBUS_all_DDD=444",
		"MICROBUS_XXX=XXX",
		"MICROBUS=XXX",
		"MICROBUS_ALL_OVERRIDE=0",
		"MICROBUS_COM_OVERRIDE=1",
		"MICROBUS_EXAMPLECOM_OVERRIDE=2",
	}

	configs := map[string]string{}
	readEnvars("www.example.com", environ, configs)
	assert.Len(t, configs, 5)
	assert.Equal(t, "111", configs["aaa"])
	assert.Equal(t, "222", configs["bbb"])
	assert.Equal(t, "333", configs["ccc"])
	assert.Equal(t, "444", configs["ddd"])
	assert.Equal(t, "2", configs["override"])
}

func TestReadEnvYaml(t *testing.T) {
	t.Parallel()

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
  override: 2

com:
  ccc: 333
  override: 1

www.another.com:
  aaa: xxx
empty:

all:
  ddd: 444
  override: 0
`)

	configs := map[string]string{}
	readEnvYamlFile("www.example.com", envYaml, configs)
	assert.Len(t, configs, 6)
	assert.Equal(t, "111", configs["aaa"])
	assert.Equal(t, "222", configs["bbb"])
	assert.Equal(t, "333", configs["ccc"])
	assert.Equal(t, "444", configs["ddd"])
	assert.Equal(t, "2", configs["override"])
	assert.Equal(t, "Line1\nLine2", configs["multiline"])
}

func TestSetConfig(t *testing.T) {
	t.Parallel()

	alpha := NewConnector()
	alpha.SetHostName("alpha.setconfig.connector")
	alpha.Startup()

	_, ok := alpha.Config("s")
	assert.False(t, ok)
	_, ok = alpha.Config("i")
	assert.False(t, ok)
	_, ok = alpha.Config("b")
	assert.False(t, ok)
	_, ok = alpha.Config("dur")
	assert.False(t, ok)

	alpha.SetConfig("s", "string")
	alpha.SetConfigInt("i", 123)
	alpha.SetConfigBool("b", true)
	alpha.SetConfigDuration("dur", time.Minute)

	s, ok := alpha.Config("s")
	if assert.True(t, ok) {
		assert.Equal(t, "string", s)
	}
	i, ok := alpha.ConfigInt("i")
	if assert.True(t, ok) {
		assert.Equal(t, 123, i)
	}
	b, ok := alpha.ConfigBool("b")
	if assert.True(t, ok) {
		assert.Equal(t, true, b)
	}
	dur, ok := alpha.ConfigDuration("dur")
	if assert.True(t, ok) {
		assert.Equal(t, time.Minute, dur)
	}

	alpha.Shutdown()
}

func TestEnvYaml(t *testing.T) {
	// No parallel

	// Create an env file
	_, err := os.Stat("env.yaml")
	if err == nil {
		// Do not overwrite if a file already exists
		t.Skip()
	}
	err = os.WriteFile("env.yaml", []byte(`
alpha.envyaml.connector:
  aaa: 111
envyaml.connector:
  aaa: 222
  bbb: 222
connector:
  aaa: 333
  bbb: 333
  ccc: 333
all:
  aaa: 444
  bbb: 444
  ccc: 444
  ddd: 444
`), 0666)
	assert.NoError(t, err)
	defer os.Remove("env.yaml")

	alpha := NewConnector()
	alpha.SetHostName("alpha.envyaml.connector")
	err = alpha.Startup()
	assert.NoError(t, err)

	v, ok := alpha.Config("aaa")
	if assert.True(t, ok) {
		assert.Equal(t, "111", v)
	}
	v, ok = alpha.Config("bbb")
	if assert.True(t, ok) {
		assert.Equal(t, "222", v)
	}
	v, ok = alpha.Config("ccc")
	if assert.True(t, ok) {
		assert.Equal(t, "333", v)
	}
	v, ok = alpha.Config("ddd")
	if assert.True(t, ok) {
		assert.Equal(t, "444", v)
	}

	alpha.Shutdown()
}

func TestEnviron(t *testing.T) {
	// No parallel

	os.Setenv("MICROBUS_ALPHAENVIRONCONNECTOR_AAA", "111")
	os.Setenv("MICROBUS_ENVIRONCONNECTOR_AAA", "222")
	os.Setenv("MICROBUS_ENVIRONCONNECTOR_BBB", "222")
	os.Setenv("MICROBUS_CONNECTOR_AAA", "333")
	os.Setenv("MICROBUS_CONNECTOR_BBB", "333")
	os.Setenv("MICROBUS_CONNECTOR_CCC", "333")
	os.Setenv("MICROBUS_ALL_AAA", "444")
	os.Setenv("MICROBUS_ALL_BBB", "444")
	os.Setenv("MICROBUS_ALL_CCC", "444")
	os.Setenv("MICROBUS_ALL_DDD", "444")

	defer func() {
		os.Setenv("MICROBUS_ALPHAENVIRONCONNECTOR_AAA", "")
		os.Setenv("MICROBUS_ENVIRONCONNECTOR_AAA", "")
		os.Setenv("MICROBUS_ENVIRONCONNECTOR_BBB", "")
		os.Setenv("MICROBUS_CONNECTOR_AAA", "")
		os.Setenv("MICROBUS_CONNECTOR_BBB", "")
		os.Setenv("MICROBUS_CONNECTOR_CCC", "")
		os.Setenv("MICROBUS_ALL_AAA", "")
		os.Setenv("MICROBUS_ALL_BBB", "")
		os.Setenv("MICROBUS_ALL_CCC", "")
		os.Setenv("MICROBUS_ALL_DDD", "")
	}()

	alpha := NewConnector()
	alpha.SetHostName("alpha.environ.connector")
	err := alpha.Startup()
	assert.NoError(t, err)

	v, ok := alpha.Config("aaa")
	if assert.True(t, ok) {
		assert.Equal(t, "111", v)
	}
	v, ok = alpha.Config("bbb")
	if assert.True(t, ok) {
		assert.Equal(t, "222", v)
	}
	v, ok = alpha.Config("ccc")
	if assert.True(t, ok) {
		assert.Equal(t, "333", v)
	}
	v, ok = alpha.Config("ddd")
	if assert.True(t, ok) {
		assert.Equal(t, "444", v)
	}

	alpha.Shutdown()
}
