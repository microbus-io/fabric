// Code generated by Microbus. DO NOT EDIT.

package main

import (
	"fmt"
	"os"

	"github.com/microbus-io/fabric/application"

	"github.com/microbus-io/fabric/services/metrics"
)

// main runs an app containing only the metrics.sys service.
func main() {
	app := application.New(
		metrics.NewService(),
	)
	err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(19)
	}
}
