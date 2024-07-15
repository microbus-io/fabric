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

// Package service defines the interfaces of a microservice, which the connector implements.
package service

import (
	"context"
	"io/fs"
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
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
}

// Subscriber are the actions used to subscribe to the bus.
type Subscriber interface {
	Subscribe(method string, path string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(method string, path string) error
}

// PublisherSubscriber is both a publisher and a subscriber.
type PublisherSubscriber interface {
	Publisher
	Subscriber
}

// Subscriber are the actions used to output log messages.
type Logger interface {
	LogDebug(ctx context.Context, msg string, args ...any)
	LogInfo(ctx context.Context, msg string, args ...any)
	LogWarn(ctx context.Context, msg string, args ...any)
	LogError(ctx context.Context, msg string, args ...any)
}

// Tracer are the actions used to operate distributed tracing spans.
type Tracer interface {
	StartSpan(ctx context.Context, spanName string, opts ...trc.Option) (context.Context, trc.Span)
	Span(ctx context.Context) trc.Span
	ForceTrace(ctx context.Context)
}

// StartupHandler handles the OnStartup callback.
type StartupHandler func(ctx context.Context) error

// StartupHandler handles the OnShutdown callback.
type ShutdownHandler func(ctx context.Context) error

// StarterStopper are the lifecycle actions of the microservice.
type StarterStopper interface {
	Startup() (err error)
	Shutdown() error
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

// StartupHandler handles the OnStartup callback.
type ConfigChangedHandler func(ctx context.Context, changed func(string) bool) error

// Configurable are the actions used to configure the microservice.
type Configurable interface {
	DefineConfig(name string, options ...cfg.Option) error
	Config(name string) (value string)
	SetConfig(name string, value any) error
	ResetConfig(name string) error

	SetOnConfigChanged(handler ConfigChangedHandler) error
}

// FS is a file system that can be associated with the connector.
type FS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

// Resourcer provides access to the connector's FS.
type Resourcer interface {
	SetResFS(resFS FS) error
	ResFS() FS
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
	Parallel(jobs ...func() (err error)) error
}

// Service are all the actions that a connector provides.
type Service interface {
	Publisher
	Subscriber
	Logger
	Tracer
	StarterStopper
	Identifier
	Configurable
	Resourcer
	Ticker
	Executor
}
