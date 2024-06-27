# Local Development

In `Microbus`, microservices are not large memory-gobbling processes but rather compact worker goroutines that ultimately consume messages from a queue. Microservices also don’t listen on ports so there is no potential of a port conflict with other microservices. These two qualities allow `Microbus` to spin up a large multitude of microservices in a single executable, enabling engineers to run and debug an entire application in their IDE. In fact, starting or restarting an application takes only a second or two, allowing engineers to iterate on code change quickly. It is a huge productivity boost.

Those same qualities also allow `Microbus` to spin a microservice along with all its downstream dependencies inside a single [application for testing](../blocks/integration-testing.md) purposes. Full-blown integration tests can then be run by `go test`, achieving a high-degree of code coverage.

In addition, the compact footprint of a `Microbus` application also enables the front-end team to run it locally rather than depend on a remote integration environment.
