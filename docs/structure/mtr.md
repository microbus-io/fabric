# Package `mtr`

The `mtr` package defines the components needed to support standard and custom Prometheus metrics. The [collection types](https://prometheus.io/docs/concepts/metric_types/) supported are `Counter`, `Gauge`, and `Histogram`. `Summary` type is not currently supported. 

Valid operations on the collection types are either `Add`, `Observe`, or both. 
* Counter	Add
* Histogram	Observe
* Guage		Add | Observe

The Prometheus library collection objects for a specific named collection are stored in a Map located in the `Connector`. Once retrieved, the metric can be incremented or observed as needed.

Example:

```go
func (c *Connector) IncrementMetric(name string, val float64, labels ...string) error {
	if c.registry == nil {
		return nil
	}
	c.metricLock.RLock()
	defer c.metricLock.RUnlock()
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}

	err := m.Add(val, append([]string{c.HostName(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
	return errors.Trace(err, name)
}

func (c *Connector) ObserveMetric(name string, val float64, labels ...string) error {
	if c.registry == nil {
		return nil
	}
	c.metricLock.Lock()
	defer c.metricLock.Unlock()
	m, ok := c.metricDefs[name]
	if !ok {
		return errors.Newf("unknown metric '%s'", name)
	}
	err := m.Observe(val, append([]string{c.HostName(), strconv.Itoa(c.Version()), c.ID()}, labels...)...)
	return errors.Trace(err, name)
}
```
