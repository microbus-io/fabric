# Package `examples`

The `examples` package holds several examples that demonstrate how the framework can be used to create microservices. When studying an example, start by looking at the `service.yaml` to get a quick overview of the functionality of the microservice. Then go deep into the code in `service.go`. All files with `-gen` in their name are code generated and can be ignored unless studying the internals of the generated code.

* [Hello](./examples-hello.md) demonstrates the key capabilities of the framework
* [Calculator](./examples-calculator.md) demonstrates functional handlers
* [Messaging](./examples-messaging.md) demonstrates load-balanced unicast, multicast and direct addressing messaging
* [Event source and sink](./examples-events.md) shows how events can be used to reverse the dependency between two microservices
* [Directory](./examples-directory.md) is an example of a microservice that provides a CRUD API backed by a MySQL database

In case you missed it, the [quick start guide](../quick-start.md) explains how to setup your system to run the examples.
