/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/directory"
	"github.com/microbus-io/fabric/examples/eventsink"
	"github.com/microbus-io/fabric/examples/eventsource"
	"github.com/microbus-io/fabric/examples/hello"
	"github.com/microbus-io/fabric/examples/messaging"
	"github.com/microbus-io/fabric/services/configurator"
	"github.com/microbus-io/fabric/services/httpingress"
	"github.com/microbus-io/fabric/services/metrics"
)

/*
main runs the example microservices.
*/
func main() {
	app := application.New(
		configurator.NewService(),
		httpingress.NewService(),
		metrics.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
		eventsource.NewService(),
		eventsink.NewService(),
		directory.NewService(),
	)
	app.Run()
}
