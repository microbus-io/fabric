# Graceful Shutdown

Microservices are designed to run on cloud hardware that often shuts down at unpredictable times. When a microservice receives the signal to terminate, it first stops accepting new operations and attempts to end all pending operations gracefully.

Graceful shutdown process:
* Disable tickers so new iterations do not run
* Stop accepting new requests
* Wait 8 seconds for running tickers, pending requests and goroutines to end naturally
* Cancel the lifetime `context.Context` of the microservice
* Wait 4 more seconds for running tickers and pending requests, and goroutines to quit
* Close the connection to the bus
* Exit

In order for goroutines to gracefully shut down, it is important to launch them using the `Connector`'s `Go` method rather than using the standard `go` keyword. When launched with `Go`, the goroutines are given the `context.Context` that gets canceled during termination.
