package application

import "github.com/microbus-io/fabric/clock"

// Service is an interface abstraction of a microservice used by the application.
// The connector implements this interface.
type Service interface {
	SetPlane(plane string) error
	SetDeployment(deployment string) error
	SetClock(newClock clock.Clock) error
	Startup() (err error)
	Shutdown() (err error)
}
