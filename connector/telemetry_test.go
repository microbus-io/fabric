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
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/microbus-io/fabric/trc"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// spanAttributes uses reflection to look into the attributes set on a span.
func spanAttributes(span trace.Span) map[string]string {
	m := map[string]string{}
	attributes := reflect.ValueOf(span).Elem().FieldByName("attributes")
	for i := 0; i < attributes.Len(); i++ {
		k := attributes.Index(i).FieldByName("Key").String()
		v := attributes.Index(i).FieldByName("Value").FieldByName("stringly").String()
		if v == "" {
			i := attributes.Index(i).FieldByName("Value").FieldByName("numeric").Uint()
			if i != 0 {
				v = strconv.Itoa(int(i))
			}
		}
		if v == "" {
			slice := attributes.Index(i).FieldByName("Value").FieldByName("slice").Elem()
			if slice.Len() == 1 {
				v = slice.Index(0).String()
			}
		}
		m[k] = v
	}
	return m
}

// spanStatus uses reflection to look into the status of a span.
func spanStatus(span trace.Span) codes.Code {
	status := reflect.ValueOf(span).Elem().FieldByName("status")
	return codes.Code(status.FieldByName("Code").Uint())
}

func TestConnector_TraceRequestAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.test.request.attributes.connector")

	var span trace.Span
	beta := New("beta.test.request.attributes.connector")
	beta.Subscribe("GET", "handle", func(w http.ResponseWriter, r *http.Request) error {
		span = beta.Span(r.Context()).(trc.SpanImpl).Span

		// The request attributes should not be added until and unless there's an error
		attributes := spanAttributes(span)
		assert.Empty(t, attributes["http.method"])
		assert.Empty(t, attributes["url.scheme"])
		assert.Empty(t, attributes["server.address"])
		assert.Empty(t, attributes["server.port"])
		assert.Empty(t, attributes["url.path"])

		assert.Equal(t, codes.Unset, spanStatus(span))

		if r.URL.Query().Get("err") != "" {
			return errors.New("oops")
		}
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// A request that returns with an error
	_, err = alpha.GET(ctx, "https://beta.test.request.attributes.connector/handle?err=1")
	if assert.Error(t, err) {
		// The request attributes should be added since there was an error
		attributes := spanAttributes(span)
		assert.Equal(t, "GET", attributes["http.method"])
		assert.Equal(t, "https", attributes["url.scheme"])
		assert.Equal(t, "beta.test.request.attributes.connector", attributes["server.address"])
		assert.Equal(t, "443", attributes["server.port"])
		assert.Equal(t, "/handle", attributes["url.path"])

		assert.Equal(t, codes.Error, spanStatus(span))
	}

	// A request that returns OK
	_, err = alpha.GET(ctx, "https://beta.test.request.attributes.connector/handle")
	if assert.NoError(t, err) {
		// The request attributes should not be added since there was no error
		attributes := spanAttributes(span)
		assert.Empty(t, attributes["http.method"])
		assert.Empty(t, attributes["url.scheme"])
		assert.Empty(t, attributes["server.address"])
		assert.Empty(t, attributes["server.port"])
		assert.Empty(t, attributes["url.path"])

		assert.Equal(t, codes.Ok, spanStatus(span))
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
	assert.NoError(t, err)
	defer alpha.Shutdown()

	wg.Wait()
	assert.Equal(t, topSpan.TraceID(), goSpan.TraceID())
}
