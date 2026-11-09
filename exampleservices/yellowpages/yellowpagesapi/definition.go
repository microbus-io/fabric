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

package yellowpagesapi

import (
	"github.com/microbus-io/fabric/define"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "yellowpages.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "YellowPages"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 12

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `YellowPages persists persons in a SQL database.`

// SQLDataSourceName is the connection string of the SQL database.
var SQLDataSourceName = define.Config{ // MARKER: SQLDataSourceName
	Value:  string(""),
	Secret: true,
}

// Create creates a new person, returning its key.
var Create = define.Function{ // MARKER: Create
	Host: Hostname, Method: "ANY", Route: "/create",
	In: CreateIn{}, Out: CreateOut{},
}

// CreateIn are the input arguments of Create.
type CreateIn struct { // MARKER: Create
	Person *Person `json:"person,omitzero"`
}

// CreateOut are the output arguments of Create.
type CreateOut struct { // MARKER: Create
	Key int `json:"key,omitzero"`
}

// Load returns the person with the given key, and whether it was found.
var Load = define.Function{ // MARKER: Load
	Host: Hostname, Method: "ANY", Route: "/load",
	In: LoadIn{}, Out: LoadOut{},
}

// LoadIn are the input arguments of Load.
type LoadIn struct { // MARKER: Load
	Key int `json:"key,omitzero"`
}

// LoadOut are the output arguments of Load.
type LoadOut struct { // MARKER: Load
	Person *Person `json:"person,omitzero"`
	Found  bool    `json:"found,omitzero"`
}

// Store updates an existing person, identified by its key, and reports whether it was found.
var Store = define.Function{ // MARKER: Store
	Host: Hostname, Method: "ANY", Route: "/store",
	In: StoreIn{}, Out: StoreOut{},
}

// StoreIn are the input arguments of Store.
type StoreIn struct { // MARKER: Store
	Person *Person `json:"person,omitzero"`
}

// StoreOut are the output arguments of Store.
type StoreOut struct { // MARKER: Store
	Stored bool `json:"stored,omitzero"`
}

// Delete deletes the person with the given key, and reports whether it was found.
var Delete = define.Function{ // MARKER: Delete
	Host: Hostname, Method: "ANY", Route: "/delete",
	In: DeleteIn{}, Out: DeleteOut{},
}

// DeleteIn are the input arguments of Delete.
type DeleteIn struct { // MARKER: Delete
	Key int `json:"key,omitzero"`
}

// DeleteOut are the output arguments of Delete.
type DeleteOut struct { // MARKER: Delete
	Deleted bool `json:"deleted,omitzero"`
}

// List returns the persons matching the query, ordered by key.
var List = define.Function{ // MARKER: List
	Host: Hostname, Method: "ANY", Route: "/list",
	In: ListIn{}, Out: ListOut{},
}

// ListIn are the input arguments of List.
type ListIn struct { // MARKER: List
	Query Query `json:"query,omitzero"`
}

// ListOut are the output arguments of List.
type ListOut struct { // MARKER: List
	Persons []*Person `json:"persons,omitzero"`
}

// CreateREST creates a new person via REST, returning its key.
var CreateREST = define.Function{ // MARKER: CreateREST
	Host: Hostname, Method: "POST", Route: "/persons",
	In: CreateRESTIn{}, Out: CreateRESTOut{},
}

// CreateRESTIn are the input arguments of CreateREST.
type CreateRESTIn struct { // MARKER: CreateREST
	HTTPRequestBody *Person `json:"-" jsonschema_description:"The person to create"`
}

// CreateRESTOut are the output arguments of CreateREST.
type CreateRESTOut struct { // MARKER: CreateREST
	Key            int `json:"key,omitzero"`
	HTTPStatusCode int `json:"-"`
}

// LoadREST loads a person by key via REST.
var LoadREST = define.Function{ // MARKER: LoadREST
	Host: Hostname, Method: "GET", Route: "/persons/{key}",
	In: LoadRESTIn{}, Out: LoadRESTOut{},
}

// LoadRESTIn are the input arguments of LoadREST.
type LoadRESTIn struct { // MARKER: LoadREST
	Key int `json:"key,omitzero"`
}

// LoadRESTOut are the output arguments of LoadREST.
type LoadRESTOut struct { // MARKER: LoadREST
	HTTPResponseBody *Person `json:"-" jsonschema_description:"The person"`
	HTTPStatusCode   int     `json:"-"`
}

// StoreREST updates an existing person via REST.
var StoreREST = define.Function{ // MARKER: StoreREST
	Host: Hostname, Method: "PUT", Route: "/persons/{key}",
	In: StoreRESTIn{}, Out: StoreRESTOut{},
}

// StoreRESTIn are the input arguments of StoreREST.
type StoreRESTIn struct { // MARKER: StoreREST
	Key             int     `json:"key,omitzero"`
	HTTPRequestBody *Person `json:"-" jsonschema_description:"The new values of the person"`
}

// StoreRESTOut are the output arguments of StoreREST.
type StoreRESTOut struct { // MARKER: StoreREST
	HTTPStatusCode int `json:"-"`
}

// DeleteREST deletes an existing person via REST.
var DeleteREST = define.Function{ // MARKER: DeleteREST
	Host: Hostname, Method: "DELETE", Route: "/persons/{key}",
	In: DeleteRESTIn{}, Out: DeleteRESTOut{},
}

// DeleteRESTIn are the input arguments of DeleteREST.
type DeleteRESTIn struct { // MARKER: DeleteREST
	Key int `json:"key,omitzero"`
}

// DeleteRESTOut are the output arguments of DeleteREST.
type DeleteRESTOut struct { // MARKER: DeleteREST
	HTTPStatusCode int `json:"-"`
}

// ListREST lists persons matching the query via REST.
var ListREST = define.Function{ // MARKER: ListREST
	Host: Hostname, Method: "GET", Route: "/persons",
	In: ListRESTIn{}, Out: ListRESTOut{},
}

// ListRESTIn are the input arguments of ListREST.
type ListRESTIn struct { // MARKER: ListREST
	Q Query `json:"q,omitzero"`
}

// ListRESTOut are the output arguments of ListREST.
type ListRESTOut struct { // MARKER: ListREST
	HTTPResponseBody []*Person `json:"-" jsonschema_description:"The persons matching the query"`
	HTTPStatusCode   int       `json:"-"`
}

// Demo serves a web page for trying out the REST API.
var Demo = define.Web{ // MARKER: Demo
	Host: Hostname, Method: "ANY", Route: "/demo",
}
