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

package openapi

import (
	"encoding/json"
	"fmt"
	"mime"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/microbus-io/fabric/httpx"
)

// Service is populated with the microservice's specs in order to generate its OpenAPI document.
type Service struct {
	ServiceName string
	Description string
	Version     int
	Endpoints   []*Endpoint
	RemoteURI   string
}

// MarshalJSON produces the JSON representation of the OpenAPI document of the service.
func (s *Service) MarshalJSON() ([]byte, error) {
	doc := oapiDoc{
		OpenAPI: "3.1.0",
		Info: oapiInfo{
			Title:       s.ServiceName,
			Description: s.Description,
			Version:     strconv.Itoa(s.Version),
		},
		Paths: map[string]map[string]*oapiOperation{},
		Components: &oapiComponents{
			Schemas: map[string]*jsonschema.Schema{},
		},
		Servers: []*oapiServer{
			{
				URL: "https://localhost/",
			},
		},
	}
	if s.RemoteURI != "" {
		p := strings.Index(s.RemoteURI, "/"+s.ServiceName+"/")
		if p < 0 {
			p = strings.Index(s.RemoteURI, "/"+s.ServiceName+":")
		}
		if p >= 0 {
			doc.Servers[0].URL = s.RemoteURI[:p+1]
		}
	}

	for _, ep := range s.Endpoints {
		var op *oapiOperation

		path := httpx.JoinHostAndPath(s.ServiceName, ep.Path)
		_, path, _ = strings.Cut(path, "://")
		path = "/" + path

		// Path arguments
		pathArgsOrder := []string{}
		pathArgs := map[string]*oapiParameter{}
		parts := strings.Split(path, "/")
		argIndex := 0
		for i := range parts {
			if strings.HasPrefix(parts[i], "{") && strings.HasSuffix(parts[i], "}") {
				argIndex++
				name := parts[i]
				name = strings.TrimPrefix(name, "{")
				name = strings.TrimSuffix(name, "}")
				if name == "" {
					name = fmt.Sprintf("path%d", argIndex)
					parts[i] = "{" + name + "}"
				} else if name == "+" {
					name = fmt.Sprintf("path%d+", argIndex)
					parts[i] = "{" + name + "}"
				}
				pathArgs[name] = &oapiParameter{
					In:   "path",
					Name: name,
					Schema: &jsonschema.Schema{
						Type: "string",
					},
					Required: true,
				}
				pathArgsOrder = append(pathArgsOrder, name)
			}
		}
		path = strings.Join(parts, "/")

		// Functions
		if ep.Type == "function" {
			if ep.Method == "" || ep.Method == "ANY" {
				ep.Method = "POST"
			}
			op = &oapiOperation{
				Summary:     cleanEndpointSummary(ep.Summary),
				Description: ep.Description,
				Responses: map[string]*oapiResponse{
					"2XX": {
						Description: "OK",
						Content: map[string]*oapiMediaType{
							"application/json": {
								Schema: &jsonschema.Schema{
									Ref: "#/components/schemas/" + ep.Name + "_OUT",
								},
							},
						},
					},
					"4XX": {
						Description: "User error",
						Content: map[string]*oapiMediaType{
							"text/plain": {},
						},
					},
					"5XX": {
						Description: "Server error",
						Content: map[string]*oapiMediaType{
							"text/plain": {},
						},
					},
				},
			}

			// OUT is JSON in the response body
			var schemaOut *jsonschema.Schema
			if field, ok := reflect.TypeOf(ep.OutputArgs).FieldByName("HTTPResponseBody"); ok {
				// httpResponseBody argument overrides the response body and preempts all other return values
				schemaOut = jsonschema.ReflectFromType(field.Type)
			} else {
				schemaOut = jsonschema.Reflect(ep.OutputArgs)
			}
			resolveRefs(doc, schemaOut, ep.Name+"_OUT")
			doc.Components.Schemas[ep.Name+"_OUT"] = schemaOut

			// httpRequestBody argument overrides the request body and forces all other arguments to be in the query or path
			httpRequestBodyExists := false
			if field, ok := reflect.TypeOf(ep.InputArgs).FieldByName("HTTPRequestBody"); ok {
				httpRequestBodyExists = true  // Makes all other args query or path args
				if methodHasBody(ep.Method) { // Only works if the method has a body
					schemaIn := jsonschema.ReflectFromType(field.Type)
					resolveRefs(doc, schemaIn, ep.Name+"_IN")
					doc.Components.Schemas[ep.Name+"_IN"] = schemaIn

					op.RequestBody = &oapiRequestBody{
						Required: true,
						Content: map[string]*oapiMediaType{
							"application/json": {
								Schema: &jsonschema.Schema{
									Ref: "#/components/schemas/" + ep.Name + "_IN",
								},
							},
						},
					}
				}
			}

			if !methodHasBody(ep.Method) || httpRequestBodyExists {
				// IN are explodable query arguments
				inType := reflect.TypeOf(ep.InputArgs)
				for i := 0; i < inType.NumField(); i++ {
					field := inType.Field(i)
					name := fieldName(field)
					if name == "" || name == "httpRequestBody" {
						continue
					}
					parameter := &oapiParameter{
						In:   "query",
						Name: name,
					}
					if pathArgs[name] != nil {
						parameter.In = "path"
						parameter.Required = true
						delete(pathArgs, name)
					} else if pathArgs[name+"+"] != nil {
						parameter.Name += "+"
						parameter.In = "path"
						parameter.Required = true
						delete(pathArgs, name+"+")
					}

					fieldSchema := jsonschema.ReflectFromType(field.Type)
					resolveRefs(doc, fieldSchema, ep.Name+"_IN")
					if fieldSchema.Ref != "" {
						// Non-primitive type
						parameter.Schema = &jsonschema.Schema{
							Ref: strings.Replace(fieldSchema.Ref, "#/$defs/", "#/components/schemas/"+ep.Name+"_IN_", 1),
						}
					} else {
						parameter.Schema = fieldSchema
					}
					parameter.Style = "deepObject"
					parameter.Explode = true
					op.Parameters = append(op.Parameters, parameter)
				}
			} else {
				// IN is JSON in the request body
				schemaIn := jsonschema.Reflect(ep.InputArgs)
				resolveRefs(doc, schemaIn, ep.Name+"_IN")
				doc.Components.Schemas[ep.Name+"_IN"] = schemaIn

				op.RequestBody = &oapiRequestBody{
					Required: true,
					Content: map[string]*oapiMediaType{
						"application/json": {
							Schema: &jsonschema.Schema{
								Ref: "#/components/schemas/" + ep.Name + "_IN",
							},
						},
					},
				}
			}
		}

		if ep.Type == "web" {
			if ep.Method == "" || ep.Method == "ANY" {
				ep.Method = "GET"
			}
			op = &oapiOperation{
				Summary:     cleanEndpointSummary(ep.Summary),
				Description: ep.Description,
				Parameters:  []*oapiParameter{},
				Responses: map[string]*oapiResponse{
					"200": {
						Description: "OK",
					},
				},
			}
			p := strings.LastIndex(ep.Path, ".")
			if p >= 0 {
				contentType := mime.TypeByExtension(ep.Path[p:])
				if contentType != "" {
					op.Responses = map[string]*oapiResponse{
						"200": {
							Content: map[string]*oapiMediaType{
								contentType: {},
							},
						},
					}
				}
			}
		}

		// Path arguments
		for i := range pathArgsOrder {
			arg := pathArgs[pathArgsOrder[i]]
			if arg != nil {
				op.Parameters = append(op.Parameters, arg)
			}
		}

		// Add to paths
		if doc.Paths[path] == nil {
			doc.Paths[path] = map[string]*oapiOperation{}
		}
		doc.Paths[path][strings.ToLower(ep.Method)] = op
	}
	return json.Marshal(doc)
}

