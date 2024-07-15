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
	"fmt"
	"log/slog"
	"time"

	"github.com/sirupsen/logrus"
)

// logHook diverts the daemon's log entries to the service.
type logHook struct {
	svc *Service
}

// Levels indicates to intercept all levels.
func (h logHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Fire diverts the daemon's log entries to the service log.
func (h logHook) Fire(e *logrus.Entry) error {
	fields := []any{}
	for n, v := range e.Data {
		switch vv := v.(type) {
		case string:
			fields = append(fields, slog.String(n, vv))

		case int:
			fields = append(fields, slog.Int(n, int(vv)))
		case int64:
			fields = append(fields, slog.Int(n, int(vv)))
		case int32:
			fields = append(fields, slog.Int(n, int(vv)))
		case int16:
			fields = append(fields, slog.Int(n, int(vv)))
		case int8:
			fields = append(fields, slog.Int(n, int(vv)))

		case uint:
			fields = append(fields, slog.Int(n, int(vv)))
		case uint64:
			fields = append(fields, slog.Int(n, int(vv)))
		case uint32:
			fields = append(fields, slog.Int(n, int(vv)))
		case uint16:
			fields = append(fields, slog.Int(n, int(vv)))
		case uint8:
			fields = append(fields, slog.Int(n, int(vv)))

		case float64:
			fields = append(fields, slog.Float64(n, float64(vv)))
		case float32:
			fields = append(fields, slog.Float64(n, float64(vv)))

		case bool:
			fields = append(fields, slog.Bool(n, vv))

		case time.Duration:
			fields = append(fields, slog.Duration(n, vv))
		case time.Time:
			fields = append(fields, slog.Time(n, vv))

		default:
			s := fmt.Sprintf("%v", v)
			fields = append(fields, slog.String(n, s))
		}
	}

	ctx := e.Context
	if ctx == nil {
		ctx = h.svc.Lifetime()
	}

	switch e.Level {
	case logrus.DebugLevel:
		h.svc.LogDebug(ctx, e.Message, fields...)
	case logrus.InfoLevel:
		h.svc.LogInfo(ctx, e.Message, fields...)
	case logrus.WarnLevel:
		h.svc.LogWarn(ctx, e.Message, fields...)
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		h.svc.LogError(ctx, e.Message, fields...)
	}
	return nil
}
