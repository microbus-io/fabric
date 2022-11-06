// Code generated by Microbus. DO NOT EDIT.

package configurator

import (
	"github.com/microbus-io/fabric/services/configurator/intermediate"

	"github.com/microbus-io/fabric/services/configurator/configuratorapi"
)

var (
	_ configuratorapi.Client
)

const ServiceName = "configurator.sys"

// NewService creates a new configurator.sys microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}
