package application

// Service is an interface abstraction of a microservice used by the application.
// The connector implements this interface.
type Service interface {
	SetPlane(plane string) error
	SetDeployment(deployment string) error
	Startup() (err error)
	Shutdown() (err error)
	IsStarted() bool
	HostName() string
}
