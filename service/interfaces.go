/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

// Package service defines the interfaces of a microservice, which the connector implements.
package service

import (
	"context"
	"io/fs"
	"iter"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/trc"
)

// Publisher are the actions used to publish to the bus.
type Publisher interface {
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	Publish(ctx context.Context, options ...pub.Option) iter.Seq[*pub.Response]
}

// Subscriber are the actions used to subscribe to the bus.
type Subscriber interface {
	Subscribe(name string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(name string) error
	ActivateSubscription(name string) error
	DeactivateSubscription(name string) error
}

// PublisherSubscriber is both a publisher and a subscriber.
type PublisherSubscriber interface {
	Publisher
	Subscriber
}

// Logger are the actions used to output log messages.
type Logger interface {
	LogDebug(ctx context.Context, msg string, args ...any)
	LogInfo(ctx context.Context, msg string, args ...any)
	LogWarn(ctx context.Context, msg string, args ...any)
	LogError(ctx context.Context, msg string, args ...any)
}

// ObserveMetricsHandler handles the OnObserveMetrics callback.
type ObserveMetricsHandler func(ctx context.Context) error

// MeterDescriber are the actions used to describe metrics.
type MeterDescriber interface {
	DescribeCounter(name string, desc string) (err error)
	DescribeGauge(name string, desc string) (err error)
	DescribeHistogram(name string, desc string, buckets []float64) (err error)

	SetOnObserveMetrics(handler ObserveMetricsHandler) error
}

// Meter are the actions used to record metrics.
type Meter interface {
	IncrementCounter(ctx context.Context, name string, val float64, attributes ...any) (err error)
	RecordGauge(ctx context.Context, name string, val float64, attributes ...any) (err error)
	RecordHistogram(ctx context.Context, name string, val float64, attributes ...any) (err error)
}

// Tracer are the actions used to operate distributed tracing spans.
type Tracer interface {
	StartSpan(ctx context.Context, spanName string, opts ...trc.Option) (context.Context, trc.Span)
	Span(ctx context.Context) trc.Span
	ForceTrace(ctx context.Context)
}

// StartupHandler handles the OnStartup callback.
type StartupHandler func(ctx context.Context) error

// ShutdownHandler handles the OnShutdown callback.
type ShutdownHandler func(ctx context.Context) error

// StarterStopper are the lifecycle actions of the microservice.
type StarterStopper interface {
	Startup(ctx context.Context) (err error)
	Shutdown(ctx context.Context) error
	IsStarted() bool
	Lifetime() context.Context

	SetHostname(hostname string) error
	SetDeployment(deployment string) error
	SetPlane(plane string) error

	SetOnStartup(handler StartupHandler) error
	SetOnShutdown(handler ShutdownHandler) error
}

// Identifier are the properties used to uniquely identify and address the microservice.
type Identifier interface {
	ID() string
	Hostname() string
	Description() string
	Version() int
	Deployment() string
	Plane() string
	Locality() string
}

// ConfigChangedHandler handles the OnConfigChanged callback.
type ConfigChangedHandler func(ctx context.Context, changed func(string) bool) error

// Configurable are the actions used to configure the microservice.
type Configurable interface {
	DefineConfig(name string, options ...cfg.Option) error
	Config(name string) (value string)
	SetConfig(name string, value any) error
	ResetConfig(name string) error

	SetOnConfigChanged(handler ConfigChangedHandler) error
}

// Resourcer provides access to the connector's FS.
type Resourcer interface {
	SetResFS(resFS fs.FS) error
	ResFS() fs.FS
}

// TickerHandler handles the ticker callbacks.
type TickerHandler func(ctx context.Context) error

// Ticker are actions used to schedule recurring jobs.
type Ticker interface {
	StartTicker(name string, interval time.Duration, handler TickerHandler) error
}

// Executor are actions for running jobs in Go routines.
type Executor interface {
	Go(ctx context.Context, f func(ctx context.Context) (err error)) error
	Parallel(ctx context.Context, jobs ...func(ctx context.Context) (err error)) error
}

// Timer are actions related to time management.
type Timer interface {
	Sleep(ctx context.Context, duration time.Duration) error
}

// Service are all the actions that a connector provides.
type Service interface {
	Publisher
	Subscriber
	Logger
	MeterDescriber
	Meter
	Tracer
	StarterStopper
	Identifier
	Configurable
	Resourcer
	Ticker
	Timer
	Executor
}
