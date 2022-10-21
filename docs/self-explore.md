# Explore on Your Own

* Start NATS in debug mode `./nats-server -D -V`, run unit tests individually and look at the messages going over the bus
* Modify `examples/main/config.yaml` and witness the impact on the `hello.example` microservice. You'll have to either restart the app or call `https://configurator:8080/refresh` to push the new config values
* Add an endpoint `/increment` to the `calculator.example` microservice that returns the value of an input integer x plus 1
* Add an endpoint `/calculate` to the `calculator.example` microservice that operates on decimal numbers, not just integers. Can you make `hello.example/calculator` work with decimals too?
* Create your own microservice from scratch and add it to `examples/main/main.go`
* Put a breakpoint in any of the microservices of the example application and try debugging
* Write unit tests that require mocking the microservice's clock
