# Quick Start

## Clone the Project

Fetch the code:

```cmd
mkdir github.com/microbus-io
cd github.com/microbus-io
git clone https://github.com/microbus-io/fabric
```

## Install and Run NATS

From the root folder of this project:

```cmd
go get github.com/nats-io/nats-server
go build github.com/nats-io/nats-server
./nats-server -D -V
```

The `-D` and `-V` flags will produce a lot of output. It's recommended to start the NATS server in a separate terminal window to better be able to see the action. Remove these flags if speed is important, such as when running benchmarks and certain tests.

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

## Configure IDE

The [Todo Tree extension](https://marketplace.visualstudio.com/items?itemName=Gruntfuggly.todo-tree) is recommended for VS Code users.

## Metrics Local Setup

Metrics will not be available by default in a local environment. Docker Desktop will need to be installed in order to run a Promethues in a container. For more information on metrics see [Metrics](docs/tech/metrics.md) 

To support and test metrics when running locally, using a docker image is recommended. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) if not already installed and pull the latest Prometheus docker image.

```cmd
docker pull prom/prometheus
```

Make sure the metrics microservice are started and then start the Prometheus container with the following command:

```cmd
docker run -p 9090:9090 -v path/to/github.com/microbus-io/fabric/examples/main/prometheus.yaml:/etc/prometheus/prometheus.yml prom/prometheus
```

The provided sample examples/main/prometheus.yaml will scrape from the metrics microservice every 15 seconds. You can verify by navigating to http://localhost:9090/graph and executing `service_uptime_duration_seconds_total` in the query box. If successful, you should see the current uptime of the metrics microservice.
