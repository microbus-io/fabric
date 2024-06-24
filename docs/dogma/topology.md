# Adaptive Deployment Topology

In `Microbus`, microservices are not large memory-gobbling processes but rather compact worker goroutines that ultimately consume messages from a queue. Microservices also don’t listen on ports so there is no potential of a port conflict with other microservices. These two qualities allow `Microbus` to spin up any number of microservices in a single executable, which provides for a wide range of deployment topologies.

## Local Development

Typically, all microservices are bundled into a single [`Application`](../structure/application.md) for purpose of local development. This way, the entire application is spun up through the IDE and development is as simple as working with a monolith. Breakpoints can be placed in any of the microservices.

In the following diagram, 9 microservices are hosted inside a single executable that is spun up by the IDE. The microservices communicate via a single NATS node.

<img src="./topology-1.drawio.svg">
<p>

## Integration Tests

[Integration tests](../blocks/integration-testing.md) typically include only the subset of the microservices that are a dependency of the microservice under test. 

In this diagram, microservice under test `H` is bundled along with only the 4 other downstream microservices that it depends on. All 5 microservices communicate via a single NATS node.

<img src="./topology-2.drawio.svg">
<p>

## Simple Replication

In this replication strategy, the all-inclusive executable is replicated N times, and additional NATS nodes are added to form a full-mesh cluster. 

<img src="./topology-3.drawio.svg">
<p>

Pros:
* Simple enough to be manageable without Kubernetes
* Can sustain the loss of any given hardware, NATS node or executable, including during a rolling deployment
* Scales horizontally to handle more traffic correlated to the replication factor N

Cons:
* Not geo-aware and will incur higher latency if the hardware is deployed across AZs
* A change to even a single microservice requires redeployment of all microservices
* All microservices share the same hardware configuration

## Exceptional Outliers 

Service A requires more executables because it handles more traffic. The bottleneck of a microservice is its connection to NATS.
Service B requires more memory and is isolated into bigger hardware.

## Individual Replication

Each microservice in its own executable.
Likely requires k8s.

## Geo-Aware Failover

NATS super cluster


TODO
* -- Develop with the simplicity and velocity of a modular monolith running locally
* Deploy as a resilient modular monolith, or a multi-region cloud deployment with geo-aware failover
* Run as a standalone executable or on Kubernetes
* Include one or many microservices in each executable or Kubernetes pod

  from localdev to full-on k8s, grows with you with no code change
  Develop as a modular monolith, deploy as microservices when the time comes
  Horizontally scalable
  Cost efficient
  1 or many microservices per deployment unit
  Microservices are not large memory-gobbling processes, but compact worker goroutines that process messages. They don’t listen on ports so they don’t clash with one another.

Flexible/adaptive deployment topologies:
  Local dev 1 process - modular monolith
  Resilient PROD, 3+ clones, single AZ
  Resilient PROD, 3+ clones - horizontally scalable monolith, multi AZ
    Connect to NATS on same AZ
  Resilient PROD, 6+ clones, multi-region. With NATS clusters
  Flexible K8s PROD, N individually deployable pods - full microservices, multi AZ
  Geo-aware fallback
