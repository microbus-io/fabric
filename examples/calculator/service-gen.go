// Code generated by Microbus. DO NOT EDIT.

package calculator

import (
	"github.com/microbus-io/fabric/examples/calculator/intermediate"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
)

var (
	_ calculatorapi.Client
)

// NewService creates a new calculator.example microservice.
func NewService() *Service {
	s := &Service{}
	s.Intermediate = intermediate.New(s, Version)
	return s
}
