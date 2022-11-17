// Code generated by Microbus. DO NOT EDIT.

/*
Package control implements the control.sys microservice.

This microservice is created for the sake of generating the client API for the :888 control subscriptions.
The microservice itself does nothing and should not be included in applications.
*/
package control

import (
	"github.com/microbus-io/fabric/services/control/intermediate"
	"github.com/microbus-io/fabric/services/control/controlapi"
)

var (
	_ controlapi.Client
)

// The default host name of the microservice is control.sys.
const HostName = "control.sys"

// NewService creates a new control.sys microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}
