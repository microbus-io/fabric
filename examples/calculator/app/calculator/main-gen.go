// Code generated by Microbus. DO NOT EDIT.

package main

import (
	"fmt"
	"os"

	"github.com/microbus-io/fabric/application"

	"github.com/microbus-io/fabric/examples/calculator"
)

// main runs an app containing only the calculator.example service.
func main() {
	app := application.New(
		calculator.NewService(),
	)
	err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(19)
	}
}