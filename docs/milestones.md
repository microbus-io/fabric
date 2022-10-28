# Milestones

Note: These milestones are maintained in separate branches in order to demonstrate the thinking process behind the building of this framework. Milestone are not releases and they are not production-ready.

[Milestone 1](https://github.com/microbus-io/fabric/tree/milestone/1):

* `Connector` construct, including startup and shutdown logic
* Config from environment or file
* Rudimentary logger
* Bi-directional (request/response) HTTP-like unicast messaging over NATS
* HTTP ingress proxy

[Milestone 2](https://github.com/microbus-io/fabric/tree/milestone/2):

* Error capture and propagation
* Time budget for requests

[Milestone 3](https://github.com/microbus-io/fabric/tree/milestone/3):

* `Application` construct to group microservices
* NATS connection settings
* Deployment environment indicator (`PROD`, `LAB`, `LOCAL`)
* Plane of communication

[Milestone 4](https://github.com/microbus-io/fabric/tree/milestone/4):

* Acks and quick timeouts
* Bi-directional (request/response) HTTP-like *multicast* messaging over NATS
* Direct addressing
* Control messages on port `:888`

[Milestone 5](https://github.com/microbus-io/fabric/tree/milestone/5):

* Advanced logger with JSON output

[Milestone 6](https://github.com/microbus-io/fabric/tree/milestone/6):

* Fragmentation of large messages
* Optimized HTTP utilities (`BodyReader` and `ResponseRecorder`)

[Milestone 7](https://github.com/microbus-io/fabric/tree/milestone/7):

* Tickers
* Mockable clock
* Context for the `Lifetime` of the microservice
* Customizable time budget for callbacks
* Drain pending operations during shutdown

[Milestone 8](https://github.com/microbus-io/fabric/tree/milestone/8):

* Configurator microservice to centralize the dissemination of configuration values to other microservices
* Config property definition with value validation
* `Connector` description
* Update of NATS connection environment variable names
* Update of deployment and plane environment variable names

[Milestone 9](https://github.com/microbus-io/fabric/tree/milestone/9):

* LRU cache
* Distributed LRU cache
