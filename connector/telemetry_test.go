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

package connector

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/microbus-io/fabric/trc"
	"github.com/microbus-io/testarossa"
)

func TestConnector_TraceRequestAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.test.request.attributes.connector")

	var span trc.Span
	beta := New("beta.test.request.attributes.connector")
	beta.Subscribe("GET", "handle", func(w http.ResponseWriter, r *http.Request) error {
		span = beta.Span(r.Context())

		// The request attributes should not be added until and unless there's an error
		attributes := span.Attributes()
		testarossa.Zero(t, len(attributes["http.method"]))
		testarossa.Zero(t, len(attributes["url.scheme"]))
		testarossa.Zero(t, len(attributes["server.address"]))
		testarossa.Zero(t, len(attributes["server.port"]))
		testarossa.Zero(t, len(attributes["url.path"]))

		testarossa.Equal(t, 0, span.Status())

		if r.URL.Query().Get("err") != "" {
			return errors.New("oops")
		}
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()

	// A request that returns with an error
	_, err = alpha.GET(ctx, "https://beta.test.request.attributes.connector/handle?err=1")
	if testarossa.Error(t, err) {
		// The request attributes should be added since there was an error
		attributes := span.Attributes()
		testarossa.Equal(t, "GET", attributes["http.method"])
		testarossa.Equal(t, "https", attributes["url.scheme"])
		testarossa.Equal(t, "beta.test.request.attributes.connector", attributes["server.address"])
		testarossa.Equal(t, "443", attributes["server.port"])
		testarossa.Equal(t, "/handle", attributes["url.path"])

		testarossa.Equal(t, 1, span.Status())
	}

	// A request that returns OK
	_, err = alpha.GET(ctx, "https://beta.test.request.attributes.connector/handle")
	if testarossa.NoError(t, err) {
		// The request attributes should not be added since there was no error
		attributes := span.Attributes()
		testarossa.Zero(t, len(attributes["http.method"]))
		testarossa.Zero(t, len(attributes["url.scheme"]))
		testarossa.Zero(t, len(attributes["server.address"]))
		testarossa.Zero(t, len(attributes["server.port"]))
		testarossa.Zero(t, len(attributes["url.path"]))

		testarossa.Equal(t, 2, span.Status())
	}
}

func TestConnector_GoTracingSpan(t *testing.T) {
	t.Parallel()

	alpha := New("go.tracing.span.connector")
	var topSpan trc.Span
	var goSpan trc.Span
	var wg sync.WaitGroup
	wg.Add(1)
	alpha.SetOnStartup(func(ctx context.Context) error {
		topSpan = alpha.Span(ctx)
		alpha.Go(ctx, func(ctx context.Context) (err error) {
			goSpan = alpha.Span(ctx)
			wg.Done()
			return nil
		})
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()

	wg.Wait()
	testarossa.Equal(t, topSpan.TraceID(), goSpan.TraceID())
}
