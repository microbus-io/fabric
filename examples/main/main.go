package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/hello"
	"github.com/microbus-io/fabric/examples/messaging"
	"github.com/microbus-io/fabric/services/configurator"
	"github.com/microbus-io/fabric/services/httpingress"
)

/*
main runs the example microservices.
*/
func main() {
	app := application.New(
		configurator.NewService(),
		httpingress.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
	)
	app.Run()
}
