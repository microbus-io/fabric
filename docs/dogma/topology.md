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
