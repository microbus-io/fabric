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

package smtpingress

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/smtpingress/smtpingressapi"
	"github.com/microbus-io/fabric/trc"
	"github.com/mnako/letters"
	"github.com/phires/go-guerrilla"
	"github.com/phires/go-guerrilla/backends"
	glog "github.com/phires/go-guerrilla/log"
	"github.com/phires/go-guerrilla/mail"
	"github.com/sirupsen/logrus"
)

const processorName = "MessageProcessor"

/*
Service implements the smtp.ingress.core microservice.

The SMTP ingress microservice listens for incoming emails and fires corresponding events.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	daemon *guerrilla.Daemon
	mux    sync.Mutex
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	err = svc.startDaemon(ctx)
	return errors.Trace(err)
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	svc.stopDaemon(ctx)
	return nil
}

// configDaemon builds the config of the email daemon.
func (svc *Service) configDaemon(_ context.Context) (*guerrilla.AppConfig, error) {
	port := strconv.Itoa(svc.Port())
	certFile := "smtpingress-" + port + "-cert.pem"
	keyFile := "smtpingress-" + port + "-key.pem"
	secure := true
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		secure = false
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		secure = false
	}

	// See https://github.com/phires/go-guerrilla/wiki/API-&-Using-as-a-package
	serverCfg := guerrilla.ServerConfig{
		ListenInterface: ":" + port,
		IsEnabled:       svc.Enabled(),
		MaxSize:         int64(svc.MaxSize()) << 20,
		MaxClients:      svc.MaxClients(),
	}
	if secure {
		serverCfg.TLS = guerrilla.ServerTLSConfig{
			PublicKeyFile:  certFile,
			PrivateKeyFile: keyFile,
			StartTLSOn:     true,
		}
	}
	cfg := &guerrilla.AppConfig{
		LogFile:      glog.OutputOff.String(),
		LogLevel:     "fail",        // Hack to prevent Guerilla from creating its own logger
		AllowedHosts: []string{"."}, // All hosts
		Servers:      []guerrilla.ServerConfig{serverCfg},
		BackendConfig: backends.BackendConfig{
			"save_workers_size": svc.Workers(),
			"save_process":      "HeadersParser|Header|" + processorName,
		},
	}
	return cfg, nil
}

// startDaemon starts the email daemon.
func (svc *Service) startDaemon(ctx context.Context) (err error) {
	cfg, err := svc.configDaemon(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	hook := logHook{svc: svc}
	svc.daemon = &guerrilla.Daemon{
		Config: cfg,
		Logger: &glog.HookedLogger{
			Logger: &logrus.Logger{
				Out:       io.Discard,
				Formatter: new(logrus.JSONFormatter),
				Hooks: logrus.LevelHooks{
					logrus.DebugLevel: []logrus.Hook{hook},
					logrus.InfoLevel:  []logrus.Hook{hook},
					logrus.WarnLevel:  []logrus.Hook{hook},
					logrus.ErrorLevel: []logrus.Hook{hook},
					logrus.FatalLevel: []logrus.Hook{hook},
					logrus.PanicLevel: []logrus.Hook{hook},
				},
				Level: logrus.DebugLevel,
			},
		},
	}

	svc.daemon.AddProcessor(processorName, func() backends.Decorator {
		return func(p backends.Processor) backends.Processor {
			return backends.ProcessWith(
				func(e *mail.Envelope, task backends.SelectTask) (res backends.Result, err error) {
					// OpenTelemetry: create the root span
					var span trc.Span
					ctx, span = svc.StartSpan(svc.Lifetime(), ":"+strconv.Itoa(svc.Port()), trc.Server()) // Use lifetime as parent ctx
					spanEnded := false
					defer func() {
						if !spanEnded {
							span.End()
						}
					}()

					err = errors.CatchPanic(func() error {
						res, err = svc.processEnvelope(ctx, p, e, task)
						return errors.Trace(err)
					})
					if err != nil || svc.Deployment() == connector.LOCAL {
						// Record identifying attributes only in LOCAL deployment or if there's an error.
						// Email headers are deliberately not recorded: they are attacker-influenced and
						// occasionally credential-bearing, and must never reach the span exporter
						span.SetAttributes(
							"email.subject", e.Subject,
							"email.from", e.MailFrom.String(),
						)
						span.SetClientIP(e.RemoteIP)
					}
					if err != nil {
						// OpenTelemetry: record the error, adding the request attributes
						span.SetError(err)
						svc.ForceTrace(ctx)
						span.End()
						spanEnded = true
						return backends.NewResult(fmt.Sprintf("554 Error: %s", err)), err // No trace
					}
					// OpenTelemetry: record the status code
					span.SetOK(res.Code())
					span.End()
					spanEnded = true
					return res, nil
				},
			)
		}
	})

	err = svc.daemon.Start()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// stopDaemon stops the email daemon.
func (svc *Service) stopDaemon(_ context.Context) (err error) {
	svc.daemon.Shutdown()
	svc.daemon = nil
	return nil
}

// restartDaemon refreshes the config of the email daemon.
func (svc *Service) restartDaemon(ctx context.Context) (err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.stopDaemon(ctx)
	err = svc.startDaemon(ctx)
	return errors.Trace(err)
}

// processEnvelope processes an incoming email message
func (svc *Service) processEnvelope(ctx context.Context, p backends.Processor, e *mail.Envelope, task backends.SelectTask) (backends.Result, error) {
	if task == backends.TaskSaveMail {
		startTime := time.Now()
		envelopeSize := e.Data.Len()
		var processErr error
		parsed, err := letters.ParseEmail(e.NewReader())
		if err != nil {
			processErr = errors.Trace(err)
		} else {
			svc.LogInfo(
				ctx,
				"Received email",
				"messageID", string(parsed.Headers.MessageID),
				"date", parsed.Headers.Date.UTC(),
			)
			for i := range smtpingressapi.NewMulticastTrigger(svc).OnIncomingEmail(ctx, &parsed) {
				if dispatchErr := i.Get(); dispatchErr != nil {
					svc.LogError(ctx, "Dispatching save mail event", "error", dispatchErr)
					processErr = dispatchErr
				}
			}
		}

		// Meter the inbound message
		errLabel := "OK"
		if processErr != nil {
			errLabel = "ERROR"
		}
		port := strconv.Itoa(svc.Port())
		_ = svc.RecordHistogram(
			ctx,
			"microbus_server_request_duration_seconds",
			time.Since(startTime).Seconds(),
			"port", port,
			"name", "ProcessEnvelope",
			"type", "ingress",
			"error", errLabel,
		)
		_ = svc.RecordHistogram(
			ctx,
			"microbus_server_response_body_bytes",
			float64(envelopeSize),
			"port", port,
			"name", "ProcessEnvelope",
			"type", "ingress",
			"error", errLabel,
		)
		if processErr != nil {
			return nil, processErr
		}
	}
	return p.Process(e, task)
}

/*
OnChangedPort is called when the Port config property changes.

Port is the TCP port to listen to.
*/
func (svc *Service) OnChangedPort(ctx context.Context) (err error) { // MARKER: Port
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedLogLevel is triggered when the value of the LogLevel config property changes.
func (svc *Service) OnChangedLogLevel(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

/*
OnChangedEnabled is called when the Enabled config property changes.

Enabled determines whether the email server is started.
*/
func (svc *Service) OnChangedEnabled(ctx context.Context) (err error) { // MARKER: Enabled
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

/*
OnChangedMaxSize is called when the MaxSize config property changes.

MaxSize is the maximum size of messages that will be accepted, in megabytes. Defaults to 10 megabytes.
*/
func (svc *Service) OnChangedMaxSize(ctx context.Context) (err error) { // MARKER: MaxSize
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

/*
OnChangedMaxClients is called when the MaxClients config property changes.

MaxClients controls how many client connections can be opened in parallel. Defaults to 128.
*/
func (svc *Service) OnChangedMaxClients(ctx context.Context) (err error) { // MARKER: MaxClients
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

/*
OnChangedWorkers is called when the Workers config property changes.

Workers controls how many workers process incoming mail. Defaults to 8.
*/
func (svc *Service) OnChangedWorkers(ctx context.Context) (err error) { // MARKER: Workers
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}
