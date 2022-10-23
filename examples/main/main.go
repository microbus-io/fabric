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

Try the following URLs:

	http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
	http://localhost:8080/calculator.example/square?x=5
	http://localhost:8080/calculator.example/square?x=not-a-number
	http://localhost:8080/hello.example/echo
	http://localhost:8080/hello.example/ping
	http://localhost:8080/hello.example/hello?name=Bella
	http://localhost:8080/hello.example/calculator
	http://localhost:8080/messaging.example/home
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
