/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package mtr

// Metric is an interface that defines operations for the metric collectors.
type Metric interface {
	Observe(val float64, labels ...string) error
	Add(val float64, labels ...string) error
}
