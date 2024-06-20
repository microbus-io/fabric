/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/coreservices/configurator"
	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/coreservices/httpingress"
	"github.com/microbus-io/fabric/coreservices/metrics"
	"github.com/microbus-io/fabric/coreservices/openapiportal"
	"github.com/microbus-io/fabric/examples/browser"
	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/directory"
	"github.com/microbus-io/fabric/examples/eventsink"
	"github.com/microbus-io/fabric/examples/eventsource"
	"github.com/microbus-io/fabric/examples/hello"
	"github.com/microbus-io/fabric/examples/helloworld"
	"github.com/microbus-io/fabric/examples/messaging"
)

/*
main runs the example microservices.
*/
func main() {
	app := application.New()
	app.Include(
		// Configurator should start first
		configurator.NewService(),
	)
	app.Include(
		httpegress.NewService(),
		openapiportal.NewService(),
		metrics.NewService(),
		// inbox.NewService(),

		helloworld.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
		eventsource.NewService(),
		eventsink.NewService(),
		directory.NewService(),
		browser.NewService(),
	)
	app.Include(
		// When everything is ready, begin to accept external requests
		httpingress.NewService(),
	)
	app.Run()
}
