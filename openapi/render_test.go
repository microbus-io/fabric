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

package openapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

// resolveSchema follows a $ref in an OpenAPI schema map to find the actual schema with properties.
func resolveSchema(schemas map[string]any, name string) map[string]any {
	schema := schemas[name].(map[string]any)
	if ref, ok := schema["$ref"].(string); ok {
		// Extract the schema name from "#/components/schemas/Name"
		parts := strings.Split(ref, "/")
		refName := parts[len(parts)-1]
		return schemas[refName].(map[string]any)
	}
	return schema
}

// TestRender_ParamDescriptions covers the GET case where descriptions on scalar query parameters come
// from the In struct fields' jsonschema tags. A query parameter is reflected from its field type alone,
// so the tag is read directly by fieldTagDescription rather than via struct reflection.
func TestRender_ParamDescriptions(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type ForecastIn struct {
		City string `json:"city,omitzero" jsonschema_description:"The city name"`
		Days int    `json:"days,omitzero" jsonschema_description:"Number of days to forecast"`
	}
	type ForecastOut struct {
		Forecast   string  `json:"forecast,omitzero" jsonschema_description:"Daily forecast summaries"`
		Confidence float64 `json:"confidence,omitzero" jsonschema_description:"Model confidence score"`
	}

	svc := &Service{
		ServiceName: "weather.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "Forecast",
				Method:     "GET",
				Route:      "/forecast",
				Summary:    "Forecast(city string, days int) (forecast string, confidence float64)",
				InputArgs:  ForecastIn{},
				OutputArgs: ForecastOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}

	// Parse the JSON to verify descriptions are present
	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	// GET method: parameters are query args
	paths := doc["paths"].(map[string]any)
	path := paths["/weather.test/forecast"].(map[string]any)
	op := path["get"].(map[string]any)
	params := op["parameters"].([]any)

	// Find city and days parameters
	var cityDesc, daysDesc string
	for _, p := range params {
		param := p.(map[string]any)
		switch param["name"] {
		case "city":
			cityDesc, _ = param["description"].(string)
		case "days":
			daysDesc, _ = param["description"].(string)
		}
	}
	assert.Expect(cityDesc, "The city name")
	assert.Expect(daysDesc, "Number of days to forecast")

	// Verify x-feature-type is present
	assert.Expect(op["x-feature-type"], "function")

	// Check output schema descriptions (follow $ref if needed)
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	outSchema := resolveSchema(schemas, "weather_test__Forecast_OUT")
	outProps := outSchema["properties"].(map[string]any)
	forecastProp := outProps["forecast"].(map[string]any)
	confidenceProp := outProps["confidence"].(map[string]any)
	assert.Expect(forecastProp["description"], "Daily forecast summaries")
	assert.Expect(confidenceProp["description"], "Model confidence score")
}

func TestRender_ParamDescriptions_POST(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type CreateIn struct {
		Name  string `json:"name,omitzero" jsonschema_description:"The user's display name"`
		Email string `json:"email,omitzero" jsonschema_description:"The user's email address"`
	}
	type CreateOut struct {
		ID string `json:"id,omitzero" jsonschema_description:"The generated user ID"`
	}

	svc := &Service{
		ServiceName: "user.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "Create",
				Method:     "POST",
				Route:      "/create",
				Summary:    "Create(name string, email string) (id string)",
				InputArgs:  CreateIn{},
				OutputArgs: CreateOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}

	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	// POST method: parameters are in request body schema
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	inSchema := resolveSchema(schemas, "user_test__Create_IN")
	inProps := inSchema["properties"].(map[string]any)
	nameProp := inProps["name"].(map[string]any)
	emailProp := inProps["email"].(map[string]any)
	assert.Expect(nameProp["description"], "The user's display name")
	assert.Expect(emailProp["description"], "The user's email address")

	outSchema := resolveSchema(schemas, "user_test__Create_OUT")
	outProps := outSchema["properties"].(map[string]any)
	idProp := outProps["id"].(map[string]any)
	assert.Expect(idProp["description"], "The generated user ID")
}

