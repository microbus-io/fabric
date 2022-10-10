package connector

import (
	"context"

	"github.com/microbus-io/fabric/log"
)

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

// Logger is the interface that the connector supports for logging
type Logger interface {
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogWarn(ctx context.Context, msg string, fields ...log.Field)
	LogError(ctx context.Context, msg string, fields ...log.Field)
}
