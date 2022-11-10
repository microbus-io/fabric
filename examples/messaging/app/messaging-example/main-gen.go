// Code generated by Microbus. DO NOT EDIT.

package main

import (
	"fmt"
	"os"

	"github.com/microbus-io/fabric/application"

	"github.com/microbus-io/fabric/examples/messaging"
)

// main runs an app containing only the messaging.example service.
func main() {
	app := application.New(
		messaging.NewService(),
	)
	err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(-1)
	}
}
