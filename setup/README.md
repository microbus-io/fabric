# Setup and Run the Development Environment

The `Microbus` framework depends on a few third-party services:

* NATS is a hard requirement used as the communication transport between microservice
* MariaDB is needed for microservices that depend on it, such as the CRUD example
* Prometheus is an optional service that can be used to collect metrics from `Microbus` microservices
* Grafana is an optional service that can visualize metrics collected by Prometheus

Use `docker compose` from within this directory to install the above and set up the development environment.

```cmd
docker compose -f microbus.yaml -p microbus up
```
