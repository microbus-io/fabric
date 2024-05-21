/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package directory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
	"github.com/microbus-io/fabric/timex"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *directoryapi.Client
)

// Initialize starts up the testing app.
func Initialize() (err error) {
	// Include all downstream microservices in the testing app
	App.Include(
		Svc,
	)

	err = App.Startup()
	if err != nil {
		return err
	}
	// All microservices are now running

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

func TestDirectory_CRUD(t *testing.T) {
	t.Parallel()

	ctx := Context(t)
	person := &directoryapi.Person{
		FirstName: "Harry",
		LastName:  "Potter",
		Email:     "harry.potter@hogwarts.edu.wiz",
		Birthday:  timex.MustParse("", "1980-07-31").UTC(),
	}
	Create(t, ctx, person).
		Assert(func(t *testing.T, created *directoryapi.Person, err error) {
			assert.Equal(t, person.FirstName, created.FirstName)
			assert.Equal(t, person.LastName, created.LastName)
			assert.Equal(t, person.Email, created.Email)
			assert.NotZero(t, created.Key.Seq)
		})

	Load(t, ctx, person.Key).
		Expect(person, true)
	LoadByEmail(t, ctx, person.Email).
		Expect(person, true)
	List(t, ctx).
		Assert(func(t *testing.T, keys []directoryapi.PersonKey, err error) {
			assert.Contains(t, keys, person.Key)
		})

	person.Email = "harry.potter@gryffindor.wiz"
	Update(t, ctx, person).
		NoError()

	Load(t, ctx, person.Key).
		Expect(person, true).
		Assert(func(t *testing.T, person *directoryapi.Person, ok bool, err error) {
			assert.Equal(t, "harry.potter@gryffindor.wiz", person.Email)
		})
	LoadByEmail(t, ctx, person.Email).
		Expect(person, true)

	dupPerson := &directoryapi.Person{
		FirstName: "Harry",
		LastName:  "Potter",
		Email:     "harry.potter@gryffindor.wiz",
	}
	Create(t, ctx, dupPerson).
		Error("")

	Delete(t, ctx, person.Key).
		NoError()

	Load(t, ctx, person.Key).
		Expect(nil, false)
	LoadByEmail(t, ctx, person.Email).
		Expect(nil, false)
}

func TestDirectory_Create(t *testing.T) {
	t.Parallel()

	ctx := Context(t)

	person := &directoryapi.Person{
		FirstName: "",
		LastName:  "Riddle",
		Email:     "tom.riddle@hogwarts.edu.wiz",
		Birthday:  timex.MustParse("", "1926-12-31").UTC(),
	}
	Create(t, ctx, person).
		Error("empty")
	person.FirstName = "Tom"

	person.LastName = ""
	Create(t, ctx, person).
		Error("empty")
	person.LastName = "Riddle"

	person.Email = ""
	Create(t, ctx, person).
		Error("empty")
	person.Email = "tom.riddle@hogwarts.edu.wiz"

	person.Birthday = timex.New(time.Now().AddDate(1, 0, 0))
	Create(t, ctx, person).
		Error("birthday")
	person.Birthday = timex.Timex{}

	person, _ = Create(t, ctx, person).
		Assert(func(t *testing.T, created *directoryapi.Person, err error) {
			assert.Equal(t, person.FirstName, created.FirstName)
			assert.Equal(t, person.LastName, created.LastName)
			assert.Equal(t, person.Email, created.Email)
			assert.NotZero(t, created.Key.Seq)
		}).
		Get()
	List(t, ctx).
		Assert(func(t *testing.T, keys []directoryapi.PersonKey, err error) {
			assert.Contains(t, keys, person.Key)
		})

	Create(t, ctx, person).
		Error("Duplicate")
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
	Create(t, ctx, person).
		NoError()

	person.FirstName = ""
	Update(t, ctx, person).
		Error("empty")
	person.FirstName = "Ron"

	person.LastName = ""
	Update(t, ctx, person).
		Error("empty")
	person.LastName = "Weasley"

	person.Email = ""
	Update(t, ctx, person).
		Error("empty")
	person.Email = "ron.weasley@hogwarts.edu.wiz"

	person.Birthday = timex.New(time.Now().AddDate(1, 0, 0))
	Create(t, ctx, person).
		Error("birthday")
	person.Birthday = timex.Timex{}

	person.Birthday = timex.MustParse("", "1980-03-01").UTC()
	Update(t, ctx, person).
		NoError()

	Load(t, ctx, person.Key).
		Expect(person, true)
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
