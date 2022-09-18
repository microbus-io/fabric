package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/echo"
	"github.com/microbus-io/fabric/examples/helloworld"
	"github.com/microbus-io/fabric/services/httpingress"
)

/*
main runs the example microservices.

Try the following links:

	http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
	http://localhost:8080/calculator.example/square?x=5
	http://localhost:8080/echo.example/echo
	http://localhost:8080/echo.example/who
	http://localhost:8080/helloworld.example/hello?name=Gopher
*/
func main() {
	// Create and startup the microservices
	ingressSvc := httpingress.NewService()
	echo1Svc := echo.NewService()
	echo2Svc := echo.NewService()
	helloWorldSvc := helloworld.NewService()
	calculatorSvc := calculator.NewService()

	ingressSvc.Startup()
	defer calculatorSvc.Shutdown()
	echo1Svc.Startup()
	defer echo1Svc.Shutdown()
	echo2Svc.Startup()
	defer echo2Svc.Shutdown()
	helloWorldSvc.Startup()
	defer helloWorldSvc.Shutdown()
	calculatorSvc.Startup()
	defer calculatorSvc.Shutdown()

	// Wait for ctrl-C interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
