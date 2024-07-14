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

package browser

import (
	"bufio"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/httpegress"
	"github.com/microbus-io/fabric/examples/browser/browserapi"
	"github.com/microbus-io/fabric/httpx"
)

var (
	_ *testing.T
	_ testarossa.TestingT
	_ *browserapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	err = App.AddAndStartup(
		Svc,
		httpegress.NewMock().
			MockMakeRequest(func(w http.ResponseWriter, r *http.Request) (err error) {
				req, _ := http.ReadRequest(bufio.NewReader(r.Body))
				if req.Method == "GET" && req.URL.String() == "https://mocked.example.com/" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(`<html><body>Lorem Ipsum<body></html>`))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
				return nil
			}),
	)
	if err != nil {
		return err
	}
	return nil
}

// Terminate gets called after the testing app shut down.
func Terminate() (err error) {
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
	ctx := Context()
	Browse_Get(t, ctx,
		"?"+httpx.QArgs{
			"url": "https://mocked.example.com/",
		}.Encode()).
		NoError().
		StatusOK().
		TagExists(`INPUT[name="url"][type="text"][value="https://mocked.example.com/"]`).
		TagContains(`PRE`, `<html><body>Lorem Ipsum<body></html>`).
		CompletedIn(100 * time.Millisecond)
}
