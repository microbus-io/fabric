/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package browser

import (
	"bufio"
	"html"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/examples/browser/browserapi"
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
		if req.Method == "GET" && req.URL.String() == "https://example.com/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Lorem Ipsum<body></html>`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return nil
	}

	// Include all downstream microservices in the testing app
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
		Browse(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			ErrorCode(http.StatusOK).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Browse(t, ctx, GET(), QueryArg("url", "https://example.com/")).
		StatusOK().
		StatusCode(http.StatusOK).
		BodyContains(`"https://example.com/"`).
		BodyContains(html.EscapeString(`<html><body>Lorem Ipsum<body></html>`)).
		NoError()
}
