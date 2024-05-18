/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package directory

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
	"github.com/microbus-io/fabric/examples/directory/intermediate"
)

var (
	// Emulated database
	indexByKey   = map[int]*directoryapi.Person{}
	indexByEmail = map[string]*directoryapi.Person{}
	nextKey      int
	mux          sync.Mutex
)

/*
Service implements the directory.example microservice.

The directory microservice stores personal records in a SQL database.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	db *sql.DB
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	dsn := svc.SQL()
	if dsn != "" {
		svc.db, err = sql.Open("mysql", dsn)
		if err == nil {
			_, err = svc.db.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS directory_persons (
					person_id BIGINT NOT NULL AUTO_INCREMENT,
					first_name VARCHAR(32) NOT NULL,
					last_name VARCHAR(32) NOT NULL,
					email_address VARCHAR(128) CHARACTER SET ascii NOT NULL,
					birthday DATE,
					PRIMARY KEY (person_id),
					CONSTRAINT UNIQUE INDEX (email_address)
				) CHARACTER SET utf8
			`)
		}
		if err != nil {
			// The database may not have been created yet. Tolerate the error and use the emulated in-memory database.
			svc.LogWarn(ctx, "Connecting to database", log.Error(errors.Trace(err)))
			if svc.db != nil {
				svc.db.Close()
				svc.db = nil
			}
		}
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	if svc.db != nil {
		svc.db.Close()
		svc.db = nil
	}
	return nil
}

/*
Create registers the person in the directory.
*/
func (svc *Service) Create(ctx context.Context, person *directoryapi.Person) (created *directoryapi.Person, err error) {
	err = person.Validate()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		_, ok := indexByKey[person.Key.Seq]
		if ok {
			return nil, errors.New("Duplicate key")
		}
		_, ok = indexByEmail[strings.ToLower(person.Email)]
		if ok {
			return nil, errors.New("Duplicate key")
		}
		nextKey++
		person.Key = directoryapi.PersonKey{Seq: nextKey}
		indexByKey[nextKey] = person
		indexByEmail[strings.ToLower(person.Email)] = person
		return person, nil
	}

	res, err := svc.db.ExecContext(ctx,
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
	err = person.Validate()
	if err != nil {
		return nil, false, errors.Trace(err)
	}

	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		existing, ok := indexByKey[person.Key.Seq]
		if !ok {
			return nil, false, nil
		}
		delete(indexByKey, existing.Key.Seq)
		delete(indexByEmail, strings.ToLower(existing.Email))
		indexByKey[person.Key.Seq] = person
		indexByEmail[strings.ToLower(person.Email)] = person
		return person, true, nil
	}

	res, err := svc.db.ExecContext(ctx,
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
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		loaded, ok := indexByKey[key.Seq]
		return loaded, ok, nil
	}

	row := svc.db.QueryRowContext(ctx,
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
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		existing, ok := indexByKey[key.Seq]
		if !ok {
			return false, nil
		}
		delete(indexByKey, existing.Key.Seq)
		delete(indexByEmail, strings.ToLower(existing.Email))
		return true, nil
	}

	res, err := svc.db.ExecContext(ctx,
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
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		loaded, ok := indexByEmail[strings.ToLower(email)]
		return loaded, ok, nil
	}

	row := svc.db.QueryRowContext(ctx,
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
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		for _, p := range indexByKey {
			keys = append(keys, p.Key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Seq < keys[j].Seq
		})
		return keys, nil
	}

	rows, err := svc.db.QueryContext(ctx, `SELECT person_id FROM directory_persons`)
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
