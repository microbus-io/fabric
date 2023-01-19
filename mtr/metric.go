package mtr

// Metric is an interface that defines operations for the metric collectors.
type Metric interface {
	Observe(val float64, labels ...string) error
	Add(val float64, labels ...string) error
}
