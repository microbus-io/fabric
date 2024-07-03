# Explore on Your Own

* Start NATS in debug and verbose modes `./nats-server -D -V`, run unit tests individually and look at the messages going over the bus
* Modify `main/config.yaml` and witness the impact on the `hello.example` microservice. You'll have to restart the app for the configurator to pick up the new values
* Add an endpoint `/increment` to the `calculator.example` microservice that returns the value of an input integer x plus 1
* Add an endpoint `/calculate` to the `calculator.example` microservice that operates on decimal numbers, not just integers. Can you make `hello.example/calculator` work with decimals too?
* Create your own microservice from scratch and add it to `main/main.go`
* Put a breakpoint in any of the microservices of the example application and try debugging
* Add a `/cache-delete` endpoint to the `messaging.example`
* Create a second event sink microservice (name it differently though) and block registrations based on a configurable exclusion list
