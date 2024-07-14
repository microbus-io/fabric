/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/examples/calculator/calculatorapi"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *calculatorapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc,
	)
	if err != nil {
		return err
	}
	return nil
}

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
	return nil
}

func TestCalculator_Arithmetic(t *testing.T) {
	t.Parallel()
	/*
		Arithmetic(t, ctx, x, op, y).
			Expect(xEcho, opEcho, yEcho, result).
			NoError()
	*/
	ctx := Context()
	Arithmetic(t, ctx, 3, "-", 8).Expect(3, "-", 8, -5)
	Arithmetic(t, ctx, -9, "+", 9).Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, -9, " ", 9).Expect(-9, "+", 9, 0)
	Arithmetic(t, ctx, 5, "*", 5).Expect(5, "*", 5, 25)
	Arithmetic(t, ctx, 5, "*", -6).Expect(5, "*", -6, -30)
	Arithmetic(t, ctx, 15, "/", 5).Expect(15, "/", 5, 3)
	Arithmetic(t, ctx, 15, "/", 0).Error("zero")
	Arithmetic(t, ctx, 15, "z", 0).Error("operator")
}

func TestCalculator_Square(t *testing.T) {
	t.Parallel()
	/*
		Square(t, ctx, x).
			Expect(xEcho, result).
			NoError()
	*/
	ctx := Context()
	Square(t, ctx, 0).Expect(0, 0)
	Square(t, ctx, 5).Expect(5, 25)
	Square(t, ctx, -8).Expect(-8, 64)
}

func TestCalculator_Distance(t *testing.T) {
	t.Parallel()
	/*
		Distance(t, ctx, p1, p2).
			Expect(td).
			NoError()
	*/
	ctx := Context()
	Distance(t, ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 3, Y: 4}).Expect(5)
	Distance(t, ctx, calculatorapi.Point{X: -5, Y: -8}, calculatorapi.Point{X: 5, Y: -8}).Expect(10)
	Distance(t, ctx, calculatorapi.Point{X: 0, Y: 0}, calculatorapi.Point{X: 0, Y: 0}).Expect(0)
}
