// Code generated by Microbus. DO NOT EDIT.

/*
Package eventsource implements the eventsource.example microservice.

The event source microservice fires events that are caught by the event sink microservice.
*/
package eventsource

import (
	"github.com/microbus-io/fabric/examples/eventsource/intermediate"

	"github.com/microbus-io/fabric/examples/eventsource/eventsourceapi"
)

var (
	_ eventsourceapi.Client
)

// The default host name of the microservice is eventsource.example.
const HostName = "eventsource.example"

// NewService creates a new eventsource.example microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}
