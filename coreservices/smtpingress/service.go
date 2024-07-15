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

package smtpingress

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/backends"
	glog "github.com/flashmob/go-guerrilla/log"
	"github.com/flashmob/go-guerrilla/mail"
	"github.com/mnako/letters"
	"github.com/sirupsen/logrus"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/smtpingress/intermediate"
	"github.com/microbus-io/fabric/coreservices/smtpingress/smtpingressapi"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/fabric/utils"
)

const processorName = "MessageProcessor"

/*
Service implements the smtp.ingress.core microservice.

The SMTP ingress microservice listens for incoming emails and fires corresponding events.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

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

	// See https://github.com/flashmob/go-guerrilla/wiki/API-&-Using-as-a-package
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
					defer span.End()

					err = utils.CatchPanic(func() error {
						res, err = svc.processEnvelope(p, e, task)
						return errors.Trace(err)
					})
					if err != nil || svc.Deployment() == connector.LOCAL {
						// The request attributes may take a lot of memory, so record only in LOCAL deployment or if there's an error
						span.SetString("email.subject", e.Subject)
						span.SetString("email.from", e.MailFrom.String())
						span.SetClientIP(e.RemoteIP)
						for k, v := range e.Header {
							span.SetStrings("email.header."+k, v)
						}
					}
					if err != nil {
						// OpenTelemetry: record the error, adding the request attributes
						span.SetError(err)
						svc.ForceTrace(ctx)
						return backends.NewResult(fmt.Sprintf("554 Error: %s", err)), err // No trace
					}
					// OpenTelemetry: record the status code
					span.SetOK(res.Code())
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
func (svc *Service) processEnvelope(p backends.Processor, e *mail.Envelope, task backends.SelectTask) (backends.Result, error) {
	if task == backends.TaskSaveMail {
		ctx := svc.Lifetime()
		parsed, err := letters.ParseEmail(e.NewReader())
		if err != nil {
			return nil, errors.Trace(err)
		}
		svc.LogInfo(
			svc.Lifetime(),
			"Received email",
			"messageID", string(parsed.Headers.MessageID),
			"date", parsed.Headers.Date.UTC(),
		)
		for i := range smtpingressapi.NewMulticastTrigger(svc).OnIncomingEmail(ctx, &parsed) {
			err := i.Get()
			if err != nil {
				svc.LogError(ctx, "Dispatching save mail event", "error", err)
			}
		}
	}
	return p.Process(e, task)
}

// OnChangedPort is triggered when the value of the Port config property changes.
func (svc *Service) OnChangedPort(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedLogLevel is triggered when the value of the LogLevel config property changes.
func (svc *Service) OnChangedLogLevel(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedEnabled is triggered when the value of the Enabled config property changes.
func (svc *Service) OnChangedEnabled(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedMaxSize is triggered when the value of the MaxSize config property changes.
func (svc *Service) OnChangedMaxSize(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedMaxClients is triggered when the value of the MaxClients config property changes.
func (svc *Service) OnChangedMaxClients(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}

// OnChangedWorkers is triggered when the value of the Workers config property changes.
func (svc *Service) OnChangedWorkers(ctx context.Context) (err error) {
	err = svc.restartDaemon(ctx)
	return errors.Trace(err)
}
