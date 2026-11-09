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
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/sequel"

	"github.com/microbus-io/fabric/exampleservices/yellowpages/yellowpagesapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ yellowpagesapi.Client
)

const (
	sequenceName = "yellowpages.persons" // Do not change: it names the migrations already applied to a database
	defaultLimit = 100
)

// personColumns are the columns of the persons table, in the order scanPerson reads them.
const personColumns = "id, first_name, last_name, email, birthday, created_at, updated_at"

/*
Service implements the yellowpages.example microservice.

YellowPages persists persons in a SQL database.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	db *sequel.DB
}

// OnStartup is called when the microservice is started up. It opens the database and applies its migrations.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	dataSourceName := svc.SQLDataSourceName()
	if dataSourceName == "" && svc.Deployment() == connector.LOCAL {
		dataSourceName = "file:local.sqlite"
	}
	if svc.Deployment() == connector.TESTING {
		// An isolated database per test, shared by the replicas of one test app
		dataSourceName, err = sequel.CreateTestingDatabase("", dataSourceName, svc.Plane())
		if err != nil {
			return errors.Trace(err)
		}
	}
	svc.db, err = sequel.OpenSingleton("", dataSourceName) // The driver is inferred from the data source name
	if err != nil {
		return errors.Trace(err)
	}
	// Route sequel's spans, metrics and migration logs through the connector
	svc.db.SetTracerProvider(svc.TracerProvider())
	svc.db.SetMeterProvider(svc.MeterProvider())
	svc.db.SetLogger(svc.Logger())
	migrations, err := fs.Sub(svc.ResFS(), "sql")
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.db.Migrate(sequenceName, migrations)
	return errors.Trace(err)
}

// OnShutdown is called when the microservice is shut down. It closes the database.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	if svc.db != nil {
		err = svc.db.Close()
	}
	return errors.Trace(err)
}

// scanPerson reads a row of personColumns into a person.
func scanPerson(scan func(dest ...any) error) (*yellowpagesapi.Person, error) {
	var p yellowpagesapi.Person
	args := []any{
		&p.Key,
		&p.FirstName,
		&p.LastName,
		&p.Email,
		sequel.Nullable(&p.Birthday),
		&p.CreatedAt,
		&p.UpdatedAt,
	}
	err := scan(args...)
	if err != nil {
		return nil, err // No trace: the caller checks for sql.ErrNoRows
	}
	err = sequel.ApplyBindings(args...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &p, nil
}

/*
Create creates a new person, returning its key.
*/
func (svc *Service) Create(ctx context.Context, person *yellowpagesapi.Person) (key int, err error) { // MARKER: Create
	if person == nil {
		return 0, errors.New("person is required", http.StatusBadRequest)
	}
	id, err := svc.db.InsertReturnID(ctx, "id",
		"INSERT INTO persons (first_name, last_name, email, birthday, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, NOW_UTC(), NOW_UTC())",
		person.FirstName, person.LastName, person.Email, sequel.Nullify(person.Birthday.UTC()),
	)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return int(id), nil
}

/*
Load returns the person with the given key, and whether it was found.
*/
func (svc *Service) Load(ctx context.Context, key int) (person *yellowpagesapi.Person, found bool, err error) { // MARKER: Load
	row := svc.db.QueryRowContext(ctx, "SELECT "+personColumns+" FROM persons WHERE id=?", key)
	person, err = scanPerson(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	return person, true, nil
}

/*
Store updates an existing person, identified by its key, and reports whether it was found.
*/
func (svc *Service) Store(ctx context.Context, person *yellowpagesapi.Person) (stored bool, err error) { // MARKER: Store
	if person == nil {
		return false, errors.New("person is required", http.StatusBadRequest)
	}
	res, err := svc.db.ExecContext(ctx,
		"UPDATE persons SET first_name=?, last_name=?, email=?, birthday=?, updated_at=NOW_UTC() WHERE id=?",
		person.FirstName, person.LastName, person.Email, sequel.Nullify(person.Birthday.UTC()), person.Key,
	)
	if err != nil {
		return false, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, errors.Trace(err)
	}
	if n > 0 {
		return true, nil
	}
	// Zero rows can mean an update that changed nothing on MySQL; confirm whether the person exists
	var count int
	err = svc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM persons WHERE id=?", person.Key).Scan(&count)
	if err != nil {
		return false, errors.Trace(err)
	}
	return count > 0, nil
}

/*
Delete deletes the person with the given key, and reports whether it was found.
*/
func (svc *Service) Delete(ctx context.Context, key int) (deleted bool, err error) { // MARKER: Delete
	res, err := svc.db.ExecContext(ctx, "DELETE FROM persons WHERE id=?", key)
	if err != nil {
		return false, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, errors.Trace(err)
	}
	return n > 0, nil
}

/*
List returns the persons matching the query, ordered by key.
*/
func (svc *Service) List(ctx context.Context, query yellowpagesapi.Query) (persons []*yellowpagesapi.Person, err error) { // MARKER: List
	var where []string
	var args []any
	if query.FirstName != "" {
		where = append(where, "first_name=?")
		args = append(args, query.FirstName)
	}
	if query.LastName != "" {
		where = append(where, "last_name=?")
		args = append(args, query.LastName)
	}
	if query.Email != "" {
		where = append(where, "email=?")
		args = append(args, query.Email)
	}
	stmt := "SELECT " + personColumns + " FROM persons"
	if len(where) > 0 {
		stmt += " WHERE " + strings.Join(where, " AND ")
	}
	limit := query.Limit
	if limit == 0 {
		limit = defaultLimit
	}
	stmt += " ORDER BY id LIMIT_OFFSET(?, ?)"
	args = append(args, limit, query.Offset)

	rows, err := svc.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		person, err := scanPerson(rows.Scan)
		if err != nil {
			return nil, errors.Trace(err)
		}
		persons = append(persons, person)
	}
	return persons, errors.Trace(rows.Err())
}

