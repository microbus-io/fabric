# Metrics

Metrics within the Microbus framework relies on Prometheus and operates on a pull model. Prometheus will pull metrics from the metrics service, which in turn pulls and aggregates metrics from all running microservices in the current deployment.

## Service Metrics

All services by default expose a minimal set of metrics pertaining to the handling of incoming and outgoing service calls. These include:
* The current service uptime
* The internal processing time of any incoming service calls
* The size of the response message to incoming service calls
* The total duration of any outgoing requests from a service
* ...
* ...


## Application Metrics
In addition, developers are free to add any other custom application level service metrics that may be useful. These should be defined in the service.yaml under the metrics collection.

```yaml
metrics:
  - signature: Likes(num int, postId string)
    description: Likes counts the number of likes for a given post.
    type: Counter | Guage | Histogram
    alias: myapp_message_post_number_of_likes
    buckets: 0,10,100,1000,100000 # Bucket boundaries, only applicable to Histogram type
```

With regard to alias names, see https://prometheus.io/docs/practices/naming/ for best practices.

The collection types supported are:
* Counter
* Histogram
* Guage

See https://prometheus.io/docs/concepts/metric_types/ for details on the different types.

## Code Examples

```go
/* TODO */
```

## Local Setup

To support and test Metrics when running locally, using a docker image is recommended. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) if not already installed and pull the latest Prometheus docker image.

```cmd
docker pull prom/prometheus
```

Make sure the metrics service are started and then start the prometheus container with the following command:

```cmd
docker run -p 9090:9090 -v path/to/github.com/microbus-io/fabric/examples/main/prometheus.yaml:/etc/prometheus/prometheus.yml prom/prometheus
```

The provided sample prometheus.yaml will scrape from the metrics service every 5 seconds. You can verify by navigating to http://localhost:9090/graph and executing `service_uptime_duration_seconds_total` in the query box. If successful, you should see the current uptime of the metrics service.