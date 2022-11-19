package connector

import "context"

// Service is the interface of the connector that is exposed to applications.
// It includes the initialization and lifecycle methods of the connector.
type Service interface {
	InitConfig(name string, value string) error
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
}
