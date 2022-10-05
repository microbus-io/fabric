package connector

// Service is the interface that all microservices must support.
// It is implemented by the connector
type Service interface {
	Startup() error
	Shutdown() error
	IsStarted() bool

	HostName() string
	SetHostName(hostName string) error
	Deployment() string
	SetDeployment(deployment string) error
	Plane() string
	SetPlane(plane string) error
}
