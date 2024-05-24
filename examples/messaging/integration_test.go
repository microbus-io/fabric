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
	Svc2 = NewService()
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
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	Home_Get(t, ctx, "").
		BodyContains(Svc.ID()).
		BodyContains(Svc2.ID())
}

func TestMessaging_NoQueue(t *testing.T) {
	t.Parallel()
	/*
		NoQueueGet(t, ctx, "").
			BodyContains(value).
			NoError()
		NoQueuePost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		NoQueue(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	NoQueue_Get(t, ctx, "").
		BodyContains("NoQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_DefaultQueue(t *testing.T) {
	t.Parallel()
	/*
		DefaultQueueGet(t, ctx, "").
			BodyContains(value).
			NoError()
		DefaultQueuePost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		DefaultQueue(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	DefaultQueue_Get(t, ctx, "").
		BodyContains("DefaultQueue").
		BodyContains(Svc.ID())
}

func TestMessaging_CacheLoad(t *testing.T) {
	t.Parallel()
	/*
		CacheLoadGet(t, ctx, "").
			BodyContains(value).
			NoError()
		CacheLoadPost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		CacheLoad(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	CacheLoad_Get(t, ctx, "?key=l1").
		BodyContains("found: no")
	CacheStore_Get(t, ctx, "?key=l1&value=val-l1").
		NoError()
	CacheLoad_Get(t, ctx, "?key=l1").
		BodyContains("found: yes").
		BodyContains("val-l1")

	CacheLoad_Get(t, ctx, "").
		Error("missing")
}

func TestMessaging_CacheStore(t *testing.T) {
	t.Parallel()
	/*
		CacheStoreGet(t, ctx, "").
			BodyContains(value).
			NoError()
		CacheStorePost(t, ctx, "", "", body).
			BodyContains(value).
			NoError()
		CacheStore(t, ctx, httpRequest).
			BodyContains(value).
			NoError()
	*/
	ctx := Context()
	CacheStore_Get(t, ctx, "?key=s1&value=s1-val").
		BodyContains("s1-val").
		BodyContains(Svc.ID())
	CacheStore_Get(t, ctx, "?key=s2&value=s2-val").
		BodyContains("s2-val").
		BodyContains(Svc.ID())
	CacheLoad_Get(t, ctx, "?key=s1").
		BodyContains("found: yes").
		BodyContains("s1-val")
	CacheLoad_Get(t, ctx, "?key=s2").
		BodyContains("found: yes").
		BodyContains("s2-val")
	CacheLoad_Get(t, ctx, "?key=s3").
		BodyContains("found: no")

	CacheStore_Get(t, ctx, "").
		Error("missing")
	CacheStore_Get(t, ctx, "?key=x").
		Error("missing")
	CacheStore_Get(t, ctx, "?value=val-x").
		Error("missing")
	CacheLoad_Get(t, ctx, "?key=x").
		BodyContains("found: no")
}
