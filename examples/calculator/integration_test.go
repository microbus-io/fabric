package calculator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *calculatorapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(options) to initialize with appropriate config values
	App.Include(
		Svc,
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestCalculator_Arithmetic(t *testing.T) {
	t.Parallel()
	/*
		Arithmetic(ctx, x, op, y).
			Expect(t, xEcho, opEcho, yEcho, result).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, xEcho, opEcho, yEcho, result, err))
	*/
	ctx := Context()
	Arithmetic(ctx, 3, "-", 8).Expect(t, 3, "-", 8, -5)
	Arithmetic(ctx, -9, "+", 9).Expect(t, -9, "+", 9, 0)
	Arithmetic(ctx, -9, " ", 9).Expect(t, -9, "+", 9, 0)
	Arithmetic(ctx, 5, "*", 5).Expect(t, 5, "*", 5, 25)
	Arithmetic(ctx, 5, "*", -6).Expect(t, 5, "*", -6, -30)
	Arithmetic(ctx, 15, "/", 5).Expect(t, 15, "/", 5, 3)
	Arithmetic(ctx, 15, "/", 0).Error(t, "zero")
	Arithmetic(ctx, 15, "z", 0).Error(t, "operator")
}

func TestCalculator_Square(t *testing.T) {
	t.Parallel()
	/*
		Square(ctx, x).
			Expect(t, xEcho, result).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, xEcho, result, err))
	*/
	ctx := Context()
	Square(ctx, 0).Expect(t, 0, 0)
	Square(ctx, 5).Expect(t, 5, 25)
	Square(ctx, -8).Expect(t, -8, 64)
}

func TestCalculator_Distance(t *testing.T) {
	t.Parallel()
	/*
		Distance(ctx, p1, p2).
			Expect(t, d).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, d, err))
	*/
	ctx := Context()
	Distance(ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 3, Y: 4}).Expect(t, 5)
	Distance(ctx, calculatorapi.Point{X: -5, Y: -8}, calculatorapi.Point{X: 5, Y: -8}).Expect(t, 10)
	Distance(ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 0, Y: 0}).Expect(t, 0)
}
