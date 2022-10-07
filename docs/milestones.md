# Milestones

[Milestone 1](https://github.com/microbus-io/fabric/tree/milestone/1):

* `Connector` construct, including startup and shutdown logic
* Config from environment or file
* Rudimentary logger
* HTTP request/response messaging over NATS
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

* Request acks and quick timeouts
* HTTP pub/sub messaging over NATS
* Direct addressing subscriptions
* Control messages on port `:888`
