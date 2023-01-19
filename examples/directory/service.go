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
	"context"
	"database/sql"
	"net/http"

	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
	"github.com/microbus-io/fabric/examples/directory/intermediate"
)

var (
	_ context.Context
	_ *http.Request
	_ *errors.TracedError
	_ *directoryapi.Client
)

/*
Service implements the directory.example microservice.

The directory microservice stores personal records in a MySQL database.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Create registers the person in the directory.
*/
func (svc *Service) Create(ctx context.Context, person *directoryapi.Person) (created *directoryapi.Person, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this example

	err = person.Validate()
	if err != nil {
		return nil, errors.Trace(err)
	}
	res, err := shard1.ExecContext(ctx,
		`INSERT INTO directory_persons (first_name,last_name,email_address,birthday) VALUE (?,?,?,?)`,
		person.FirstName, person.LastName, person.Email, person.Birthday,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	insertID, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Trace(err)
	}
	person.Key.Seq = int(insertID)
	return person, nil
}

/*
Update updates the person's data in the directory.
*/
func (svc *Service) Update(ctx context.Context, person *directoryapi.Person) (updated *directoryapi.Person, ok bool, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this example

	err = person.Validate()
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	res, err := shard1.ExecContext(ctx,
		`UPDATE directory_persons SET first_name=?, last_name=?, email_address=?, birthday=? WHERE person_id=?`,
		person.FirstName, person.LastName, person.Email, person.Birthday, person.Key.Seq,
	)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 1 {
		return person, true, nil
	}
	// Zero may be returned if no value was updated so need to verify using load
	return svc.Load(ctx, person.Key)
}

/*
Load looks up a person in the directory.
*/
func (svc *Service) Load(ctx context.Context, key directoryapi.PersonKey) (person *directoryapi.Person, ok bool, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this example

	row := shard1.QueryRowContext(ctx,
		`SELECT first_name,last_name,email_address,birthday FROM directory_persons WHERE person_id=?`,
		key.Seq)
	person = &directoryapi.Person{
		Key: key,
	}
	err = row.Scan(&person.FirstName, &person.LastName, &person.Email, &person.Birthday)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	return person, true, nil
}

/*
Delete removes a person from the directory.
*/
func (svc *Service) Delete(ctx context.Context, key directoryapi.PersonKey) (ok bool, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this example

	res, err := shard1.ExecContext(ctx,
		`DELETE FROM directory_persons WHERE person_id=?`,
		key.Seq,
	)
	if err != nil {
		return false, errors.Trace(err)
	}
	affected, _ := res.RowsAffected()
	return affected == 1, nil
}

/*
LoadByEmail looks up a person in the directory by their email.
*/
func (svc *Service) LoadByEmail(ctx context.Context, email string) (person *directoryapi.Person, ok bool, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this simple example

	row := shard1.QueryRowContext(ctx,
		`SELECT person_id,first_name,last_name,birthday FROM directory_persons WHERE email_address=?`,
		email)
	person = &directoryapi.Person{
		Email: email,
	}
	err = row.Scan(&person.Key.Seq, &person.FirstName, &person.LastName, &person.Birthday)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	return person, true, nil
}

/*
List returns the keys of all the persons in the directory.
*/
func (svc *Service) List(ctx context.Context) (keys []directoryapi.PersonKey, err error) {
	shard1 := svc.MariaDatabase().Shard(1) // No sharding in this simple example

	rows, err := shard1.QueryContext(ctx, `SELECT person_id FROM directory_persons`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key directoryapi.PersonKey
		err = rows.Scan(&key.Seq)
		if err != nil {
			return nil, errors.Trace(err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}
