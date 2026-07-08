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

package httpingress

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/httpingress/middleware"
)

// Middleware names
const (
	CharsetUTF8     = "CharsetUTF8"
	ErrorPrinter    = "ErrorPrinter"
	BlockedPaths    = "BlockedPaths"
	Logger          = "Logger"
	Enter           = "Enter"
	SecureRedirect  = "SecureRedirect"
	CORS            = "CORS"
	XForwarded      = "XForwarded"
	InternalHeaders = "InternalHeaders"
	RootPath        = "RootPath"
	Timeout         = "Timeout"
	Authorization   = "Authorization"
	Ready           = "Ready"
	CacheControl    = "CacheControl"
	Compress        = "Compress"
	DefaultFavIcon  = "DefaultFavIcon"
)

// defaultMiddleware prepares the default middleware of the ingress proxy.
func (svc *Service) defaultMiddleware() *middleware.Chain {
	// Warning: renaming or removing middleware is a breaking change because the names are used as location markers
	m := &middleware.Chain{}
	m.Append(CharsetUTF8, middleware.CharsetUTF8())
	m.Append(ErrorPrinter, middleware.ErrorPrinter(func() bool {
		// Redact in every deployed mode; only the developer deployments (LOCAL, TESTING) print full errors.
		d := svc.Deployment()
		return d != connector.LOCAL && d != connector.TESTING
	}))
	m.Append(BlockedPaths, middleware.BlockedPaths(func(path string) bool {
		if svc.blockedPaths[path] {
			return true
		}
		p := path
		for p != "" {
			if svc.blockedPaths[p+"/*"] {
				return true
			}
			slash := strings.LastIndex(p, "/")
			if slash >= 0 {
				p = p[:slash]
			} else {
				p = ""
			}
		}
		dot := strings.LastIndex(path, ".")
		if dot >= 0 && svc.blockedPaths["*"+path[dot:]] {
			return true
		}
		return false
	}))
	m.Append(Logger, middleware.Logger(svc))
	m.Append(Enter, middleware.NoOp()) // Marker
	m.Append(SecureRedirect, middleware.SecureRedirect(func() bool {
		return svc.secure443
	}))
	m.Append(CORS, middleware.Cors(func(r *http.Request, origin string) (allowed string, credentialed bool) {
		if svc.credentialedOrigins[origin] {
			return origin, true
		}
		if svc.uncredentialedOrigins["*"] {
			// The literal * allows any origin without credentials
			return "*", false
		}
		if svc.uncredentialedOrigins[origin] {
			return origin, false
		}
		if len(svc.credentialedOrigins) == 0 && len(svc.uncredentialedOrigins) == 0 {
			// No allowlist configured: permit only same-origin reads by pinning
			// ACAO to the request's own scheme://host. The browser then rejects
			// cross-origin reads because the reflected ACAO won't match the
			// caller's Origin. X-Forwarded-* headers are deliberately ignored
			// since they are attacker-controlled at the edge.
			return requestSameOrigin(r), false
		}
		return "", false
	}))
	m.Append(XForwarded, middleware.XForwarded())
	m.Append(InternalHeaders, middleware.InternalHeaders())
	m.Append(RootPath, middleware.RootPath("/root"))
	m.Append(Timeout, middleware.Timeout(func() time.Duration {
		return svc.TimeBudget()
	}))
	m.Append(Authorization, middleware.Authorization(func(ctx context.Context, bearerToken string) (accessToken string, err error) {
		accessToken, err = svc.exchangeToken(ctx, bearerToken)
		return accessToken, errors.Trace(err)
	}))
	m.Append(Ready, middleware.NoOp()) // Marker
	m.Append(CacheControl, middleware.CacheControl("no-cache, no-store, max-age=0"))
	m.Append(Compress, middleware.Compress())
	m.Append(DefaultFavIcon, middleware.DefaultFavIcon())

	return m
}

// requestSameOrigin returns scheme://host derived directly from r, never from
// X-Forwarded-* headers. r.Host carries host[:port] exactly as the browser
// uses it when constructing the Origin header, so the returned value matches
// only same-origin callers.
func requestSameOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
