/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package yellowpages

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/exampleservices/yellowpages/yellowpagesapi"
)

var personSeq atomic.Int64

// newPerson returns a valid person with a unique email.
func newPerson() *yellowpagesapi.Person {
	n := personSeq.Add(1)
	return &yellowpagesapi.Person{
		FirstName: "Harry",
		LastName:  "Potter",
		Email:     fmt.Sprintf("person%d@hogwarts.edu", n),
		Birthday:  time.Date(1980, 7, 31, 0, 0, 0, 0, time.UTC),
	}
}

// startApp runs the microservice under test in its own app, with its own testing database, and returns a client.
func startApp(t *testing.T) yellowpagesapi.Client {
	svc := NewService()
	tester := connector.New("tester.client")
	app := application.New()
	app.Add(
		svc,
		tester,
	)
	app.RunInTest(t)
	return yellowpagesapi.NewClient(tester)
}

func TestYellowPages_Create(t *testing.T) { // MARKER: Create
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("create_and_load", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		key, err := client.Create(ctx, person)
		if !assert.NoError(err) || !assert.True(key > 0) {
			return
		}
		loaded, found, err := client.Load(ctx, key)
		if assert.NoError(err) && assert.True(found) {
			assert.Expect(
				loaded.Key, key,
				loaded.FirstName, person.FirstName,
				loaded.LastName, person.LastName,
				loaded.Email, person.Email,
			)
			assert.True(loaded.Birthday.Equal(person.Birthday))
			assert.False(loaded.CreatedAt.IsZero())
			assert.False(loaded.UpdatedAt.IsZero())
		}
	})

	t.Run("no_birthday", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		person.Birthday = time.Time{}
		key, err := client.Create(ctx, person)
		if assert.NoError(err) {
			loaded, _, err := client.Load(ctx, key)
			if assert.NoError(err) {
				assert.True(loaded.Birthday.IsZero())
			}
		}
	})

	t.Run("invalid_person", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		person.FirstName = ""
		_, err := client.Create(ctx, person)
		assert.Expect(errorStatus(err), http.StatusBadRequest)

		person = newPerson()
		person.Birthday = time.Now().Add(24 * time.Hour)
		_, err = client.Create(ctx, person)
		assert.Expect(errorStatus(err), http.StatusBadRequest)

		_, err = client.Create(ctx, nil)
		assert.Expect(errorStatus(err), http.StatusBadRequest)
	})

	t.Run("duplicate_email", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		_, err := client.Create(ctx, person)
		assert.NoError(err)
		_, err = client.Create(ctx, person)
		assert.Error(err)
	})
}

func TestYellowPages_Load(t *testing.T) { // MARKER: Load
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		person, found, err := client.Load(ctx, 999999)
		assert.Expect(
			person, (*yellowpagesapi.Person)(nil),
			found, false,
			err, nil,
		)
	})
}

func TestYellowPages_Store(t *testing.T) { // MARKER: Store
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("store_existing", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		key, err := client.Create(ctx, person)
		if !assert.NoError(err) {
			return
		}
		person.Key = key
		person.FirstName = "Hermione"
		stored, err := client.Store(ctx, person)
		if assert.NoError(err) && assert.True(stored) {
			loaded, _, err := client.Load(ctx, key)
			if assert.NoError(err) {
				assert.Expect(loaded.FirstName, "Hermione")
			}
		}
		// Storing the same values again still finds the person
		stored, err = client.Store(ctx, person)
		assert.Expect(stored, true, err, nil)
	})

	t.Run("store_missing", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		person.Key = 999999
		stored, err := client.Store(ctx, person)
		assert.Expect(stored, false, err, nil)
	})
}

func TestYellowPages_Delete(t *testing.T) { // MARKER: Delete
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("delete_once", func(t *testing.T) {
		assert := testarossa.For(t)
		key, err := client.Create(ctx, newPerson())
		if !assert.NoError(err) {
			return
		}
		deleted, err := client.Delete(ctx, key)
		assert.Expect(deleted, true, err, nil)
		deleted, err = client.Delete(ctx, key)
		assert.Expect(deleted, false, err, nil)
		_, found, err := client.Load(ctx, key)
		assert.Expect(found, false, err, nil)
	})
}

func TestYellowPages_List(t *testing.T) { // MARKER: List
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	var keys []int
	for i := range 5 {
		person := newPerson()
		person.LastName = fmt.Sprintf("Family%d", i%2)
		key, err := client.Create(ctx, person)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
	}

	t.Run("all_in_key_order", func(t *testing.T) {
		assert := testarossa.For(t)
		persons, err := client.List(ctx, yellowpagesapi.Query{})
		if assert.NoError(err) && assert.Expect(len(persons), 5) {
			for i, p := range persons {
				assert.Expect(p.Key, keys[i])
			}
		}
	})

	t.Run("filter", func(t *testing.T) {
		assert := testarossa.For(t)
		persons, err := client.List(ctx, yellowpagesapi.Query{LastName: "Family0"})
		if assert.NoError(err) {
			assert.Expect(len(persons), 3)
		}
		persons, err = client.List(ctx, yellowpagesapi.Query{LastName: "Family1", FirstName: "Nobody"})
		if assert.NoError(err) {
			assert.Expect(len(persons), 0)
		}
	})

	t.Run("pages", func(t *testing.T) {
		assert := testarossa.For(t)
		persons, err := client.List(ctx, yellowpagesapi.Query{Limit: 2, Offset: 2})
		if assert.NoError(err) && assert.Expect(len(persons), 2) {
			assert.Expect(persons[0].Key, keys[2], persons[1].Key, keys[3])
		}
	})

	t.Run("limit_too_large", func(t *testing.T) {
		assert := testarossa.For(t)
		_, err := client.List(ctx, yellowpagesapi.Query{Limit: 1001})
		assert.Expect(errorStatus(err), http.StatusBadRequest)
	})
}

