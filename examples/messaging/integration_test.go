/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
	Svc2 = NewService().(*Service)
	App.Include(
		Svc,
		Svc2,
	)

	err := App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

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
		Home(ctx, httpRequest).
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
	HomeGet(t, ctx, "").
		BodyContains(Svc.ID()).
		BodyContains(Svc2.ID())
}

func TestMessaging_NoQueue(t *testing.T) {
	t.Parallel()
	/*
		NoQueue(t, ctx, httpRequest).
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
	NoQueueGet(t, ctx, "").
		BodyContains("NoQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_DefaultQueue(t *testing.T) {
	t.Parallel()
	/*
		DefaultQueue(t, ctx, httpRequest).
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
	DefaultQueueGet(t, ctx, "").
		BodyContains("DefaultQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_CacheLoad(t *testing.T) {
	t.Parallel()
	/*
		CacheLoad(t, ctx, httpRequest).
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
	CacheLoadGet(t, ctx, "?key=l1").
		BodyContains("found: no")
	CacheStoreGet(t, ctx, "?key=l1&value=val-l1").
		NoError()
	CacheLoadGet(t, ctx, "?key=l1").
		BodyContains("found: yes").
		BodyContains("val-l1")

	CacheLoadGet(t, ctx, "").
		Error("missing")
}

func TestMessaging_CacheStore(t *testing.T) {
	t.Parallel()
	/*
		CacheStore(t, ctx, httpRequest).
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
	CacheStoreGet(t, ctx, "?key=s1&value=s1-val").
		BodyContains("s1-val").
		BodyContains(Svc.ID())
	CacheStoreGet(t, ctx, "?key=s2&value=s2-val").
		BodyContains("s2-val").
		BodyContains(Svc.ID())
	CacheLoadGet(t, ctx, "?key=s1").
		BodyContains("found: yes").
		BodyContains("s1-val")
	CacheLoadGet(t, ctx, "?key=s2").
		BodyContains("found: yes").
		BodyContains("s2-val")
	CacheLoadGet(t, ctx, "?key=s3").
		BodyContains("found: no")

	CacheStoreGet(t, ctx, "").
		Error("missing")
	CacheStoreGet(t, ctx, "?key=x").
		Error("missing")
	CacheStoreGet(t, ctx, "?value=val-x").
		Error("missing")
	CacheLoadGet(t, ctx, "?key=x").
		BodyContains("found: no")
}
