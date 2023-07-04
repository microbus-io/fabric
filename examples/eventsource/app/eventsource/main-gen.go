/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Code generated by Microbus. DO NOT EDIT.

package main

import (
	"fmt"
	"os"

	"github.com/microbus-io/fabric/application"

	"github.com/microbus-io/fabric/examples/eventsource"
)

// main runs an app containing only the eventsource.example service.
func main() {
	app := application.New(
		eventsource.NewService(),
	)
	err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(19)
	}
}
