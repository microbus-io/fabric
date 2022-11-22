package calculator

import (
	"context"
	"math"
	"net/http"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
	"github.com/microbus-io/fabric/examples/calculator/intermediate"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ calculatorapi.Client
)

/*
Service implements the calculator.example microservice.

The Calculator microservice performs simple mathematical operations.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Arithmetic perform an arithmetic operation between two integers x and y given an operator op.
*/
func (svc *Service) Arithmetic(ctx context.Context, x int, op string, y int) (xEcho int, opEcho string, yEcho int, result int, err error) {
	if op == " " {
		op = "+" // + is interpreted as a space in URLs
	}
	// Perform the arithmetic operation
	switch op {
	case "*":
		result = x * y
	case "+":
		result = x + y
	case "-":
		result = x - y
	case "/":
		result = x / y
	default:
		return x, op, y, result, errors.Newf("invalid operator '%s'", op)
	}
	return x, op, y, result, nil
}

/*
Square prints the square of the integer x.
*/
func (svc *Service) Square(ctx context.Context, x int) (xEcho int, result int, err error) {
	return x, x * x, nil
}

/*
Distance calculates the distance between two points.
It demonstrates the use of the defined type Point.
*/
func (svc *Service) Distance(ctx context.Context, p1 calculatorapi.Point, p2 calculatorapi.Point) (d float64, err error) {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy), nil
}
