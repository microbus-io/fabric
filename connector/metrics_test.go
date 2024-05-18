/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_DefineMetrics(t *testing.T) {
	t.Parallel()

	con := New("define.metrics.connector")
	assert.False(t, con.IsStarted())

	// Define all three collector types before starting up
	err := con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram",
		"my historgram",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge",
		"my gauge",
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)

	// Duplicate key
	err = con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a", "b", "c"},
	)
	assert.Error(t, err)

	// Startup
	con.initErr = nil
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Define all three collector types after starting up
	err = con.DefineCounter(
		"my_counter_2",
		"my counter 2",
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram_2",
		"my historgram 2",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge_2",
		"my gauge 2",
		[]string{"a", "b", "c"},
	)
	assert.NoError(t, err)

	// Duplicate key
	err = con.DefineCounter(
		"my_counter_2",
		"my counter 2",
		[]string{"a", "b", "c"},
	)
	assert.Error(t, err)
}

func TestConnector_ObserveMetrics(t *testing.T) {
	t.Parallel()

	con := New("observe.metrics.connector")
	assert.False(t, con.IsStarted())

	// Define all three collector types before starting up
	err := con.DefineCounter(
		"my_counter",
		"my counter",
		[]string{"a"},
	)
	assert.NoError(t, err)
	err = con.DefineHistogram(
		"my_histogram",
		"my historgram",
		[]float64{1, 2, 3, 4, 5},
		[]string{"a"},
	)
	assert.NoError(t, err)
	err = con.DefineGauge(
		"my_gauge",
		"my gauge",
		[]string{"a"},
	)
	assert.NoError(t, err)

	// Startup
	con.initErr = nil
	err = con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Histogram
	err = con.ObserveMetric("my_histogram", 2.5, "1")
	assert.NoError(t, err)
	err = con.IncrementMetric("my_histogram", 1.5, "1")
	assert.Error(t, err)

	// Gauge
	err = con.ObserveMetric("my_gauge", 2.5, "1")
	assert.NoError(t, err)
	err = con.ObserveMetric("my_gauge", 2.5, "1")
	assert.NoError(t, err)
	err = con.ObserveMetric("my_gauge", -2.5, "1")
	assert.NoError(t, err)
	err = con.IncrementMetric("my_gauge", 1.5, "1")
	assert.NoError(t, err)

	// Counter
	err = con.ObserveMetric("my_counter", 2.5, "1")
	assert.NoError(t, err)
	err = con.ObserveMetric("my_counter", 2.5, "1")
	assert.NoError(t, err)
	err = con.ObserveMetric("my_counter", 3.5, "1")
	assert.NoError(t, err)
	err = con.ObserveMetric("my_counter", 1.5, "1")
	assert.Error(t, err)
	err = con.IncrementMetric("my_counter", 1.5, "1")
	assert.NoError(t, err)
}

func TestConnector_StandardMetrics(t *testing.T) {
	t.Parallel()

	con := New("standard.metrics.connector")
	assert.Equal(t, 11, len(con.metricDefs))
	assert.NotNil(t, con.metricDefs["microbus_callback_duration_seconds"])
	assert.NotNil(t, con.metricDefs["microbus_response_duration_seconds"])
	assert.NotNil(t, con.metricDefs["microbus_response_size_bytes"])
	assert.NotNil(t, con.metricDefs["microbus_request_count_total"])
	assert.NotNil(t, con.metricDefs["microbus_ack_duration_seconds"])
	assert.NotNil(t, con.metricDefs["microbus_log_messages_total"])
	assert.NotNil(t, con.metricDefs["microbus_uptime_duration_seconds_total"])
	assert.NotNil(t, con.metricDefs["microbus_cache_hits_total"])
	assert.NotNil(t, con.metricDefs["microbus_cache_misses_total"])
	assert.NotNil(t, con.metricDefs["microbus_cache_weight_total"])
	assert.NotNil(t, con.metricDefs["microbus_cache_len_total"])
}
