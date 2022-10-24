package service

import (
	"context"
	"net/http"
)

// HTTPHandler extends the standard http.Handler to also return an error
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// StartupHandler handles the OnStartup callback.
type StartupHandler func(ctx context.Context) error

// StartupHandler handles the OnShutdown callback.
type ShutdownHandler func(ctx context.Context) error

// StartupHandler handles the OnStartup callback.
type ConfigChangedHandler func(ctx context.Context, changed func(string) bool) error

// TickerHandler handles the ticker callbacks.
type TickerHandler func(ctx context.Context) error
