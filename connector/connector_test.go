package connector

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_HostAndID(t *testing.T) {
	t.Parallel()

	con := NewConnector()
	assert.Empty(t, con.HostName())
	assert.NotEmpty(t, con.ID())
	con.SetHostName("example.com")
	assert.Equal(t, "example.com", con.HostName())
}

func TestConnector_BadHostName(t *testing.T) {
	t.Parallel()

	con := NewConnector()
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
		err := con.SetHostName(s)
		assert.Error(t, err)
	}
}

func TestConnector_Plane(t *testing.T) {
	t.Parallel()

	con := NewConnector()
	con.SetHostName("plane.connector")

	// Before starting
	assert.Empty(t, con.Plane())
	err := con.SetPlane("bad.plane.name")
	assert.Error(t, err)
	err = con.SetPlane("123plane456")
	assert.NoError(t, err)
	assert.Equal(t, "123plane456", con.Plane())
	err = con.SetPlane("")
	assert.NoError(t, err)
	assert.Equal(t, "", con.Plane())

	// Start connector
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// After starting
	assert.NotEmpty(t, con.Plane())
	err = con.SetPlane("123plane456")
	assert.Error(t, err)
}

func TestConnector_PlaneEnv(t *testing.T) {
	// No parallel

	con := NewConnector()
	con.SetHostName("planeenv.connector")

	// Bad plane name
	defer os.Setenv("MICROBUS_PLANE", "")
	os.Setenv("MICROBUS_PLANE", "bad.plane.name")

	err := con.Startup()
	assert.Error(t, err)

	// Good plane name
	os.Setenv("MICROBUS_PLANE", "goodone")

	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, "goodone", con.Plane())
}

func TestConnector_Deployment(t *testing.T) {
	t.Parallel()

	con := NewConnector()
	con.SetHostName("deployment.connector")

	// Before starting
	assert.Empty(t, con.Deployment())
	err := con.SetDeployment("NOGOOD")
	assert.Error(t, err)
	err = con.SetDeployment("lAb")
	assert.NoError(t, err)
	assert.Equal(t, LAB, con.Deployment())
	err = con.SetDeployment("")
	assert.NoError(t, err)
	assert.Equal(t, "", con.Deployment())

	// Start connector
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// After starting
	assert.Equal(t, LOCAL, con.Deployment())
	err = con.SetDeployment(LAB)
	assert.Error(t, err)
}

func TestConnector_DeploymentEnv(t *testing.T) {
	// No parallel

	con := NewConnector()
	con.SetHostName("deploymentenv.connector")

	// Bad plane name
	defer os.Setenv("MICROBUS_DEPLOYMENT", "")
	os.Setenv("MICROBUS_DEPLOYMENT", "lAb")

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, LAB, con.Deployment())
}
