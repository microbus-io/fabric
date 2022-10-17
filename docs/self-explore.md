# Explore on Your Own

* Start NATS in debug mode `./nats-server -D -V`, run unit tests individually and look at the messages going over the bus
* Modify `examples/main/env.yaml` and witness the impact on the `hello.example` microservice
* Add an endpoint `/increment` to the `calculator.example` microservice that returns the value of an input integer x plus 1
* Add an endpoint `/calculate` to the `calculator.example` microservice that operates on decimal numbers, not just integers
* Create your own microservice from scratch and add it to `examples/main/main.go`
* Put a breakpoint in any of the microservices of the example application and try debugging
* Write unit tests that require mocking the microservice's clock
