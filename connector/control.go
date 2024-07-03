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
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
)

// subscribeControl creates subscriptions for control requests on the reserved port 888.
func (c *Connector) subscribeControl() error {
	type ctrlSub struct {
		path    string
		handler HTTPHandler
		options []sub.Option
	}
	subs := []*ctrlSub{
		{
			path:    "ping",
			handler: c.handleControlPing,
			options: []sub.Option{sub.NoQueue()},
		},
		{
			path:    "config-refresh",
			handler: c.handleControlConfigRefresh,
			options: []sub.Option{sub.NoQueue()},
		},
		{
			path:    "metrics",
			handler: c.handleMetrics,
			options: []sub.Option{sub.NoQueue()},
		},
		{
			path:    "trace",
			handler: c.handleTrace,
			options: []sub.Option{sub.NoQueue()},
		},
	}
	for _, s := range subs {
		err := c.Subscribe("ANY", ":888/"+s.path, s.handler, s.options...)
		if err != nil {
			return errors.Trace(err)
		}
		err = c.Subscribe("ANY", "https://all:888/"+s.path, s.handler, s.options...)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// handleControlPing responds to the :888/ping control request with a pong.
func (c *Connector) handleControlPing(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"pong":0}`))
	return nil
}

// handleControlConfigRefresh responds to the :888/config-refresh control request
// by pulling the latest config values from the configurator service.
func (c *Connector) handleControlConfigRefresh(w http.ResponseWriter, r *http.Request) error {
	err := c.refreshConfig(r.Context(), true)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}

// handleMetrics responds to the :888/metrics control request with collected metrics.
func (c *Connector) handleMetrics(w http.ResponseWriter, r *http.Request) error {
	_ = c.ObserveMetric("microbus_uptime_duration_seconds_total", time.Since(c.startupTime).Seconds())
	_ = c.ObserveMetric("microbus_cache_len_total", float64(c.distribCache.LocalCache().Len()))
	_ = c.ObserveMetric("microbus_cache_weight_total", float64(c.distribCache.LocalCache().Weight()))
	_ = c.ObserveMetric("microbus_cache_hits_total", float64(c.distribCache.Hits()))
	_ = c.ObserveMetric("microbus_cache_misses_total", float64(c.distribCache.Misses()))
	if c.metricsHandler != nil {
		c.metricsHandler.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotImplemented)
	}
	return nil
}

// handleTrace responds to the :888/trace control request
// to force exporting the indicated tracing span.
func (c *Connector) handleTrace(w http.ResponseWriter, r *http.Request) error {
	if c.traceProcessor != nil {
		traceID := r.URL.Query().Get("id")
		c.traceProcessor.Select(traceID)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}
