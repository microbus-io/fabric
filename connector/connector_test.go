package connector

import (
	"os"
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/stretchr/testify/assert"
)

func TestConnector_HostAndID(t *testing.T) {
	t.Parallel()

	c := NewConnector()
	assert.Empty(t, c.HostName())
	assert.NotEmpty(t, c.ID())
	c.SetHostName("example.com")
	assert.Equal(t, "example.com", c.HostName())
}

func TestConnector_BadHostName(t *testing.T) {
	t.Parallel()

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

func TestConnector_CatchPanic(t *testing.T) {
	t.Parallel()

	// String
	err := catchPanic(func() error {
		panic("message")
	})
	assert.Error(t, err)
	assert.Equal(t, "message", err.Error())

	// Error
	err = catchPanic(func() error {
		panic(errors.New("panic"))
	})
	assert.Error(t, err)
	assert.Equal(t, "panic", err.Error())

	// Number
	err = catchPanic(func() error {
		panic(5)
	})
	assert.Error(t, err)
	assert.Equal(t, "5", err.Error())

	// Division by zero
	err = catchPanic(func() error {
		j := 1
		j--
		i := 5 / j
		i++
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "runtime error: integer divide by zero", err.Error())

	// Nil map
	err = catchPanic(func() error {
		x := map[int]int{}
		if true {
			x = nil
		}
		x[5] = 6
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, "assignment to entry in nil map", err.Error())

	// Standard error
	err = catchPanic(func() error {
		return errors.New("standard")
	})
	assert.Error(t, err)
	assert.Equal(t, "standard", err.Error())
}

func TestConnector_Plane(t *testing.T) {
	t.Parallel()

	c := NewConnector()
	c.SetHostName("plane.connector")

	// Before starting
	assert.Empty(t, c.Plane())
	err := c.SetPlane("bad.plane.name")
	assert.Error(t, err)
	err = c.SetPlane("123plane456")
	assert.NoError(t, err)
	assert.Equal(t, "123plane456", c.Plane())
	err = c.SetPlane("")
	assert.NoError(t, err)
	assert.Equal(t, "", c.Plane())

	// Start connector
	err = c.Startup()
	assert.NoError(t, err)
	defer c.Shutdown()

	// After starting
	assert.NotEmpty(t, c.Plane())
	err = c.SetPlane("123plane456")
	assert.Error(t, err)
}

func TestConnector_PlaneEnv(t *testing.T) {
	// No parallel

	c := NewConnector()
	c.SetHostName("planeenv.connector")

	// Bad plane name
	defer os.Setenv("MICROBUS_ALL_PLANE", "")
	os.Setenv("MICROBUS_ALL_PLANE", "bad.plane.name")

	err := c.Startup()
	assert.Error(t, err)

	// Good plane name
	os.Setenv("MICROBUS_ALL_PLANE", "goodone")

	err = c.Startup()
	assert.NoError(t, err)
	defer c.Shutdown()

	assert.Equal(t, "goodone", c.Plane())
}

func TestConnector_Deployment(t *testing.T) {
	t.Parallel()

	c := NewConnector()
	c.SetHostName("deployment.connector")

	// Before starting
	assert.Empty(t, c.Deployment())
	err := c.SetDeployment("NOGOOD")
	assert.Error(t, err)
	err = c.SetDeployment("lAb")
	assert.NoError(t, err)
	assert.Equal(t, "LAB", c.Deployment())
	err = c.SetDeployment("")
	assert.NoError(t, err)
	assert.Equal(t, "", c.Deployment())

	// Start connector
	err = c.Startup()
	assert.NoError(t, err)
	defer c.Shutdown()

	// After starting
	assert.Equal(t, "LOCAL", c.Deployment())
	err = c.SetDeployment("LAB")
	assert.Error(t, err)
}

func TestConnector_DeploymentEnv(t *testing.T) {
	// No parallel

	c := NewConnector()
	c.SetHostName("deploymentenv.connector")

	// Bad plane name
	defer os.Setenv("MICROBUS_ALL_DEPLOYMENT", "")
	os.Setenv("MICROBUS_ALL_DEPLOYMENT", "lAb")

	err := c.Startup()
	assert.NoError(t, err)
	defer c.Shutdown()

	assert.Equal(t, "LAB", c.Deployment())
}
