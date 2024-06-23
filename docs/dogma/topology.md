# Adaptive Deployment Topology

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
