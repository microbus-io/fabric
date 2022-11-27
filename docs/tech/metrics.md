# Metrics

Metrics within the `Microbus` framework relies on Prometheus and operates on a pull model. Prometheus pulls metrics from the metrics system microservice, which in turn pulls and aggregates metrics from all running microservices in the current deployment.

## Standard Metrics

All services by default expose a minimal set of metrics pertaining to the handling of incoming and outgoing requests. These include:
* The current service uptime
* The internal processing time of any incoming requests
* The size of the response message to incoming requests
* The total duration of any outgoing requests
* ...
* ...


## Application Metrics
In addition, developers are free to add any other custom application metrics that may be useful. These can be defined in the `service.yaml` under the metrics section.

```yaml
# Metrics
# signature - A Go function signature. Example:
#   RequestDurationSeconds(dur time.Duration, method string, success bool)
#   MemoryUsageBytes(b int64)
#   DistanceMiles(miles float64, countryCode int)
#   RequestsTotal(count int, domain string) ... unit-less accumulating count
#   CPUSecondsTotal(dur time.Duration) ... accumulating count with unit
# description - Go-doc description of the endpoint
# type - The type of the metric: "Histogram", "Gauge" or "Counter" (default)
# alias - Override the name of the metric to convey to Prometheus
# buckets - Bucket boundaries, for histograms
metrics:
  - signature: Likes(num int, postId string)
    description: Likes counts the number of likes for a given post.
    type: Counter | Guage | Histogram
    alias: myapp_message_post_number_of_likes
```

With regard to alias names, see [naming best practices](https://prometheus.io/docs/practices/naming/) for best practices.

The [collection types](https://prometheus.io/docs/concepts/metric_types/) supported are:
* Counter
* Histogram
* Guage


## Code Examples

```go
/* TODO */
```