/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpingress

import (
	"log"
	"strings"
)

// logWriter captures logs generated by the HTTP servers and directs them to the service's logger.
type logWriter struct {
	svc *Service
}

// Write sends the output to the service's logger.
func (lw *logWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if !strings.Contains(msg, "TLS handshake error") {
		lw.svc.LogError(lw.svc.Lifetime(), msg)
	}
	return len(p), nil
}

// newHTTPLogger creates a logger that redirects output to the service's logger.
func newHTTPLogger(svc *Service) *log.Logger {
	return log.New(&logWriter{svc}, "", 0)
}
