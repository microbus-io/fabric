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

package directory

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/examples/directory/directoryapi"
	"github.com/microbus-io/fabric/examples/directory/intermediate"
)

var (
	// Emulated database
	indexByKey   = map[directoryapi.PersonKey]*directoryapi.Person{}
	indexByEmail = map[string]*directoryapi.Person{}
	nextKey      int
	mux          sync.Mutex
)

/*
Service implements the directory.example microservice.

The directory microservice exposes a RESTful API for persisting personal records in a SQL database.
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
			svc.LogWarn(ctx, "Connecting to database", "error", errors.Trace(err))
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
func (svc *Service) Create(ctx context.Context, httpRequestBody *directoryapi.Person) (key directoryapi.PersonKey, err error) {
	person := httpRequestBody
	err = person.Validate()
	if err != nil {
		return 0, errors.Tracec(http.StatusBadRequest, err)
	}

	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		_, ok := indexByKey[person.Key]
		if ok {
			return 0, errors.Newc(http.StatusBadRequest, "Duplicate key")
		}
		_, ok = indexByEmail[strings.ToLower(person.Email)]
		if ok {
			return 0, errors.Newc(http.StatusBadRequest, "Duplicate key")
		}
		nextKey++
		person.Key = directoryapi.PersonKey(nextKey)
		indexByKey[person.Key] = person
		indexByEmail[strings.ToLower(person.Email)] = person
		return person.Key, nil
	}

	res, err := svc.db.ExecContext(ctx,
		`INSERT INTO directory_persons (first_name,last_name,email_address,birthday) VALUE (?,?,?,?)`,
		person.FirstName, person.LastName, person.Email, person.Birthday,
	)
	if err != nil {
		return 0, errors.Trace(err)
	}
	insertID, err := res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}
	person.Key = directoryapi.PersonKey(insertID)
	return person.Key, nil
}

/*
Update updates the person's data in the directory.
*/
func (svc *Service) Update(ctx context.Context, key directoryapi.PersonKey, httpRequestBody *directoryapi.Person) (err error) {
	person := httpRequestBody
	err = person.Validate()
	if err != nil {
		return errors.Tracec(http.StatusBadRequest, err)
	}

	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		existing, ok := indexByKey[key]
		if !ok {
			return errors.Newc(http.StatusNotFound, "")
		}
		delete(indexByKey, existing.Key)
		delete(indexByEmail, strings.ToLower(existing.Email))
		person.Key = key
		indexByKey[key] = person
		indexByEmail[strings.ToLower(person.Email)] = person
		return nil
	}

	res, err := svc.db.ExecContext(ctx,
		`UPDATE directory_persons SET first_name=?, last_name=?, email_address=?, birthday=? WHERE person_id=?`,
		person.FirstName, person.LastName, person.Email, person.Birthday, key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 1 {
		return nil
	}
	// Zero may be returned if no value was updated so need to verify using load
	_, err = svc.Load(ctx, key)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
Load looks up a person in the directory.
*/
func (svc *Service) Load(ctx context.Context, key directoryapi.PersonKey) (httpResponseBody *directoryapi.Person, err error) {
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		loaded, ok := indexByKey[key]
		if ok {
			return loaded, nil
		} else {
			return nil, errors.Newc(http.StatusNotFound, "")
		}
	}

	row := svc.db.QueryRowContext(ctx,
		`SELECT first_name,last_name,email_address,birthday FROM directory_persons WHERE person_id=?`,
		key)
	person := &directoryapi.Person{
		Key: key,
	}
	err = row.Scan(&person.FirstName, &person.LastName, &person.Email, &person.Birthday)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Newc(http.StatusNotFound, "")
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	return person, nil
}

/*
Delete removes a person from the directory.
*/
func (svc *Service) Delete(ctx context.Context, key directoryapi.PersonKey) (err error) {
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		existing, ok := indexByKey[key]
		if !ok {
			return errors.Newc(http.StatusNotFound, "")
		}
		delete(indexByKey, existing.Key)
		delete(indexByEmail, strings.ToLower(existing.Email))
		return nil
	}

	res, err := svc.db.ExecContext(ctx,
		`DELETE FROM directory_persons WHERE person_id=?`,
		key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	affected, _ := res.RowsAffected()
	if affected > 0 {
		return nil
	} else {
		return errors.Newc(http.StatusNotFound, "")
	}
}

/*
LoadByEmail looks up a person in the directory by their email.
*/
func (svc *Service) LoadByEmail(ctx context.Context, email string) (httpResponseBody *directoryapi.Person, err error) {
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		loaded, ok := indexByEmail[strings.ToLower(email)]
		if ok {
			return loaded, nil
		} else {
			return nil, errors.Newc(http.StatusNotFound, "")
		}
	}

	row := svc.db.QueryRowContext(ctx,
		`SELECT person_id,first_name,last_name,birthday FROM directory_persons WHERE email_address=?`,
		email)
	person := &directoryapi.Person{
		Email: email,
	}
	err = row.Scan(&person.Key, &person.FirstName, &person.LastName, &person.Birthday)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Newc(http.StatusNotFound, "")
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	return person, nil
}

/*
List returns the keys of all the persons in the directory.
*/
func (svc *Service) List(ctx context.Context) (httpResponseBody []directoryapi.PersonKey, err error) {
	if svc.db == nil {
		// Emulate a database in-memory
		mux.Lock()
		defer mux.Unlock()
		for _, p := range indexByKey {
			httpResponseBody = append(httpResponseBody, p.Key)
		}
		sort.Slice(httpResponseBody, func(i, j int) bool {
			return httpResponseBody[i] < httpResponseBody[j]
		})
		return httpResponseBody, nil
	}

	rows, err := svc.db.QueryContext(ctx, `SELECT person_id FROM directory_persons`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key directoryapi.PersonKey
		err = rows.Scan(&key)
		if err != nil {
			return nil, errors.Trace(err)
		}
		httpResponseBody = append(httpResponseBody, key)
	}
	return httpResponseBody, nil
}

/*
WebUI provides a form for making web requests to the CRUD endpoints.
*/
func (svc *Service) WebUI(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	err = r.ParseForm()
	if err != nil {
		return errors.Tracec(http.StatusBadRequest, err)
	}

	data := struct {
		Method     string
		Path       string
		Body       string
		StatusCode int
		Response   string
	}{
		Method: r.FormValue("method"),
		Path:   r.FormValue("path"),
		Body:   r.FormValue("body"),
	}
	if r.Method == "POST" {
		method := r.FormValue("method")
		u, err := url.JoinPath("https://"+Hostname, r.FormValue("path"))
		if err != nil {
			return errors.Trace(err)
		}
		var body []byte
		contentType := ""
		if method == "POST" || method == "PUT" {
			body = []byte(r.FormValue("body"))
			contentType = "application/json"
		}
		res, err := svc.Request(
			ctx,
			pub.Method(method),
			pub.URL(u),
			pub.Body(body),
			pub.ContentType(contentType),
		)
		if err != nil {
			data.Response = fmt.Sprintf("%+v", err)
			data.StatusCode = errors.StatusCode(err)
		} else {
			data.StatusCode = res.StatusCode
			b, err := io.ReadAll(res.Body)
			if err != nil {
				return errors.Trace(err)
			}
			data.Response = string(b)
		}
	}
	output, err := svc.ExecuteResTemplate("webui.html", data)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(output))
	return nil
}
