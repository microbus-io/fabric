# Graceful Shutdown

Microservices typically run on hardware that can shut down at any time. When a microservice receives the `SIGTERM` command to shut down, it stops accepting new operations and attempts to end all pending operations gracefully.

Graceful shutdown process:
* Disable tickers so new iterations are not run
* Stop accepting new requests
* Wait 8 seconds for running tickers, pending requests and goroutines to end naturally
* Cancel the lifetime `context.Context`
* Wait 4 more seconds for running tickers and pending requests, and goroutines to end
* Close the NATS connection
* Exit

In order for goroutines to be gracefully shut down, it is important to launch them using the `Connector`'s `Go` method rather than using the standard `go` directive. When launched with `Go`, the goroutines have the `context.Context` that gets canceled during shut down.
