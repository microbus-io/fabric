package inbox

import (
	"github.com/microbus-io/fabric/log"
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
	fields := []log.Field{}
	for n, v := range e.Data {
		fields = append(fields, log.Any(n, v))
	}
	switch e.Level {
	case logrus.DebugLevel:
		h.svc.LogDebug(e.Context, e.Message, fields...)
	case logrus.InfoLevel:
		h.svc.LogInfo(e.Context, e.Message, fields...)
	case logrus.WarnLevel:
		h.svc.LogWarn(e.Context, e.Message, fields...)
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		h.svc.LogError(e.Context, e.Message, fields...)
	}
	return nil
}
