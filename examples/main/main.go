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

// main runs the example microservices
func main() {
	// Create and startup the microservices
	ingressSvc := httpingress.NewService()
	echoSvc := echo.NewService()
	helloWorldSvc := helloworld.NewService()
	calculatorSvc := calculator.NewService()

	ingressSvc.Startup()
	echoSvc.Startup()
	helloWorldSvc.Startup()
	calculatorSvc.Startup()
	defer func() {
		calculatorSvc.Shutdown()
		helloWorldSvc.Shutdown()
		echoSvc.Shutdown()
		ingressSvc.Shutdown()
	}()

	// Wait for ctrl-C interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
