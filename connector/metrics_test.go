/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestConnector_DefineMetrics(t *testing.T) {
	t.Parallel()

	con := New("define.metrics.connector")
	testarossa.False(t, con.IsStarted())

	// Define all three collector types before starting up
	err := con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram",
		"my historgram",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge",
		"my gauge",
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)

	// Duplicate key
	err = con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a", "b", "c"},
	)
	testarossa.Error(t, err)

	// Startup
	con.initErr = nil
	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Define all three collector types after starting up
	err = con.DefineCounter(
		"my_counter_2",
		"my counter 2",
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram_2",
		"my historgram 2",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge_2",
		"my gauge 2",
		[]string{"a", "b", "c"},
	)
	testarossa.NoError(t, err)

	// Duplicate key
	err = con.DefineCounter(
		"my_counter_2",
		"my counter 2",
		[]string{"a", "b", "c"},
	)
	testarossa.Error(t, err)
}

func TestConnector_ObserveMetrics(t *testing.T) {
	t.Parallel()

	con := New("observe.metrics.connector")
	testarossa.False(t, con.IsStarted())

	// Define all three collector types before starting up
	err := con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a"},
	)
	testarossa.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram",
		"my histogram",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a"},
	)
	testarossa.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge",
		"my gauge",
		[]string{"a"},
	)
	testarossa.NoError(t, err)

	// Startup
	con.initErr = nil
	err = con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Histogram
	err = con.ObserveMetric("my_histogram", 2.5, "1")
	testarossa.NoError(t, err)
	err = con.IncrementMetric("my_histogram", 1.5, "1")
	testarossa.Error(t, err)

	// Gauge
	err = con.ObserveMetric("my_gauge", 2.5, "1")
	testarossa.NoError(t, err)
	err = con.ObserveMetric("my_gauge", 2.5, "1")
	testarossa.NoError(t, err)
	err = con.ObserveMetric("my_gauge", -2.5, "1")
	testarossa.NoError(t, err)
	err = con.IncrementMetric("my_gauge", 1.5, "1")
	testarossa.NoError(t, err)
	err = con.IncrementMetric("my_gauge", -0.5, "1")
	testarossa.NoError(t, err)

	// Counter
	err = con.IncrementMetric("my_counter", 1.5, "1")
	testarossa.NoError(t, err)
	err = con.IncrementMetric("my_counter", -1.5, "1")
	testarossa.Error(t, err)
	err = con.ObserveMetric("my_counter", 1.5, "1")
	testarossa.Error(t, err)
}

func TestConnector_StandardMetrics(t *testing.T) {
	t.Parallel()

	con := New("standard.metrics.connector")
	testarossa.Equal(t, 11, len(con.metricDefs))
	testarossa.NotNil(t, con.metricDefs["microbus_callback_duration_seconds"])
	testarossa.NotNil(t, con.metricDefs["microbus_response_duration_seconds"])
	testarossa.NotNil(t, con.metricDefs["microbus_response_size_bytes"])
	testarossa.NotNil(t, con.metricDefs["microbus_request_count_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_ack_duration_seconds"])
	testarossa.NotNil(t, con.metricDefs["microbus_log_messages_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_uptime_duration_seconds_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_cache_hits_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_cache_misses_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_cache_weight_total"])
	testarossa.NotNil(t, con.metricDefs["microbus_cache_len_total"])
}
