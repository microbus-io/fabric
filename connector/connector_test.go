/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"testing"

	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/rand"
	"github.com/stretchr/testify/assert"
)

var (
	randomPlane = rand.AlphaNum64(12)
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
		"my!example.com",
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

	// Before starting
	con := New("plane.connector")
	assert.Empty(t, con.Plane())
	err := con.SetPlane("bad.plane.name")
	assert.Error(t, err)
	err = con.SetPlane(randomPlane)
	assert.NoError(t, err)
	assert.Equal(t, randomPlane, con.Plane())
	err = con.SetPlane("")
	assert.NoError(t, err)
	assert.Equal(t, "", con.Plane())

	// After starting
	con = New("plane.connector")
	con.SetPlane(randomPlane)
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()
	assert.NotEmpty(t, con.Plane())
	err = con.SetPlane("123plane456")
	assert.Error(t, err)
}

func TestConnector_PlaneEnv(t *testing.T) {
	// No parallel

	con := New("plane.env.connector")

	// Bad plane name
	env.Push("MICROBUS_PLANE", "bad.plane.name")
	defer env.Pop("MICROBUS_PLANE")

	err := con.Startup()
	assert.Error(t, err)

	// Good plane name
	env.Push("MICROBUS_PLANE", randomPlane)
	defer env.Pop("MICROBUS_PLANE")

	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, randomPlane, con.Plane())
}

func TestConnector_Deployment(t *testing.T) {
	t.Parallel()

	// Before starting
	con := New("deployment.connector")
	con.SetPlane(randomPlane)
	assert.Empty(t, con.Deployment())
	err := con.SetDeployment("NOGOOD")
	assert.Error(t, err)
	err = con.SetDeployment("lAb")
	assert.NoError(t, err)
	assert.Equal(t, LAB, con.Deployment())
	err = con.SetDeployment("")
	assert.NoError(t, err)
	assert.Equal(t, "", con.Deployment())

	// After starting
	con = New("deployment.connector")
	con.SetPlane(randomPlane)
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()
	assert.Equal(t, TESTINGAPP, con.Deployment())
	err = con.SetDeployment(LAB)
	assert.Error(t, err)
}

func TestConnector_DeploymentEnv(t *testing.T) {
	// No parallel

	con := New("deployment.env.connector")
	con.SetPlane(randomPlane)

	env.Push("MICROBUS_DEPLOYMENT", "lAb")
	defer env.Pop("MICROBUS_DEPLOYMENT")

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Equal(t, LAB, con.Deployment())
}

func TestConnector_Version(t *testing.T) {
	t.Parallel()

	// Before starting
	con := New("version.connector")
	con.SetPlane(randomPlane)
	assert.Equal(t, randomPlane, con.Plane())
	err := con.SetVersion(-1)
	assert.Error(t, err)
	err = con.SetVersion(123)
	assert.NoError(t, err)
	assert.Equal(t, 123, con.Version())
	err = con.SetVersion(0)
	assert.NoError(t, err)
	assert.Equal(t, 0, con.Version())

	// After starting
	con = New("version.connector")
	con.SetPlane(randomPlane)
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()
	err = con.SetVersion(123)
	assert.Error(t, err)
}
