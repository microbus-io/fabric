# Package `mtr`

The `mtr` package defines the components needed to support Prometheus metrics. The [collector types](https://prometheus.io/docs/concepts/metric_types/) supported are `Counter`, `Gauge`, and `Histogram`. The `Summary` collector type is currently not supported. 

Valid operations on the collector types are `Increment`, `Observe`, or both. 

| Type | Increment | Observe |
|---|---|---
| Counter | Yes | If increasing |
| Histogram | No | Yes |
| Gauge | Yes | Yes |

Once defined using either `DefineCounter`, `DefineHistorgram` or `DefineGauge`, a metric can be incremented or observed using either  `IncrementMetric` or `ObserveMetric`, as appropriate.