// TestRender_MagicBodyDescriptions covers the magic HTTP body case. A jsonschema_description tag on the
// HTTPRequestBody / HTTPResponseBody field is dropped by struct reflection (the body schema is reflected
// from the field's type alone), so it is read directly and set on the requestBody / response nodes.
func TestRender_MagicBodyDescriptions(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type Payload struct {
		Value string `json:"value,omitzero"`
	}
	type CreateIn struct {
		HTTPRequestBody *Payload `json:"-" jsonschema_description:"The object to create"`
	}
	type CreateOut struct {
		HTTPResponseBody *Payload `json:"-" jsonschema_description:"The created object"`
		HTTPStatusCode   int      `json:"-"`
	}

	svc := &Service{
		ServiceName: "store.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "Create",
				Method:     "POST",
				Route:      "/create",
				Summary:    "Create(httpRequestBody *Payload) (httpResponseBody *Payload, httpStatusCode int)",
				InputArgs:  CreateIn{},
				OutputArgs: CreateOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}

	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	paths := doc["paths"].(map[string]any)
	op := paths["/store.test/create"].(map[string]any)["post"].(map[string]any)

	// Request body description comes from the HTTPRequestBody field tag
	reqBody := op["requestBody"].(map[string]any)
	assert.Expect(reqBody["description"], "The object to create")

	// Response description comes from the HTTPResponseBody field tag, replacing the "OK" default
	responses := op["responses"].(map[string]any)
	resp := responses["2XX"].(map[string]any)
	assert.Expect(resp["description"], "The created object")
}

func TestRender_JsonSchemaTags(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type Location struct {
		Lat  float64 `json:"lat" jsonschema_description:"Latitude in decimal degrees"`
		Long float64 `json:"long" jsonschema_description:"Longitude in decimal degrees"`
	}
	type SearchIn struct {
		Location Location `json:"location,omitzero"`
	}
	type SearchOut struct {
		Found bool `json:"found,omitzero"`
	}

	svc := &Service{
		ServiceName: "geo.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "Search",
				Method:     "POST",
				Route:      "/search",
				Summary:    "Search(location Location) (found bool)",
				InputArgs:  SearchIn{},
				OutputArgs: SearchOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}

	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	// The Location type should be in components/schemas with field descriptions from jsonschema tags
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	locSchema := schemas["geo_test__Search_IN_Location"].(map[string]any)
	locProps := locSchema["properties"].(map[string]any)
	latProp := locProps["lat"].(map[string]any)
	longProp := locProps["long"].(map[string]any)
	assert.Expect(latProp["description"], "Latitude in decimal degrees")
	assert.Expect(longProp["description"], "Longitude in decimal degrees")
}

func TestRender_GreedyPath(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type LoadObjectIn struct {
		Category int    `json:"category,omitzero"`
		Name     string `json:"name,omitzero"`
	}
	type LoadObjectOut struct {
		Found bool `json:"found,omitzero"`
	}

	svc := &Service{
		ServiceName: "objects.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "LoadObject",
				Method:     "GET",
				Route:      "/load/{category}/{name...}",
				InputArgs:  LoadObjectIn{},
				OutputArgs: LoadObjectOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}

	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	// The greedy "..." suffix must not leak into the rendered path or parameter name.
	paths := doc["paths"].(map[string]any)
	_, hasClean := paths["/objects.test/load/{category}/{name}"]
	assert.True(hasClean, "rendered path should be /objects.test/load/{category}/{name}, got: %v", paths)

	// Both category and name must be path parameters; nothing should be a query parameter.
	op := paths["/objects.test/load/{category}/{name}"].(map[string]any)["get"].(map[string]any)
	params := op["parameters"].([]any)
	gotIn := map[string]string{}
	for _, p := range params {
		param := p.(map[string]any)
		name, _ := param["name"].(string)
		in, _ := param["in"].(string)
		gotIn[name] = in
	}
	assert.Expect(gotIn["category"], "path")
	assert.Expect(gotIn["name"], "path")
	_, nameDotted := gotIn["name..."]
	assert.False(nameDotted, "parameter name must not include the trailing dots")
}

