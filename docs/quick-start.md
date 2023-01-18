# Quick Start

## Clone the Project

Fetch the code:

```cmd
mkdir github.com/microbus-io
cd github.com/microbus-io
git clone https://github.com/microbus-io/fabric
```

## Run the development environment

The docker environment is self-contained with the following preconfigured services:

* Nats
* MySql
* Prometheus
* Grafana

From the docker folder of this project:

```cmd
docker-compose -f microbus.yaml up
```

The `-DV` flags will produce a lot of output. It's recommended to start the NATS server in a separate terminal window to better be able to see the action. Remove these flags from the nats service in the microbus.yaml docker compose file if speed is important, such as when running benchmarks and certain tests.

## Run the Examples

Run the example app:

```cmd
cd examples/main
go run main.go
```

It is important to set the working directory to `examples/main` so that the `examples/main/env.yaml` file is located.

If you're using Visual Studio Code, simply press `F5`. The `.vscode/launch.json` file includes a launch configuration for running `examples/main`.

Try the following URLs in your browser:

* http://localhost:8080/calculator.example/arithmetic?x=5&op=*&y=8
* http://localhost:8080/calculator.example/square?x=5
* http://localhost:8080/calculator.example/square?x=not-a-number
* http://localhost:8080/calculator.example/distance?p1.x=0&p1.y=0&p2.x=3&p2.y=4
* http://localhost:8080/hello.example/echo
* http://localhost:8080/hello.example/ping
* http://localhost:8080/hello.example/hello?name=Bella
* http://localhost:8080/hello.example/calculator
* http://localhost:8080/hello.example/bus.jpeg
* http://localhost:8080/messaging.example/home
* http://localhost:8080/messaging.example/cache-store?key=foo&value=bar
* http://localhost:8080/messaging.example/cache-load?key=foo

Feel free to experiment with different values for the query arguments.

## Metrics

[Metrics](docs/tech/metrics.md) in `Microbus` are collected by [Prometheus](https://prometheus.io) and can be viewed with [Grafana](https://grafana.com/) dashboards for greater insight into the microbus system. A docker compose [yaml file](docker/microbus.yaml) is provided which will start both Prometheus and Grafana. From the docker directory run:

Make sure that the development environment has been started with docker-compose and the metrics system microservice is included in your app and started in order for Prometheus to be able to collect metrics from your microservices.

The provided sample `docker/prometheus.yaml` instructs Prometheus to scrape metrics from the metrics system microservice every 15 seconds. To verify, navigate to http://localhost:9090/graph and execute the query `microbus_uptime_duration_seconds_total`. If successful, you should see the uptime of the running microservices.

To view the Grafana dashboard, navigate to http://localhost:3000 and login with admin:admin. TODO: Create Grafana dashboards

## Configure IDE

The [Todo Tree extension](https://marketplace.visualstudio.com/items?itemName=Gruntfuggly.todo-tree) is recommended for VS Code users.
