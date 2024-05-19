/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package browser

import (
	"bufio"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/examples/browser/browserapi"
	"github.com/microbus-io/fabric/httpx"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *browserapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	mockEgress := httpegress.NewMock()
	mockEgress.MockMakeRequest = func(w http.ResponseWriter, r *http.Request) (err error) {
		req, _ := http.ReadRequest(bufio.NewReader(r.Body))
		if req.Method == "GET" && req.URL.String() == "https://mocked.example.com/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Lorem Ipsum<body></html>`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return nil
	}

	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc,
		mockEgress,
	)

	err = App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

	return nil
}

// Terminate shuts down the testing app.
func Terminate() (err error) {
	err = App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestBrowser_Browse(t *testing.T) {
	t.Parallel()
	/*
		BrowseGet(t, ctx, "").
			BodyContains(value).
			NoError()
		BrowsePost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		Browse(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context(t)
	BrowseGet(t, ctx, "?"+httpx.PrepareQueryString("url", "https://mocked.example.com/")).
		NoError().
		StatusOK().
		TagExists(`INPUT[name="url"][type="text"][value="https://mocked.example.com/"]`).
		TagContains(`PRE`, `<html><body>Lorem Ipsum<body></html>`).
		CompletedIn(10 * time.Millisecond)
}