func cleanEndpointSummary(sig string) string {
	// Remove request/response
	sig = strings.Replace(sig, "(w http.ResponseWriter, r *http.Request)", "()", -1)
	sig = strings.Replace(sig, "(w http.ResponseWriter, r *http.Request, ", "(", -1)
	// Remove ctx argument
	sig = strings.Replace(sig, "(ctx context.Context)", "()", -1)
	sig = strings.Replace(sig, "(ctx context.Context, ", "(", -1)
	// Remove error return value
	sig = strings.Replace(sig, " (err error)", "", -1)
	sig = strings.Replace(sig, ", err error)", ")", -1)
	// Remove pointers
	sig = strings.Replace(sig, "*", "", -1)
	// Remove package identifiers
	sig = regexp.MustCompile(`\w+\.`).ReplaceAllString(sig, "")
	return sig
}

// methodHasBody indicates if the HTTP method has a body.
func methodHasBody(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "DELETE", "TRACE", "OPTIONS", "HEAD":
		return false
	default:
		return true
	}
}

func fieldName(field reflect.StructField) string {
	if field.Name[:1] != strings.ToUpper(field.Name[:1]) {
		// Not a public field
		return ""
	}
	name := field.Tag.Get("json")
	if comma := strings.Index(name, ","); comma >= 0 {
		name = name[:comma]
	}
	if name == "" {
		// No JSON tag, use field name
		name = field.Name
	}
	if name == "-" {
		// Omitted
		name = ""
	}
	return name
}

// resolveRefs recursively resolves all type references in the schema and moves them to the component section of the OpenAPI document.
func resolveRefs(doc oapiDoc, schema *jsonschema.Schema, endpoint string) {
	// Move $defs into the components section of the document
	// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
	for defKey, defSchema := range schema.Definitions {
		doc.Components.Schemas[endpoint+"_"+defKey] = defSchema
		// Recurse on nested schemas
		resolveRefs(doc, defSchema, endpoint)
	}
	// Resolve all $def references to the components section of the document
	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		if strings.HasPrefix(pair.Value.Ref, "#/$defs/") {
			// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
			pair.Value.Ref = "#/components/schemas/" + endpoint + "_" + pair.Value.Ref[8:]
		}
	}
	schema.Definitions = nil
	schema.Version = "" // Avoid rendering the $schema property
}