func TestYellowPages_CreateREST(t *testing.T) { // MARKER: CreateREST
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("created", func(t *testing.T) {
		assert := testarossa.For(t)
		key, status, err := client.CreateREST(ctx, newPerson())
		if assert.NoError(err) {
			assert.Expect(status, http.StatusCreated)
			assert.True(key > 0)
		}
	})
}

func TestYellowPages_LoadREST(t *testing.T) { // MARKER: LoadREST
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("found_and_not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		key, err := client.Create(ctx, person)
		if !assert.NoError(err) {
			return
		}
		loaded, status, err := client.LoadREST(ctx, key)
		if assert.NoError(err) && assert.Expect(status, http.StatusOK) {
			assert.Expect(loaded.Email, person.Email)
		}
		_, status, err = client.LoadREST(ctx, 999999)
		assert.Expect(status, http.StatusNotFound, err, nil)
	})
}

func TestYellowPages_StoreREST(t *testing.T) { // MARKER: StoreREST
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("stored_and_not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		key, err := client.Create(ctx, person)
		if !assert.NoError(err) {
			return
		}
		person.LastName = "Granger"
		status, err := client.StoreREST(ctx, key, person)
		assert.Expect(status, http.StatusNoContent, err, nil)
		status, err = client.StoreREST(ctx, 999999, newPerson())
		assert.Expect(status, http.StatusNotFound, err, nil)
	})
}

func TestYellowPages_DeleteREST(t *testing.T) { // MARKER: DeleteREST
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("deleted_then_not_found", func(t *testing.T) {
		assert := testarossa.For(t)
		key, err := client.Create(ctx, newPerson())
		if !assert.NoError(err) {
			return
		}
		status, err := client.DeleteREST(ctx, key)
		assert.Expect(status, http.StatusNoContent, err, nil)
		status, err = client.DeleteREST(ctx, key)
		assert.Expect(status, http.StatusNotFound, err, nil)
	})
}

func TestYellowPages_ListREST(t *testing.T) { // MARKER: ListREST
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("by_email", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		_, err := client.Create(ctx, person)
		if !assert.NoError(err) {
			return
		}
		_, err = client.Create(ctx, newPerson())
		if !assert.NoError(err) {
			return
		}
		persons, status, err := client.ListREST(ctx, yellowpagesapi.Query{Email: person.Email})
		if assert.NoError(err) && assert.Expect(status, http.StatusOK, len(persons), 1) {
			assert.Expect(persons[0].Email, person.Email)
		}
	})

	t.Run("query_string", func(t *testing.T) {
		assert := testarossa.For(t)
		person := newPerson()
		_, err := client.Create(ctx, person)
		if !assert.NoError(err) {
			return
		}
		other := newPerson()
		_, err = client.Create(ctx, other)
		if !assert.NoError(err) {
			return
		}
		// The filter in a query argument, submitted through the demo page as its "GET by email" link does
		form := url.Values{"method": {"GET"}, "path": {"/persons?q.email=" + person.Email}}
		res, err := client.Demo(ctx, "POST", "?"+form.Encode(), nil)
		if assert.NoError(err) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Contains(string(body), ">200<")
				assert.Contains(string(body), person.Email)
				assert.NotContains(string(body), other.Email)
			}
		}
	})

	t.Run("no_match_is_empty_array", func(t *testing.T) {
		assert := testarossa.For(t)
		persons, status, err := client.ListREST(ctx, yellowpagesapi.Query{Email: "nobody@hogwarts.edu"})
		assert.Expect(len(persons), 0, status, http.StatusOK, err, nil)
	})
}

func TestYellowPages_Demo(t *testing.T) { // MARKER: Demo
	t.Parallel()
	ctx := t.Context()
	client := startApp(t)

	t.Run("renders_page", func(t *testing.T) {
		assert := testarossa.For(t)
		res, err := client.Demo(ctx, "GET", "?method=GET&path=/persons", nil)
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Contains(string(body), "Yellow Pages")
			}
		}
	})

	t.Run("submits_request", func(t *testing.T) {
		assert := testarossa.For(t)
		form := url.Values{
			"method": {"POST"},
			"path":   {"/persons"},
			"body":   {`{"firstName":"Ron","lastName":"Weasley","email":"ron@hogwarts.edu"}`},
		}
		res, err := client.Demo(ctx, "POST", "?"+form.Encode(), nil)
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Contains(string(body), "201")
			}
		}
	})
}

// errorStatus returns the HTTP status code of err, or 0 for no error.
func errorStatus(err error) int {
	if err == nil {
		return 0
	}
	return errors.StatusCode(err)
}
