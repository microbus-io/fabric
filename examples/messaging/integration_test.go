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

package messaging

import (
	"testing"
)

var (
	Svc2 *Service
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Add microservices to the testing app
	Svc2 = NewService()
	err = App.AddAndStartup(
		Svc,
		Svc2, // Replica
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
		BodyContains("NoQueue")
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
		BodyContains("DefaultQueue")
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
		BodyContains("s1-val")
	CacheStore_Get(t, ctx, "?key=s2&value=s2-val").
		BodyContains("s2-val")
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
