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
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains)
	*/
	ctx := Context()
	Home(ctx, GET()).
		BodyContains(t, Svc.ID()).
		BodyContains(t, Svc2.ID())
}

func TestMessaging_NoQueue(t *testing.T) {
	t.Parallel()
	/*
		NoQueue(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains)
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
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains)
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
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains)
	*/
	ctx := Context()
	CacheLoad(ctx, GET(), QueryArg("key", "load")).
		BodyContains(t, "found: no")
	CacheStore(ctx, GET(), QueryArg("key", "load"), QueryArg("value", "myvalue")).
		NoError(t)
	CacheLoad(ctx, GET(), QueryArg("key", "load")).
		BodyContains(t, "found: yes").
		BodyContains(t, "myvalue")

	CacheLoad(ctx, GET()).
		Error(t, "missing")
}

func TestMessaging_CacheStore(t *testing.T) {
	t.Parallel()
	/*
		CacheStore(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains)
	*/
	ctx := Context()
	CacheStore(ctx, GET(), QueryArg("key", "store1"), QueryArg("value", "myvalue1")).
		BodyContains(t, "myvalue1").
		BodyContains(t, Svc.ID())
	CacheStore(ctx, GET(), QueryArg("key", "store2"), QueryArg("value", "myvalue2")).
		BodyContains(t, "myvalue2").
		BodyContains(t, Svc.ID())
	CacheLoad(ctx, GET(), QueryArg("key", "store1")).
		BodyContains(t, "found: yes")
	CacheLoad(ctx, GET(), QueryArg("key", "store2")).
		BodyContains(t, "found: yes")
	CacheLoad(ctx, GET(), QueryArg("key", "store3")).
		BodyContains(t, "found: no")

	CacheStore(ctx, GET()).
		Error(t, "missing")
	CacheStore(ctx, GET(), QueryArg("key", "storex")).
		Error(t, "missing")
	CacheStore(ctx, GET(), QueryArg("value", "myvaluex")).
		Error(t, "missing")
	CacheLoad(ctx, GET(), QueryArg("key", "storex")).
		BodyContains(t, "found: no")
}