/*
CreateREST creates a new person via REST, returning its key.
*/
func (svc *Service) CreateREST(ctx context.Context, httpRequestBody *yellowpagesapi.Person) (key int, httpStatusCode int, err error) { // MARKER: CreateREST
	key, err = svc.Create(ctx, httpRequestBody)
	if err != nil {
		return 0, 0, errors.Trace(err)
	}
	return key, http.StatusCreated, nil
}

/*
LoadREST loads a person by key via REST.
*/
func (svc *Service) LoadREST(ctx context.Context, key int) (httpResponseBody *yellowpagesapi.Person, httpStatusCode int, err error) { // MARKER: LoadREST
	person, found, err := svc.Load(ctx, key)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	if !found {
		return nil, http.StatusNotFound, nil
	}
	return person, http.StatusOK, nil
}

/*
StoreREST updates an existing person via REST.
*/
func (svc *Service) StoreREST(ctx context.Context, key int, httpRequestBody *yellowpagesapi.Person) (httpStatusCode int, err error) { // MARKER: StoreREST
	if httpRequestBody == nil {
		return 0, errors.New("person is required", http.StatusBadRequest)
	}
	httpRequestBody.Key = key
	stored, err := svc.Store(ctx, httpRequestBody)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if !stored {
		return http.StatusNotFound, nil
	}
	return http.StatusNoContent, nil
}

/*
DeleteREST deletes an existing person via REST.
*/
func (svc *Service) DeleteREST(ctx context.Context, key int) (httpStatusCode int, err error) { // MARKER: DeleteREST
	deleted, err := svc.Delete(ctx, key)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if !deleted {
		return http.StatusNotFound, nil
	}
	return http.StatusNoContent, nil
}

/*
ListREST lists persons matching the query via REST.
*/
func (svc *Service) ListREST(ctx context.Context, q yellowpagesapi.Query) (httpResponseBody []*yellowpagesapi.Person, httpStatusCode int, err error) { // MARKER: ListREST
	persons, err := svc.List(ctx, q)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	if persons == nil {
		persons = []*yellowpagesapi.Person{} // An empty JSON array rather than null
	}
	return persons, http.StatusOK, nil
}

/*
Demo serves a web page for trying out the REST API.
*/
func (svc *Service) Demo(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Demo
	ctx := r.Context()
	err = r.ParseForm()
	if err != nil {
		return errors.Trace(err, http.StatusBadRequest)
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
		target, err := url.Parse(r.FormValue("path"))
		if err != nil {
			return errors.Trace(err, http.StatusBadRequest)
		}
		u := (&url.URL{Scheme: "https", Host: Hostname}).JoinPath(target.Path)
		u.RawQuery = target.RawQuery
		var body []byte
		contentType := ""
		if method == "POST" || method == "PUT" {
			body = []byte(r.FormValue("body"))
			contentType = "application/json"
		}
		res, err := svc.Request(
			ctx,
			pub.Method(method),
			pub.URL(u.String()),
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
	w.Header().Set("Content-Type", "text/html")
	err = svc.WriteResTemplate(w, "demo.html", data)
	return errors.Trace(err)
}
