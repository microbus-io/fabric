/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

package directory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/examples/directory/directoryapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *directoryapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc,
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

func TestDirectory_CRUD(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	person := &directoryapi.Person{
		FirstName: "Harry",
		LastName:  "Potter",
		Email:     "harry.potter@hogwarts.edu.wiz",
		Birthday:  clock.MustParseNullTimeUTC("", "1980-07-31"),
	}
	Create(ctx, person).
		Assert(t, func(t *testing.T, created *directoryapi.Person, err error) {
			assert.Equal(t, person.FirstName, created.FirstName)
			assert.Equal(t, person.LastName, created.LastName)
			assert.Equal(t, person.Email, created.Email)
			assert.NotZero(t, created.Key.Seq)
		})

	Load(ctx, person.Key).
		Expect(t, person, true)
	LoadByEmail(ctx, person.Email).
		Expect(t, person, true)
	List(ctx).
		Assert(t, func(t *testing.T, keys []directoryapi.PersonKey, err error) {
			assert.Contains(t, keys, person.Key)
		})

	person.Email = "harry.potter@gryffindor.wiz"
	Update(ctx, person).
		NoError(t)

	Load(ctx, person.Key).
		Expect(t, person, true).
		Assert(t, func(t *testing.T, person *directoryapi.Person, ok bool, err error) {
			assert.Equal(t, "harry.potter@gryffindor.wiz", person.Email)
		})
	LoadByEmail(ctx, person.Email).
		Expect(t, person, true)

	dupPerson := &directoryapi.Person{
		FirstName: "Harry",
		LastName:  "Potter",
		Email:     "harry.potter@gryffindor.wiz",
	}
	Create(ctx, dupPerson).
		Error(t, "")

	Delete(ctx, person.Key).
		NoError(t)

	Load(ctx, person.Key).
		Expect(t, nil, false)
	LoadByEmail(ctx, person.Email).
		Expect(t, nil, false)
}

func TestDirectory_Create(t *testing.T) {
	t.Parallel()

	ctx := Context(t)

	person := &directoryapi.Person{
		FirstName: "",
		LastName:  "Riddle",
		Email:     "tom.riddle@hogwarts.edu.wiz",
		Birthday:  clock.MustParseNullTimeUTC("", "1926-12-31"),
	}
	Create(ctx, person).
		Error(t, "empty")
	person.FirstName = "Tom"

	person.LastName = ""
	Create(ctx, person).
		Error(t, "empty")
	person.LastName = "Riddle"

	person.Email = ""
	Create(ctx, person).
		Error(t, "empty")
	person.Email = "tom.riddle@hogwarts.edu.wiz"

	person.Birthday = clock.NewNullTime(time.Now().AddDate(1, 0, 0))
	Create(ctx, person).
		Error(t, "birthday")
	person.Birthday = clock.NullTime{}

	Create(ctx, person).
		Assert(t, func(t *testing.T, created *directoryapi.Person, err error) {
			assert.Equal(t, person.FirstName, created.FirstName)
			assert.Equal(t, person.LastName, created.LastName)
			assert.Equal(t, person.Email, created.Email)
			assert.NotZero(t, created.Key.Seq)
		})
	List(ctx).
		Assert(t, func(t *testing.T, keys []directoryapi.PersonKey, err error) {
			assert.Contains(t, keys, person.Key)
		})

	Create(ctx, person).
		Error(t, "Duplicate")
}

func TestDirectory_Load(t *testing.T) {
	t.Skip() // Tested elsewhere
}

func TestDirectory_Update(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	person := &directoryapi.Person{
		FirstName: "Ron",
		LastName:  "Weasley",
		Email:     "ron.weasley@hogwarts.edu.wiz",
	}
	Create(ctx, person).
		NoError(t)

	person.FirstName = ""
	Update(ctx, person).
		Error(t, "empty")
	person.FirstName = "Ron"

	person.LastName = ""
	Update(ctx, person).
		Error(t, "empty")
	person.LastName = "Weasley"

	person.Email = ""
	Update(ctx, person).
		Error(t, "empty")
	person.Email = "ron.weasley@hogwarts.edu.wiz"

	person.Birthday = clock.NewNullTime(time.Now().AddDate(1, 0, 0))
	Create(ctx, person).
		Error(t, "birthday")
	person.Birthday = clock.NullTime{}

	person.Birthday = clock.MustParseNullTimeUTC("", "1980-03-01")
	Update(ctx, person).
		NoError(t)

	Load(ctx, person.Key).
		Expect(t, person, true)
}

func TestDirectory_LoadByEmail(t *testing.T) {
	t.Skip() // Tested elsewhere
}

func TestDirectory_Delete(t *testing.T) {
	t.Skip() // Tested elsewhere
}

func TestDirectory_List(t *testing.T) {
	t.Skip() // Tested elsewhere
}
