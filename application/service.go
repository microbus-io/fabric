/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package application

// Service is an interface abstraction of a microservice used by the application.
// The connector implements this interface.
type ServiceX interface {
	SetPlane(plane string) error
	SetDeployment(deployment string) error
	Startup() (err error)
	Shutdown() (err error)
	IsStarted() bool
	HostName() string
}
