/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package browser

import (
	"context"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/coreservices/httpegress/httpegressapi"
	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/browser/browserapi"
	"github.com/microbus-io/fabric/examples/browser/intermediate"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *browserapi.Client
)

/*
Service implements the browser.example microservice.

The browser microservice implements a simple web browser that utilizes the egress proxy.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Browser shows a simple address bar and the source code of a URL.
*/
func (svc *Service) Browse(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	u := r.URL.Query().Get("url")
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}

	var page strings.Builder
	page.WriteString("<html><head>")
	page.WriteString("</head><body>")

	// Address bar and button
	page.WriteString(`<form method=GET action=browse>`)
	page.WriteString(`<input type=text name=url size=80 placeholder="Enter a URL" value="`)
	page.WriteString(html.EscapeString(u))
	page.WriteString(`">`)
	page.WriteString(`<input type=submit value="View Source">`)
	page.WriteString(`</form>`)

	// Source code
	if u != "" {
		resp, err := httpegressapi.NewClient(svc).Get(ctx, u)
		if err != nil {
			return errors.Trace(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Trace(err)
		}
		page.WriteString(`<pre style="white-space:pre-wrap">`)
		page.WriteString(html.EscapeString(string(body)))
		page.WriteString("</pre>")
	}

	page.WriteString("</body></html>")

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(page.String()))
	return nil
}
