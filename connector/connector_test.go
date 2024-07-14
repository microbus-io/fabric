/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"regexp"
	"testing"

	"github.com/microbus-io/fabric/env"
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/testarossa"
)

func TestConnector_HostAndID(t *testing.T) {
	t.Parallel()

	con := NewConnector()
	testarossa.Equal(t, "", con.Hostname())
	testarossa.NotEqual(t, "", con.ID())
	con.SetHostname("example.com")
	testarossa.Equal(t, "example.com", con.Hostname())
}

func TestConnector_BadHostname(t *testing.T) {
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
		err := con.SetHostname(s)
		testarossa.Error(t, err)
	}
}

func TestConnector_Plane(t *testing.T) {
	t.Parallel()

	randomPlane := rand.AlphaNum64(12)

	// Before starting
	con := New("plane.connector")
	testarossa.Equal(t, "", con.Plane())
	err := con.SetPlane("bad.plane.name")
	testarossa.Error(t, err)
	err = con.SetPlane(randomPlane)
	testarossa.NoError(t, err)
	testarossa.Equal(t, randomPlane, con.Plane())
	err = con.SetPlane("")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "", con.Plane())

	// After starting
	con = New("plane.connector")
	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()
	testarossa.NotEqual(t, "", con.Plane())
	testarossa.NotEqual(t, "microbus", con.Plane())
	testarossa.True(t, regexp.MustCompile(`\w{12,}`).MatchString(con.Plane())) // Hash of test name
	err = con.SetPlane("123plane456")
	testarossa.Error(t, err)
}

func TestConnector_PlaneEnv(t *testing.T) {
	// No parallel

	// Bad plane name
	env.Push("MICROBUS_PLANE", "bad.plane.name")
	defer env.Pop("MICROBUS_PLANE")

	con := New("plane.env.connector")
	err := con.Startup()
	testarossa.Error(t, err)

	// Good plane name
	randomPlane := rand.AlphaNum64(12)
	env.Push("MICROBUS_PLANE", randomPlane)
	defer env.Pop("MICROBUS_PLANE")

	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Equal(t, randomPlane, con.Plane())
}

func TestConnector_Deployment(t *testing.T) {
	t.Parallel()

	// Before starting
	con := New("deployment.connector")
	testarossa.Equal(t, "", con.Deployment())
	err := con.SetDeployment("NOGOOD")
	testarossa.Error(t, err)
	err = con.SetDeployment("lAb")
	testarossa.NoError(t, err)
	testarossa.Equal(t, LAB, con.Deployment())
	err = con.SetDeployment("")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "", con.Deployment())

	// After starting
	con = New("deployment.connector")
	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()
	testarossa.Equal(t, TESTING, con.Deployment())
	err = con.SetDeployment(LAB)
	testarossa.Error(t, err)
}

func TestConnector_DeploymentEnv(t *testing.T) {
	// No parallel

	con := New("deployment.env.connector")

	env.Push("MICROBUS_DEPLOYMENT", "lAb")
	defer env.Pop("MICROBUS_DEPLOYMENT")

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	testarossa.Equal(t, LAB, con.Deployment())
}

func TestConnector_Version(t *testing.T) {
	t.Parallel()

	// Before starting
	con := New("version.connector")
	err := con.SetVersion(-1)
	testarossa.Error(t, err)
	err = con.SetVersion(123)
	testarossa.NoError(t, err)
	testarossa.Equal(t, 123, con.Version())
	err = con.SetVersion(0)
	testarossa.NoError(t, err)
	testarossa.Zero(t, con.Version())

	// After starting
	con = New("version.connector")
	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()
	err = con.SetVersion(123)
	testarossa.Error(t, err)
}
