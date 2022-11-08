// Code generated by Microbus. DO NOT EDIT.

/*
Package httpingress implements the http.ingress.sys microservice.

The HTTP Ingress microservice relays incoming HTTP requests to the NATS bus.
*/
package httpingress

import (
	"github.com/microbus-io/fabric/services/httpingress/intermediate"

	"github.com/microbus-io/fabric/services/httpingress/httpingressapi"
)

var (
	_ httpingressapi.Client
)

const ServiceName = "http.ingress.sys"

// NewService creates a new http.ingress.sys microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}

type Initializer = intermediate.Initializer

// Config initializers
var (
	// TimeBudget initializes the TimeBudget config property of the microservice
	TimeBudget = intermediate.TimeBudget
	// Port initializes the Port config property of the microservice
	Port = intermediate.Port
)

/*
With initializes the config properties of the microservice for testings purposes.

	httpingressSvc := httpingress.NewService().With(...)
*/
func (svc *Service) With(initializers ...Initializer) *Service {
	svc.Intermediate.With(initializers...)
	return svc
}
