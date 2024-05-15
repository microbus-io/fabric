/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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

func TestConnector_TracingCopySpan(t *testing.T) {
	t.Parallel()

	alpha := New("tracing.copy.span.connector")
	// alpha.SetDeployment(TESTINGAPP)
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
	assert.Equal(t, topSpan, goSpan)
}
