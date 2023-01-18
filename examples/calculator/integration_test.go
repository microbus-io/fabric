/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
		Arithmetic(ctx, x, op, y).
			Name(testName).
			Expect(t, xEcho, opEcho, yEcho, result).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, xEcho, opEcho, yEcho, result, err))
	*/
	ctx := Context(t)
	Arithmetic(ctx, 3, "-", 8).Name("subtraction").Expect(t, 3, "-", 8, -5)
	Arithmetic(ctx, -9, "+", 9).Name("addition").Expect(t, -9, "+", 9, 0)
	Arithmetic(ctx, -9, " ", 9).Name("space for addition").Expect(t, -9, "+", 9, 0)
	Arithmetic(ctx, 5, "*", 5).Name("multiplication").Expect(t, 5, "*", 5, 25)
	Arithmetic(ctx, 5, "*", -6).Name("multiplication negative").Expect(t, 5, "*", -6, -30)
	Arithmetic(ctx, 15, "/", 5).Name("division").Expect(t, 15, "/", 5, 3)
	Arithmetic(ctx, 15, "/", 0).Name("division by zero").Error(t, "zero")
	Arithmetic(ctx, 15, "z", 0).Name("invalid op").Error(t, "operator")
}

func TestCalculator_Square(t *testing.T) {
	t.Parallel()
	/*
		Square(ctx, x).
			Name(testName).
			Expect(t, xEcho, result).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, xEcho, result, err))
	*/
	ctx := Context(t)
	Square(ctx, 0).Name("zero").Expect(t, 0, 0)
	Square(ctx, 5).Name("positive").Expect(t, 5, 25)
	Square(ctx, -8).Name("negative").Expect(t, -8, 64)
}

func TestCalculator_Distance(t *testing.T) {
	t.Parallel()
	/*
		Distance(ctx, p1, p2).
			Name(testName).
			Expect(t, d).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, d, err))
	*/
	ctx := Context(t)
	Distance(ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 3, Y: 4}).
		Name("3-4-5 triangle").Expect(t, 5)
	Distance(ctx, calculatorapi.Point{X: -5, Y: -8}, calculatorapi.Point{X: 5, Y: -8}).
		Name("straight line").Expect(t, 10)
	Distance(ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 0, Y: 0}).
		Name("same point").Expect(t, 0)
}
