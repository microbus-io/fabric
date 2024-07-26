# Adaptable Topology

In `Microbus`, microservices are not large memory-gobbling processes but rather compact worker goroutines that ultimately consume messages from a queue. Microservices also don’t listen on ports so there is no potential of a port conflict with other microservices. These two qualities allow `Microbus` to spin up any number of microservices in a single application, which provides for a wide range of deployment topologies. Each application can take the form of a standalone executable or a container. 

## Local Development

All microservices are typically bundled into a single [`Application`](../structure/application.md) for purpose of [local development](../tech/local-dev.md). This way, the entire application is spun up through the IDE and development is as simple as working with a monolith. Breakpoints can be placed in any of the microservices.

In the following diagram, 9 microservices are hosted inside a single executable that is spun up by the IDE. The microservices communicate via a single NATS node.

<img src="./topology-1.drawio.svg">
<p></p>

## Integration Tests

[Integration tests](../blocks/integration-testing.md) are a form of local development, except that the application typically includes only the subset of the microservices that are a dependency of the microservice under test. In this diagram, microservice under test `H` is bundled along with only the 4 other downstream microservices that it depends on: `A`, `C`, `E` and `I`. All 5 microservices communicate via a single NATS node.

<img src="./topology-2.drawio.svg">
<p></p>

## Simple Bundled Replication

In this deployment topology, the all-inclusive application is replicated on a multitude of hardware, and additional NATS nodes are added to form a full-mesh cluster. This strategy is a good choice for solutions with low to medium load.

<img src="./topology-3.drawio.svg">
<p></p>

Pros:
* Simple enough to be manageable without Kubernetes
* Can sustain the loss of any given hardware, NATS node or executable, including during a rolling deployment
* Scales horizontally to handle more load
* Resiliency to AZ failure can be achieved by placing each hardware in a different AZ

Cons:
* A change to even a single microservice requires redeployment of all microservices
* All microservices share the same hardware configuration

## Weighted Bundled Replication 

If a microservice handles a lot of traffic, it risks having its single connection to NATS getting bogged down. Deploying additional replicas alleviates the pressure and is a simple technique for scaling up throughput. In the diagram below, both the HTTP ingress proxy and microservice `A` are deployed twice as many times as other microservices.

<img src="./topology-4.drawio.svg">
<p></p>

Pros and cons are the same as for simple bundled replication.

## Asymmetrical Hardware

In some cases, a microservice may be required to be deployed separately from the rest of the microservices due to different SLA requirements. In this example, microservice `B` uses a high amount of memory. Isolating it to its own hardware maximizes its available memory and avoids the noisy neighbor problem.

<img src="./topology-5.drawio.svg">
<p></p>

Pros:
* Simple enough to be manageable without Kubernetes if the number of exceptions is low
* Can sustain the loss of any given hardware, NATS node or executable, including during a rolling deployment
* Scales horizontally to handle more load
* Resiliency to AZ failure can be achieved by placing each hardware type in multiple AZs

Cons:
* A change to even a single microservice is likely to require redeployment of practically all microservices
* Gets complicated to manage with many exceptions
* Increased hardware provisioning costs

## Individually Wrapped

In this deployment topology, each microservice replica is wrapped in its own individual application.

<img src="./topology-6.drawio.svg">
<p></p>

Pros:
* Well-suited for running on Kubernetes
* Maximum flexibility in setting the number of replicas and their distribution across hardware
* Can sustain the loss of any given hardware, NATS node or executable, including during a rolling deployment
* Scales horizontally to handle more load
* Redeployment is required only for changed microservices
* Resiliency to AZ failure can be achieved by placing hardware in multiple AZs and spreading the microservices so that each are replicated in at least 2 AZs

Cons:
* Memory requirements are modestly higher
* Added complexity of Kubernetes
