// Code generated by Microbus. DO NOT EDIT.

package hello

import "github.com/microbus-io/fabric/examples/hello/intermediate"

// NewService creates a new "hello.example" microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}

// Config initializers
var (
	// Greeting initializes the "Greeting" config property of the microservice.
	Greeting = intermediate.Greeting
	// Repeat initializes the "Repeat" config property of the microservice.
	Repeat = intermediate.Repeat
)

/*
With initializes the config properties of the microservice for testings purposes.

	helloSvc := hello.NewService().With(...)
*/
func (svc *Service) With(initializers ...intermediate.Initializer) *Service {
	svc.Intermediate.With(initializers...)
	return svc
}