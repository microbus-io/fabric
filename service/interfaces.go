/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

// Package service defines the interfaces of a microservice, which the connector implements.
package service

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/log"
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
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogWarn(ctx context.Context, msg string, fields ...log.Field)
	LogError(ctx context.Context, msg string, fields ...log.Field)
}

// Tracer are the actions used to operate distributed tracing spans.
type Tracer interface {
	StartSpan(ctx context.Context, spanName string, opts ...trc.Option) (context.Context, trc.Span)
	Span(ctx context.Context) trc.Span
	ForceTrace(span trc.Span)
}

// StarterStopper are the lifecycle actions of the microservice.
type StarterStopper interface {
	Startup() (err error)
	Shutdown() error
	IsStarted() bool
	Lifetime() context.Context

	SetHostName(hostName string) error
	SetDeployment(deployment string) error
	SetPlane(plane string) error
	With(options ...func(Service) error) Service
}

// Identifier are the properties used to uniquely identify and address the microservice.
type Identifier interface {
	ID() string
	HostName() string
	Description() string
	Version() int
	Deployment() string
	Plane() string
}

// Configurable are the actions used to configure the microservice.
type Configurable interface {
	SetConfig(name string, value any) error
	ResetConfig(name string) error
}

// Service are all the actions that a microservice provides.
type Service interface {
	Publisher
	Subscriber
	Logger
	Tracer
	StarterStopper
	Identifier
	Configurable
}
