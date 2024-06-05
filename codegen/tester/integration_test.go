/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package tester

import (
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/service"

	"github.com/microbus-io/fabric/codegen/tester/testerapi"
)

var (
	openAPI map[string]any
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	App.Init(func(svc service.Service) {
		// Initialize all microservices
	})

	// Include all downstream microservices in the testing app
	App.Include(
		Svc.Init(func(svc *Service) {
			// Initialize the microservice under test
		}),
		// downstream.NewService().Init(func(svc *downstream.Service) {}),
	)

	err = App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

	ctx := Context()
	res, err := Svc.Request(ctx, pub.GET("https://"+Hostname+"/openapi.json"))
	if err != nil {
		return err
	}
	err = json.NewDecoder(res.Body).Decode(&openAPI)
	if err != nil {
		return err
	}
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

func TestTester_StringCut(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		StringCut(t, ctx, s, sep).
			Expect(before, after, found)
	*/

	ctx := Context()
	StringCut(t, ctx, "Hello World", " ").
		Expect("Hello", "World", true)
	StringCut(t, ctx, "Hello World", "X").
		Expect("Hello World", "", false)
}

func TestTester_PointDistance(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		PointDistance(t, ctx, p1, p2).
			Expect(d)
	*/

	ctx := Context()
	PointDistance(t, ctx, testerapi.XYCoord{X: 1, Y: 1}, testerapi.XYCoord{X: 4, Y: 5}).
		Expect(5)
	PointDistance(t, ctx, testerapi.XYCoord{X: 4, Y: 5}, testerapi.XYCoord{X: 1, Y: 1}).
		Expect(5)
	PointDistance(t, ctx, testerapi.XYCoord{X: 1.5, Y: 1.6}, testerapi.XYCoord{X: 2.5, Y: 2.6}).
		Assert(func(t *testing.T, d float64, err error) {
			assert.InDelta(t, math.Sqrt(2.0), d, 0.01)
		})
	PointDistance(t, ctx, testerapi.XYCoord{X: 6.1, Y: 7.6}, testerapi.XYCoord{X: 6.1, Y: 7.6}).
		Expect(0)
}

func TestTester_SubArrayRange(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		SubArrayRange(t, ctx, httpRequestBody, min, max).
			Expect(httpResponseBody, sum)
	*/

	ctx := Context()
	SubArrayRange(t, ctx, []int{1, 2, 3, 4, 5, 6}, 2, 4).
		Expect([]int{2, 3, 4}, 9, http.StatusAccepted) // Sum is returned because calling directly

	sub, sum, status, err := testerapi.NewClient(Svc).SubArrayRange(ctx, []int{1, 2, 3, 4, 5, 6}, 2, 4)
	if assert.NoError(t, err) {
		assert.Equal(t, sub, []int{2, 3, 4})
		assert.Equal(t, 0, sum) // Sum cannot be returned because httpResponseBody is present
		assert.Equal(t, http.StatusAccepted, status)
	}

	// OpenAPI
	basePath := "paths|/" + Hostname + ":443/sub-array-range/{max}|post|"
	// Argument pushed to query because of httpRequestBody
	assert.Equal(t, "min", openAPIValue(basePath+"parameters|0|name"))
	assert.Equal(t, "query", openAPIValue(basePath+"parameters|0|in"))
	// Argument indicated in path
	assert.Equal(t, "max", openAPIValue(basePath+"parameters|1|name"))
	assert.Equal(t, "path", openAPIValue(basePath+"parameters|1|in"))
	// httpRequestBody should not be listed as an argument
	assert.Len(t, openAPIValue(basePath+"parameters"), 2)
	// Request schema is an array
	schemaRef := openAPIValue(basePath + "requestBody|content|application/json|schema|$ref").(string)
	schemaRef = strings.ReplaceAll(schemaRef, "/", "|")[2:] + "|"
	assert.Equal(t, "array", openAPIValue(schemaRef+"type"))
	assert.Equal(t, "integer", openAPIValue(schemaRef+"items|type"))
	// Response schema is an array
	schemaRef = openAPIValue(basePath + "responses|200|content|application/json|schema|$ref").(string)
	schemaRef = strings.ReplaceAll(schemaRef, "/", "|")[2:] + "|"
	assert.Equal(t, "array", openAPIValue(schemaRef+"type"))
	assert.Equal(t, "integer", openAPIValue(schemaRef+"items|type"))
}

func openAPIValue(path string) any {
	var at any
	at = openAPI
	parts := strings.Split(path, "|")
	for i := range parts {
		var next any
		if m, ok := at.(map[string]any); ok {
			next = m[parts[i]]
		}
		if a, ok := at.([]any); ok {
			i, _ := strconv.Atoi(parts[i])
			next = a[i]
		}
		if i == len(parts)-1 {
			return next
		}
		if next == nil {
			return nil
		}
		at = next
	}
	return nil
}

func TestTester_WebPathArguments(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		httpReq, _ := http.NewRequestWithContext(ctx, method, "?arg=val", body)
		WebPathArguments_Get(t, ctx, "").BodyContains(value)
		WebPathArguments_Post(t, ctx, "", "", body).BodyContains(value)
		WebPathArguments(t, httpRequest).BodyContains(value)
	*/

	ctx := Context()
	WebPathArguments_Get(t, ctx, "?named=1&path2=2&suffix=3/4").
		BodyContains("/fixed/1/2/3/4$").
		BodyNotContains("?").
		BodyNotContains("{").
		BodyNotContains("}")
	WebPathArguments_Get(t, ctx, "?named=1&path2=2&suffix=3/4&q=5").
		BodyContains("/fixed/1/2/3/4?q=5$").
		BodyNotContains("&").
		BodyNotContains("{").
		BodyNotContains("}")
	WebPathArguments_Get(t, ctx, "").
		BodyContains("/fixed///$").
		BodyNotContains("?").
		BodyNotContains("&").
		BodyNotContains("{").
		BodyNotContains("}")
	WebPathArguments_Get(t, ctx, "?named="+url.QueryEscape("[a&b/c]")+"&path2="+url.QueryEscape("[d&e/f]")+"&suffix="+url.QueryEscape("[g&h/i]")+"&q="+url.QueryEscape("[j&k/l]")).
		BodyContains("/fixed/" + url.PathEscape("[a&b/c]") + "/" + url.PathEscape("[d&e/f]") + "/" + url.PathEscape("[g&h") + "/" + url.PathEscape("i]") + "?q=" + url.QueryEscape("[j&k/l]")).
		BodyNotContains("{").
		BodyNotContains("}")
}

func TestTester_FunctionPathArguments(t *testing.T) {
	t.Parallel()
	/*
		ctx := Context()
		FunctionPathArguments(t, ctx, named, path2, suffix).
			Expect(joined)
	*/

	ctx := Context()
	FunctionPathArguments(t, ctx, "1", "2", "3/4").
		Expect("1 2 3/4")
	FunctionPathArguments(t, ctx, "", "", "").
		Expect("  ")
	FunctionPathArguments(t, ctx, "[a&b/c]", "[d&e/f]", "[g&h/i]").
		Expect("[a&b/c] [d&e/f] [g&h/i]")
}
