# Elevated Engineering Experience

TODO
* Speed up development with code generation of boilerplate code
* -- Add new microservices and endpoints in minutes
* Observe your system with pinpoint accuracy to troubleshoot and optimize your code, while building and in production
* -- Start an entire application on a development machine in seconds
* Run, test and debug an entire application comprising a multitude of microservices in your IDE
* -- Perform thorough integration tests that include a multitude of microservices in a single test
* -- Easily integrate external clients such as single-page applications using automated OpenAPI documents
* -- Avoid common pitfalls with best-practices that are transparently baked-in
* Front end team can run the app locally too!

  Developer productivity is directly correlated to customer value
  Best practices baked in: e.g. 12factor.net
  Diagram: service.yaml -> codegen -> service stub, client stub, test harness
  Observability out of the box: Jaeger, grafana, logs, error capture. See your code! Troubleshooting.
  OpenAPI to integrate with FE team or customers
  Take the mundane nitty-gritty boilerplate work out of the picture. Focus on business logic that brings customer value. On domain architecture that keeps the system stable.
  Eliminate common pitfalls
  Run entire app local. Restart in seconds after code change
  Contrast with k3s, port mapping
  Strip away need for boilerplate code. Focus on biz logic
  Uniform code structure: your engineers are portable. 75% of time is reading other people’s code or your own code 6 months later
  Codegen RAD tools
  Best practices (DLRU, ack or fail fast, time budget, events, graceful shutdown)
  Guardrails and handholds (prevent spaghetti coding)
  Full integration testings in unit tests
  Engineering velocity: development, testing and operation
  Develop as a modular monolith. Codegen guides that.

  Microservices are not large memory-gobbling processes, but compact worker goroutines that process messages. They don’t listen on ports so they don’t clash with one another.
