/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package calculator

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
	"github.com/microbus-io/fabric/pub"
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
		Arithmetic(t, ctx, x, op, y).
			Name(testName).
			Expect(xEcho, opEcho, yEcho, result).
			NoError().
			Error(errContains).
			Assert(func(t, xEcho, opEcho, yEcho, result, err))
	*/
	ctx := Context(t)
	Arithmetic(t, ctx, 3, "-", 8).Name("subtraction").Expect(3, "-", 8, -5)
	Arithmetic(t, ctx, -9, "+", 9).Name("addition").Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, -9, " ", 9).Name("space for addition").Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, 5, "*", 5).Name("multiplication").Expect(5, "*", 5, 25)
	Arithmetic(t, ctx, 5, "*", -6).Name("multiplication negative").Expect(5, "*", -6, -30)
	Arithmetic(t, ctx, 15, "/", 5).Name("division").Expect(15, "/", 5, 3)
	Arithmetic(t, ctx, 15, "/", 0).Name("division by zero").Error("zero")
	Arithmetic(t, ctx, 15, "z", 0).Name("invalid op").Error("operator")
}

func TestCalculator_Square(t *testing.T) {
	t.Parallel()
	/*
		Square(t, ctx, x).
			Name(testName).
			Expect( xEcho, result).
			NoError().
			Error(errContains).
			Assert(func(t, xEcho, result, err))
	*/
	ctx := Context(t)
	Square(t, ctx, 0).Name("zero").Expect(0, 0)
	Square(t, ctx, 5).Name("positive").Expect(5, 25)
	Square(t, ctx, -8).Name("negative").Expect(-8, 64)
}

func TestCalculator_Distance(t *testing.T) {
	t.Parallel()
	/*
		Distance(t, ctx, p1, p2).
			Name(testName).
			Expect(td).
			NoError().
			Error(errContains).
			Assert(func(t, d, err))
	*/
	ctx := Context(t)
	Distance(t, ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 3, Y: 4}).
		Name("3-4-5 triangle").Expect(5)
	Distance(t, ctx, calculatorapi.Point{X: -5, Y: -8}, calculatorapi.Point{X: 5, Y: -8}).
		Name("straight line").Expect(10)
	Distance(t, ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 0, Y: 0}).
		Name("same point").Expect(0)
}

func TestCalculator_OpenAPI(t *testing.T) {
	ctx := Context(t)
	res, err := Svc.Request(ctx, pub.GET("https://"+Svc.HostName()+"/openapi.json"))
	if assert.NoError(t, err) {
		body, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			fns := []string{
				"443", "Arithmetic",
				"443", "Square",
				"443", "Distance",
			}
			for i := 0; i < len(fns); i += 2 {
				if assert.NoError(t, err) {
					res.Body.Close()
					assert.Contains(t, string(body), `"summary": "`+fns[i+1]+"(")
					assert.Contains(t, string(body), fns[i+1]+"_IN")
					assert.Contains(t, string(body), fns[i+1]+"_OUT")
				}
			}

			assert.Contains(t, string(body), `"Distance_IN_Point":`)
			assert.Contains(t, string(body), `"p1":`)
			assert.Contains(t, string(body), `"p2":`)
			assert.Contains(t, string(body), `"x":`)
			assert.Contains(t, string(body), `"y":`)
		}
	}
}
