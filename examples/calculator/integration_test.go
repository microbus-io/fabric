/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
