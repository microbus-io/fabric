package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/echo"
	"github.com/microbus-io/fabric/examples/helloworld"
	"github.com/microbus-io/fabric/services/httpingress"
)

/*
main runs the example microservices.

Try the following URLs:

	http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
	http://localhost:8080/calculator.example/square?x=5
	http://localhost:8080/calculator.example/square?x=not-a-number
	http://localhost:8080/echo.example/echo
	http://localhost:8080/echo.example/who
	http://localhost:8080/echo.example/ping
	http://localhost:8080/helloworld.example/hello?name=Gopher
*/
func main() {
	app := application.New(
		httpingress.NewService(),
		echo.NewService(),
		echo.NewService(),
		helloworld.NewService(),
		calculator.NewService(),
	)
	app.Run()
}
