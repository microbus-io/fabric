/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package messaging

import (
	"testing"
)

var (
	Svc2 *Service
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	Svc2 = NewService().(*Service)
	App.Include(
		Svc,
		Svc2,
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestMessaging_Home(t *testing.T) {
	t.Parallel()
	/*
		Home(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	Home(t, ctx, GET()).
		Name("all replicas").
		BodyContains(Svc.ID()).
		BodyContains(Svc2.ID())
}

func TestMessaging_NoQueue(t *testing.T) {
	t.Parallel()
	/*
		NoQueue(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	NoQueue(t, ctx, GET()).
		BodyContains("NoQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_DefaultQueue(t *testing.T) {
	t.Parallel()
	/*
		DefaultQueue(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	DefaultQueue(t, ctx, GET()).
		BodyContains("DefaultQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_CacheLoad(t *testing.T) {
	t.Parallel()
	/*
		CacheLoad(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	CacheLoad(t, ctx, GET(), QueryArg("key", "l1")).
		Name("load not found").
		BodyContains("found: no")
	CacheStore(t, ctx, GET(), QueryArg("key", "l1"), QueryArg("value", "val-l1")).
		Name("store l1").
		NoError()
	CacheLoad(t, ctx, GET(), QueryArg("key", "l1")).
		Name("load found").
		BodyContains("found: yes").
		BodyContains("val-l1")

	CacheLoad(t, ctx, GET()).
		Name("no key arg on load").
		Error("missing")
}

func TestMessaging_CacheStore(t *testing.T) {
	t.Parallel()
	/*
		CacheStore(t, ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK().
			StatusCode(statusCode).
			BodyContains(bodyContains).
			BodyNotContains(bodyNotContains).
			HeaderContains(headerName, valueContains).
			NoError().
			Error(errContains).
			Assert(func(t, httpResponse, err))
	*/
	ctx := Context(t)
	CacheStore(t, ctx, GET(), QueryArg("key", "s1"), QueryArg("value", "s1-val")).
		Name("store s1").
		BodyContains("s1-val").
		BodyContains(Svc.ID())
	CacheStore(t, ctx, GET(), QueryArg("key", "s2"), QueryArg("value", "s2-val")).
		Name("store s2").
		BodyContains("s2-val").
		BodyContains(Svc.ID())
	CacheLoad(t, ctx, GET(), QueryArg("key", "s1")).
		Name("load s1").
		BodyContains("found: yes").
		BodyContains("s1-val")
	CacheLoad(t, ctx, GET(), QueryArg("key", "s2")).
		Name("load s2").
		BodyContains("found: yes").
		BodyContains("s2-val")
	CacheLoad(t, ctx, GET(), QueryArg("key", "s3")).
		Name("load s3").
		BodyContains("found: no")

	CacheStore(t, ctx, GET()).
		Name("no key and value args on store").
		Error("missing")
	CacheStore(t, ctx, GET(), QueryArg("key", "x")).
		Name("no key arg on store").
		Error("missing")
	CacheStore(t, ctx, GET(), QueryArg("value", "val-x")).
		Name("no value arg on store").
		Error("missing")
	CacheLoad(t, ctx, GET(), QueryArg("key", "x")).
		Name("no x").
		BodyContains("found: no")
}
