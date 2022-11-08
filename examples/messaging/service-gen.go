// Code generated by Microbus. DO NOT EDIT.

/*
Package messaging implements the messaging.example microservice.

The Messaging microservice demonstrates service-to-service communication patterns.
*/
package messaging

import (
	"github.com/microbus-io/fabric/examples/messaging/intermediate"

	"github.com/microbus-io/fabric/examples/messaging/messagingapi"
)

var (
	_ messagingapi.Client
)

const ServiceName = "messaging.example"

// NewService creates a new messaging.example microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}
