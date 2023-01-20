/*
Copyright 2023 Microbus LLC and various contributors

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

// Code generated by Microbus. DO NOT EDIT.

/*
Package intermediate serves as the foundation of the directory.example microservice.

The directory microservice stores personal records in a SQL database.
*/
package intermediate

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/shardedsql"
	"github.com/microbus-io/fabric/sub"

	"github.com/microbus-io/fabric/examples/directory/resources"
	"github.com/microbus-io/fabric/examples/directory/directoryapi"
)

var (
	_ context.Context
	_ *embed.FS
	_ *json.Decoder
	_ fmt.Stringer
	_ *http.Request
	_ filepath.WalkFunc
	_ strconv.NumError
	_ strings.Reader
	_ time.Duration
	_ cb.Option
	_ cfg.Option
	_ *errors.TracedError
	_ *httpx.ResponseRecorder
	_ *log.Field
	_ *shardedsql.DB
	_ sub.Option
	_ directoryapi.Client
)

// ToDo defines the interface that the microservice must implement.
// The intermediate delegates handling to this interface.
type ToDo interface {
	OnStartup(ctx context.Context) (err error)
	OnShutdown(ctx context.Context) (err error)
	Create(ctx context.Context, person *directoryapi.Person) (created *directoryapi.Person, err error)
	Load(ctx context.Context, key directoryapi.PersonKey) (person *directoryapi.Person, ok bool, err error)
	Delete(ctx context.Context, key directoryapi.PersonKey) (ok bool, err error)
	Update(ctx context.Context, person *directoryapi.Person) (updated *directoryapi.Person, ok bool, err error)
	LoadByEmail(ctx context.Context, email string) (person *directoryapi.Person, ok bool, err error)
	List(ctx context.Context) (keys []directoryapi.PersonKey, err error)
}

// Intermediate extends and customizes the generic base connector.
// Code generated microservices then extend the intermediate.
type Intermediate struct {
	*connector.Connector
	impl ToDo
	dbMaria *shardedsql.DB
}

// NewService creates a new intermediate service.
func NewService(impl ToDo, version int) *Intermediate {
	svc := &Intermediate{
		Connector: connector.New("directory.example"),
		impl: impl,
	}
	svc.SetVersion(version)
	svc.SetDescription(`The directory microservice stores personal records in a SQL database.`)

	// SQL databases
	svc.SetOnStartup(svc.dbMariaOnStartup)
	svc.SetOnShutdown(svc.dbMariaOnShutdown)
	svc.DefineConfig(
		"Maria",
		cfg.Description("Maria is the connection string to the sharded SQL database."),
		cfg.Secret(),
	)
	svc.SetOnConfigChanged(svc.dbMariaOnConfigChanged)

	// Lifecycle
	svc.SetOnStartup(svc.impl.OnStartup)
	svc.SetOnShutdown(svc.impl.OnShutdown)	

	// Functions
	svc.Subscribe(`:443/create`, svc.doCreate)
	svc.Subscribe(`:443/load`, svc.doLoad)
	svc.Subscribe(`:443/delete`, svc.doDelete)
	svc.Subscribe(`:443/update`, svc.doUpdate)
	svc.Subscribe(`:443/load-by-email`, svc.doLoadByEmail)
	svc.Subscribe(`:443/list`, svc.doList)

	return svc
}

// Resources is the in-memory file system of the embedded resources.
func (svc *Intermediate) Resources() embed.FS {
	return resources.FS
}

// doOnConfigChanged is called when the config of the microservice changes.
func (svc *Intermediate) doOnConfigChanged(ctx context.Context, changed func(string) bool) (err error) {
	return nil
}

// dbMariaOnStartup opens the connection to the Maria database and runs schema migrations.
func (svc *Intermediate) dbMariaOnStartup(ctx context.Context) (err error) {
	if svc.dbMaria != nil {
		svc.dbMariaOnShutdown(ctx)
	}
	dataSource := svc.Maria()
	if dataSource != "" {
		svc.dbMaria, err = shardedsql.Open(ctx, "mariadb", dataSource)
		if err != nil {
			return errors.Trace(err)
		}
		svc.LogInfo(ctx, "Opened database", log.String("db", "Maria"))

		migrations := shardedsql.NewStatementSequence(svc.HostName() + " Maria")
		scripts, _ := svc.Resources().ReadDir("maria")
		for _, script := range scripts {
			if script.IsDir() || filepath.Ext(script.Name())!=".sql" {
				continue
			}
			number, err := strconv.Atoi(strings.TrimSuffix(script.Name(), ".sql"))
			if err != nil {
				continue
			}
			statement, _ := svc.Resources().ReadFile(filepath.Join("maria", script.Name()))
			migrations.Insert(number, string(statement))
		}
		err = svc.dbMaria.MigrateSchema(ctx, migrations)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// dbMariaOnStartup closes the connection to the Maria database.
func (svc *Intermediate) dbMariaOnShutdown(ctx context.Context) (err error) {
	if svc.dbMaria != nil {
		svc.dbMaria.Close()
		svc.dbMaria = nil
		svc.LogInfo(ctx, "Closed database", log.String("db", "Maria"))
	}
	return nil
}

// dbMariaOnConfigChanged reconnects to the Maria database when the data source name changes.
func (svc *Intermediate) dbMariaOnConfigChanged(ctx context.Context, changed func(string) bool) (err error) {
	if changed("Maria") {
		err = svc.dbMariaOnStartup(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Maria is the data source name to the sharded SQL database.
func (svc *Intermediate) Maria() (dsn string) {
	return svc.Config("Maria")
}

// Maria initializes the Maria config property of the microservice.
func Maria(dsn string) (func(connector.Service) error) {
	return func(svc connector.Service) error {
		return svc.SetConfig("Maria", dsn)
	}
}

// MariaDatabase is the sharded SQL database.
func (svc *Intermediate) MariaDatabase() *shardedsql.DB {
	return svc.dbMaria
}

// doCreate handles marshaling for the Create function.
func (svc *Intermediate) doCreate(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.CreateIn
	var o directoryapi.CreateOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Created, err = svc.impl.Create(
		r.Context(),
		i.Person,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doLoad handles marshaling for the Load function.
func (svc *Intermediate) doLoad(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.LoadIn
	var o directoryapi.LoadOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Person, o.Ok, err = svc.impl.Load(
		r.Context(),
		i.Key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doDelete handles marshaling for the Delete function.
func (svc *Intermediate) doDelete(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.DeleteIn
	var o directoryapi.DeleteOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Ok, err = svc.impl.Delete(
		r.Context(),
		i.Key,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doUpdate handles marshaling for the Update function.
func (svc *Intermediate) doUpdate(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.UpdateIn
	var o directoryapi.UpdateOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Updated, o.Ok, err = svc.impl.Update(
		r.Context(),
		i.Person,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doLoadByEmail handles marshaling for the LoadByEmail function.
func (svc *Intermediate) doLoadByEmail(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.LoadByEmailIn
	var o directoryapi.LoadByEmailOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Person, o.Ok, err = svc.impl.LoadByEmail(
		r.Context(),
		i.Email,
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// doList handles marshaling for the List function.
func (svc *Intermediate) doList(w http.ResponseWriter, r *http.Request) error {
	var i directoryapi.ListIn
	var o directoryapi.ListOut
	err := httpx.ParseRequestData(r, &i)
	if err!=nil {
		return errors.Trace(err)
	}
	o.Keys, err = svc.impl.List(
		r.Context(),
	)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
