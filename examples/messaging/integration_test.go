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
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context()
	Home(ctx, GET()).
		Name("all replicas").
		BodyContains(t, Svc.ID()).
		BodyContains(t, Svc2.ID())
}

func TestMessaging_NoQueue(t *testing.T) {
	t.Parallel()
	/*
		NoQueue(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context()
	NoQueue(ctx, GET()).
		BodyContains(t, "NoQueue").
		BodyContains(t, Svc.ID())
}

func TestMessaging_DefaultQueue(t *testing.T) {
	t.Parallel()
	/*
		DefaultQueue(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context()
	DefaultQueue(ctx, GET()).
		BodyContains(t, "DefaultQueue").
		BodyContains(t, Svc.ID())
}

func TestMessaging_CacheLoad(t *testing.T) {
	t.Parallel()
	/*
		CacheLoad(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context()
	CacheLoad(ctx, GET(), QueryArg("key", "l1")).
		Name("load not found").
		BodyContains(t, "found: no")
	CacheStore(ctx, GET(), QueryArg("key", "l1"), QueryArg("value", "val-l1")).
		Name("store l1").
		NoError(t)
	CacheLoad(ctx, GET(), QueryArg("key", "l1")).
		Name("load found").
		BodyContains(t, "found: yes").
		BodyContains(t, "val-l1")

	CacheLoad(ctx, GET()).
		Name("no key arg on load").
		Error(t, "missing")
}

func TestMessaging_CacheStore(t *testing.T) {
	t.Parallel()
	/*
		CacheStore(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	ctx := Context()
	CacheStore(ctx, GET(), QueryArg("key", "s1"), QueryArg("value", "s1-val")).
		Name("store s1").
		BodyContains(t, "s1-val").
		BodyContains(t, Svc.ID())
	CacheStore(ctx, GET(), QueryArg("key", "s2"), QueryArg("value", "s2-val")).
		Name("store s2").
		BodyContains(t, "s2-val").
		BodyContains(t, Svc.ID())
	CacheLoad(ctx, GET(), QueryArg("key", "s1")).
		Name("load s1").
		BodyContains(t, "found: yes").
		BodyContains(t, "s1-val")
	CacheLoad(ctx, GET(), QueryArg("key", "s2")).
		Name("load s2").
		BodyContains(t, "found: yes").
		BodyContains(t, "s2-val")
	CacheLoad(ctx, GET(), QueryArg("key", "s3")).
		Name("load s3").
		BodyContains(t, "found: no")

	CacheStore(ctx, GET()).
		Name("no key and value args on store").
		Error(t, "missing")
	CacheStore(ctx, GET(), QueryArg("key", "x")).
		Name("no key arg on store").
		Error(t, "missing")
	CacheStore(ctx, GET(), QueryArg("value", "val-x")).
		Name("no value arg on store").
		Error(t, "missing")
	CacheLoad(ctx, GET(), QueryArg("key", "x")).
		Name("no x").
		BodyContains(t, "found: no")
}