// TestRender_DV8BodyConstraints covers the projection of dv8 field directives onto the schema of a
// request body: length and value bounds, patterns, enums, defaults, required fields, and the each/key
// prefixes that descend into the elements and keys of containers.
func TestRender_DV8BodyConstraints(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type Person struct {
		Name    string            `json:"name,omitzero" dv8:"trim,notzero,len<=64"`
		Email   string            `json:"email,omitzero" dv8:"regexp ^.+@.+$"`
		Age     int               `json:"age,omitzero" dv8:"val>=0,val<120"`
		Size    string            `json:"size,omitzero" dv8:"oneof S|M|L,default=M"`
		Tags    []string          `json:"tags,omitzero" dv8:"len>0,each len<=16"`
		Attribs map[string]string `json:"attribs,omitzero" dv8:"len<8,key len>=2,each notzero"`
	}
	type CreateIn struct {
		Person Person `json:"person,omitzero" dv8:"notzero"`
	}
	type CreateOut struct {
		Created bool `json:"created,omitzero"`
	}

	svc := &Service{
		ServiceName: "directory.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "Create",
				Method:     "POST",
				Route:      "/create",
				Summary:    "Create(person Person) (created bool)",
				InputArgs:  CreateIn{},
				OutputArgs: CreateOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}
	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	// notzero on the top-level field marks it required on the IN schema
	inSchema := resolveSchema(schemas, "directory_test__Create_IN")
	assert.Expect(inSchema["required"], []any{"person"})

	personSchema := schemas["directory_test__Create_IN_Person"].(map[string]any)
	assert.Expect(personSchema["required"], []any{"name"})
	props := personSchema["properties"].(map[string]any)

	nameProp := props["name"].(map[string]any)
	assert.Expect(nameProp["minLength"], float64(1)) // notzero
	assert.Expect(nameProp["maxLength"], float64(64))

	emailProp := props["email"].(map[string]any)
	assert.Expect(emailProp["pattern"], "^.+@.+$")

	ageProp := props["age"].(map[string]any)
	assert.Expect(ageProp["minimum"], float64(0))
	assert.Expect(ageProp["exclusiveMaximum"], float64(120))

	sizeProp := props["size"].(map[string]any)
	assert.Expect(sizeProp["enum"], []any{"S", "M", "L"})
	assert.Expect(sizeProp["default"], "M")

	tagsProp := props["tags"].(map[string]any)
	assert.Expect(tagsProp["minItems"], float64(1)) // len>0
	tagsItems := tagsProp["items"].(map[string]any)
	assert.Expect(tagsItems["maxLength"], float64(16)) // each len<=16

	attribsProp := props["attribs"].(map[string]any)
	assert.Expect(attribsProp["maxProperties"], float64(7)) // len<8
	attribsKeys := attribsProp["propertyNames"].(map[string]any)
	assert.Expect(attribsKeys["minLength"], float64(2)) // key len>=2
	attribsVals := attribsProp["additionalProperties"].(map[string]any)
	assert.Expect(attribsVals["minLength"], float64(1)) // each notzero
}

// TestRender_DV8ParamConstraints covers the projection of dv8 directives declared on the fields backing
// scalar query parameters and magic HTTP body arguments, whose schemas are reflected from the field type
// alone (dropping the field-level tag). Query parameters remain not required regardless of notzero.
func TestRender_DV8ParamConstraints(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	type ListIn struct {
		Filter string `json:"filter,omitzero" dv8:"notzero,len<=32"`
		Limit  int    `json:"limit,omitzero" dv8:"val>0,val<=100"`
	}
	type ListOut struct {
		HTTPResponseBody []string `json:"-"`
	}
	type ImportIn struct {
		HTTPRequestBody []string `json:"-" dv8:"notzero,each len>0"`
	}
	type ImportOut struct {
		Imported int `json:"imported,omitzero"`
	}

	svc := &Service{
		ServiceName: "catalog.test",
		Endpoints: []*Endpoint{
			{
				Type:       "function",
				Name:       "List",
				Method:     "GET",
				Route:      "/list",
				Summary:    "List(filter string, limit int) (items []string)",
				InputArgs:  ListIn{},
				OutputArgs: ListOut{},
			},
			{
				Type:       "function",
				Name:       "Import",
				Method:     "POST",
				Route:      "/import",
				Summary:    "Import(items []string) (imported int)",
				InputArgs:  ImportIn{},
				OutputArgs: ImportOut{},
			},
		},
	}

	data, err := json.Marshal(Render(svc))
	if !assert.NoError(err) {
		return
	}
	var doc map[string]any
	err = json.Unmarshal(data, &doc)
	if !assert.NoError(err) {
		return
	}

	paths := doc["paths"].(map[string]any)

	listOp := paths["/catalog.test/list"].(map[string]any)["get"].(map[string]any)
	params := listOp["parameters"].([]any)
	byName := map[string]map[string]any{}
	for _, p := range params {
		param := p.(map[string]any)
		byName[param["name"].(string)] = param
	}
	filterSchema := byName["filter"]["schema"].(map[string]any)
	assert.Expect(filterSchema["minLength"], float64(1)) // notzero
	assert.Expect(filterSchema["maxLength"], float64(32))
	assert.Expect(byName["filter"]["required"], nil) // query params stay optional
	limitSchema := byName["limit"]["schema"].(map[string]any)
	assert.Expect(limitSchema["exclusiveMinimum"], float64(0))
	assert.Expect(limitSchema["maximum"], float64(100))

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	importIn := schemas["catalog_test__Import_IN"].(map[string]any)
	assert.Expect(importIn["minItems"], float64(1)) // notzero
	importItems := importIn["items"].(map[string]any)
	assert.Expect(importItems["minLength"], float64(1)) // each len>0
}
