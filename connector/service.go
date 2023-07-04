/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import "context"

// Service is the interface of the connector that is exposed to applications.
// It includes the initialization and lifecycle methods of the connector.
type Service interface {
	SetConfig(name string, value any) error
	ResetConfig(name string) error
	ID() string
	SetHostName(hostName string) error
	HostName() string
	Description() string
	Version() int
	Deployment() string
	SetDeployment(deployment string) error
	Plane() string
	SetPlane(plane string) error
	Startup() (err error)
	Shutdown() error
	IsStarted() bool
	Lifetime() context.Context
	With(options ...func(Service) error) Service
	StartupSequence() int
	SetStartupSequence(seq int)
}
