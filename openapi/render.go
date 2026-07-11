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
	"fmt"
	"mime"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/httpx"
)

// Render builds an OpenAPI 3.1 [Document] from a service registration. All schema names
// (component keys and $ref pointers) are prefixed with the service hostname, with dots
// converted to underscores. The prefix scopes schemas to their owning service so multiple
// services' docs can be aggregated without component-key collisions.
func Render(s *Service) *Document {
	doc := &Document{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       s.ServiceName,
			Description: s.Description,
			Version:     strconv.Itoa(s.Version),
		},
		Paths: map[string]map[string]*Operation{},
		Components: &Components{
			Schemas:         map[string]*jsonschema.Schema{},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
		Servers: []*Server{
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

	scopePrefix := strings.ReplaceAll(s.ServiceName, ".", "_")

	// Error schema. Not scoped to the service: every service produces an identical
	// StreamedError schema, so we use a stable framework-level key. When multiple per-service
	// docs are aggregated, the identical schemas merge into one entry rather than N copies.
	type Error errors.StreamedError
	type ErrorResponse struct {
		Err Error `json:"err"`
	}
	errorType := reflect.TypeOf(ErrorResponse{})
	errorSchema := jsonschemaReflectFromType(errorType)
	resolveRefs(doc, errorSchema, "error")
	errorRef := "#/components/schemas/error_" + errorType.Name()

	for _, ep := range s.Endpoints {
		var op *Operation
		// Use locals so Render stays pure - the caller's *Endpoint is not mutated.
		// reflect operations panic on a nil interface, so substitute an empty struct
		// (zero fields) when the endpoint declared no In/Out shape.
		inputArgs := ep.InputArgs
		if inputArgs == nil {
			inputArgs = struct{}{}
		}
		outputArgs := ep.OutputArgs
		if outputArgs == nil {
			outputArgs = struct{}{}
		}

		host := ep.Hostname
		if host == "" {
			host = s.ServiceName
		}
		path := httpx.JoinHostAndPath(host, ep.Route)
		_, path, _ = strings.Cut(path, "://")
		path = "/" + path

		// epKey scopes the endpoint's schema names within this service. The service-level
		// scopePrefix prevents collisions across services in an aggregated doc. The double
		// underscore separates the (single-underscored) hostname from the endpoint name so the
		// boundary is visually obvious in component keys.
		epKey := scopePrefix + "__" + ep.Name

		// Path arguments
		pathArgsOrder := []string{}
		pathArgs := map[string]*Parameter{}
		parts := strings.Split(path, "/")
		argIndex := 0
		for i := range parts {
			if strings.HasPrefix(parts[i], "{") && strings.HasSuffix(parts[i], "}") {
				argIndex++
				name := parts[i]
				name = strings.TrimPrefix(name, "{")
				name = strings.TrimSuffix(name, "}")
				// {name...} is the framework's greedy form (captures the rest of the path),
				// but OpenAPI's path templating syntax is just {name} - there is no spec-level
				// notation for greedy matching. Emitting {name...} would produce an invalid
				// OpenAPI document, so the dots are stripped from both the rendered path and
				// the parameter name. Substitution still works because consumers fill {name}
				// with a value that may contain slashes.
				name = strings.TrimSuffix(name, "...")
				if name == "" {
					name = fmt.Sprintf("path%d", argIndex)
				}
				parts[i] = "{" + name + "}"
				pathArgs[name] = &Parameter{
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
		if ep.Type == FeatureFunction || ep.Type == FeatureWorkflow || ep.Type == FeatureTask {
			method := ep.Method
			if method == "" || method == "ANY" {
				method = "POST"
			}
			op = &Operation{
				Summary:      cleanEndpointSummary(ep.Summary),
				Description:  ep.Description,
				XFeatureType: ep.Type,
				XName:        ep.Name,
				Responses: map[string]*Response{
					"2XX": {
						Description: "OK",
						Content: map[string]*MediaType{
							"application/json": {
								Schema: &jsonschema.Schema{
									Ref: "#/components/schemas/" + epKey + "_OUT",
								},
							},
						},
					},
				},
			}

			// OUT is JSON in the response body
			var schemaOut *jsonschema.Schema
			if field, ok := reflect.TypeOf(outputArgs).FieldByName("HTTPResponseBody"); ok {
				// httpResponseBody argument overrides the response body and preempts all other return values
				schemaOut = jsonschemaReflectFromType(field.Type)
				// The body schema is reflected from the field's type alone, which drops the field-level
				// tag, so read the description directly and set it on the response node.
				if desc := fieldTagDescription(field); desc != "" {
					op.Responses["2XX"].Description = desc
				}
			} else {
				schemaOut = jsonschemaReflect(outputArgs)
			}
			resolveRefs(doc, schemaOut, epKey+"_OUT")
			doc.Components.Schemas[epKey+"_OUT"] = schemaOut

			// httpRequestBody argument overrides the request body and forces all other arguments to be in the query or path
			httpRequestBodyExists := false
			if field, ok := reflect.TypeOf(inputArgs).FieldByName("HTTPRequestBody"); ok {
				httpRequestBodyExists = true // Makes all other args query or path args
				if methodHasBody(method) {   // Only works if the method has a body
					schemaIn := jsonschemaReflectFromType(field.Type)
					// The body schema is reflected from the field's type alone, which drops the field-level
					// tag, so dv8 directives on the field itself are applied directly.
					applyDV8FieldTag(schemaIn, field)
					resolveRefs(doc, schemaIn, epKey+"_IN")
					doc.Components.Schemas[epKey+"_IN"] = schemaIn

					op.RequestBody = &RequestBody{
						Required: true,
						Content: map[string]*MediaType{
							"application/json": {
								Schema: &jsonschema.Schema{
									Ref: "#/components/schemas/" + epKey + "_IN",
								},
							},
						},
					}
					// The body schema is reflected from the field's type alone, which drops the field-level
					// tag, so read the description directly and set it on the request body node.
					if desc := fieldTagDescription(field); desc != "" {
						op.RequestBody.Description = desc
					}
				}
			}

			if !methodHasBody(method) || httpRequestBodyExists {
				// IN are explodable query arguments
				inType := reflect.TypeOf(inputArgs)
				for i := range inType.NumField() {
					field := inType.Field(i)
					name := fieldName(field)
					if name == "" || name == "httpRequestBody" {
						continue
					}
					parameter := &Parameter{
						In:   "query",
						Name: name,
					}
					if pathArgs[name] != nil {
						parameter.In = "path"
						parameter.Required = true
						delete(pathArgs, name)
					}

					fieldSchema := jsonschemaReflectFromType(field.Type)
					resolveRefs(doc, fieldSchema, epKey+"_IN")
					if fieldSchema.Ref != "" {
						// Non-primitive type
						parameter.Schema = &jsonschema.Schema{
							Ref: strings.Replace(fieldSchema.Ref, "#/$defs/", "#/components/schemas/"+epKey+"_IN_", 1),
						}
					} else {
						parameter.Schema = fieldSchema
					}
					// `deepObject`/`explode` are query-only per OpenAPI 3.1. Setting them on
					// path parameters fails validation; path params use the default `simple`
					// style.
					if parameter.In == "query" {
						parameter.Style = "deepObject"
						parameter.Explode = true
					}
					if desc := fieldTagDescription(field); desc != "" {
						parameter.Description = desc
					}
					// The parameter schema is reflected from the field's type alone, which drops the
					// field-level tag, so dv8 directives on the field itself are applied directly.
					// The parameter is deliberately not marked required (see CLAUDE.md).
					applyDV8FieldTag(parameter.Schema, field)
					op.Parameters = append(op.Parameters, parameter)
				}
			} else {
				// IN is JSON in the request body
				schemaIn := jsonschemaReflect(inputArgs)
				resolveRefs(doc, schemaIn, epKey+"_IN")
				doc.Components.Schemas[epKey+"_IN"] = schemaIn

				op.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]*MediaType{
						"application/json": {
							Schema: &jsonschema.Schema{
								Ref: "#/components/schemas/" + epKey + "_IN",
							},
						},
					},
				}
			}
		}

		if ep.Type == FeatureWeb {
			method := ep.Method
			if method == "" || method == "ANY" {
				method = "GET"
			}
			op = &Operation{
				Summary:      cleanEndpointSummary(ep.Summary),
				Description:  ep.Description,
				XFeatureType: ep.Type,
				XName:        ep.Name,
				Parameters:   []*Parameter{},
				Responses: map[string]*Response{
					"2XX": {
						Description: "OK",
					},
				},
			}
			p := strings.LastIndex(ep.Route, ".")
			if p >= 0 {
				contentType := mime.TypeByExtension(ep.Route[p:])
				if contentType != "" {
					op.Responses["2XX"].Content = map[string]*MediaType{
						contentType: {},
					}
				}
			}
		}

		if op == nil {
			continue
		}

		// Authorization
		if ep.RequiredClaims != "" {
			const securitySchemaName = "http_bearer_jwt"
			if doc.Components.SecuritySchemes[securitySchemaName] == nil {
				doc.Components.SecuritySchemes[securitySchemaName] = &SecurityScheme{
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				}
			}
			op.Security = append(op.Security, &SecurityRequirement{securitySchemaName: {}})
			op.Responses["401"] = &Response{
				Description: "Unauthorized",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &jsonschema.Schema{
							Ref: errorRef,
						},
					},
				},
			}
			op.Responses["403"] = &Response{
				Description: "Forbidden; token required with: " + ep.RequiredClaims,
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &jsonschema.Schema{
							Ref: errorRef,
						},
					},
				},
			}
		}

		op.Responses["4XX"] = &Response{
			Description: "User error",
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &jsonschema.Schema{
						Ref: errorRef,
					},
				},
			},
		}
		op.Responses["5XX"] = &Response{
			Description: "Server error",
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &jsonschema.Schema{
						Ref: errorRef,
					},
				},
			},
		}

		// Path arguments
		for i := range pathArgsOrder {
			arg := pathArgs[pathArgsOrder[i]]
			if arg != nil {
				op.Parameters = append(op.Parameters, arg)
			}
		}

		// Determine the HTTP method for the path key
		pathMethod := ep.Method
		if ep.Type == FeatureFunction || ep.Type == FeatureWorkflow || ep.Type == FeatureTask {
			if pathMethod == "" || pathMethod == "ANY" {
				pathMethod = "POST"
			}
		} else if ep.Type == FeatureWeb {
			if pathMethod == "" || pathMethod == "ANY" {
				pathMethod = "GET"
			}
		}

		// Add to paths
		if doc.Paths[path] == nil {
			doc.Paths[path] = map[string]*Operation{}
		}
		doc.Paths[path][strings.ToLower(pathMethod)] = op
	}
	return doc
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

// fieldTagDescription returns the description declared on a struct field, mirroring how
// invopop/jsonschema sources it when reflecting a whole struct: the dedicated `jsonschema_description`
// tag (read whole) takes precedence, then a `description=` directive in the comma-separated `jsonschema`
// tag. A query/path parameter's schema is reflected from the field's type alone, which drops the
// field-level tag, so the description is read directly here - keeping query/path params describable by
// the same tags as body fields. As with invopop, a `description=` directive in the `jsonschema` tag may
// not contain a comma (it is split on commas); use `jsonschema_description` for a description that does.
func fieldTagDescription(field reflect.StructField) string {
	if desc := field.Tag.Get("jsonschema_description"); desc != "" {
		return desc
	}
	tag := field.Tag.Get("jsonschema")
	for _, part := range strings.Split(tag, ",") {
		if desc, ok := strings.CutPrefix(part, "description="); ok {
			return strings.TrimSpace(desc)
		}
	}
	return ""
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
func resolveRefs(doc *Document, schema *jsonschema.Schema, endpoint string) {
	// Strip `$id`. The reflector sets it from the Go package path, but in the OpenAPI doc
	// the schema's identity is its components.schemas key. Leaving `$id` in place can clash
	// with the `$ref` wrapper pattern (OpenAPI validators reject a schema entry that has
	// both `$id` and `$ref`) and contributes nothing to consumers.
	schema.ID = ""
	if strings.HasPrefix(schema.Ref, "#/$defs/") {
		// Move $defs into the components section of the document
		// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
		schema.Ref = "#/components/schemas/" + endpoint + "_" + schema.Ref[8:]
	}
	if schema.Ref == "" && schema.Type == "" {
		// Type "any"
		schema.Examples = []any{
			struct{}{},
		}
	}
	// Recurse on nested schemas
	for defKey, defSchema := range schema.Definitions {
		doc.Components.Schemas[endpoint+"_"+defKey] = defSchema
		resolveRefs(doc, defSchema, endpoint)
	}
	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() { // Properties of a struct
		resolveRefs(doc, pair.Value, endpoint)
	}
	if schema.Items != nil { // Items of an array
		resolveRefs(doc, schema.Items, endpoint)
	}
	schema.Definitions = nil
	schema.Version = "" // Avoid rendering the $schema property
}

// jsonschemaReflectFromType generates root schema, allowing additional properties by default.
func jsonschemaReflectFromType(t reflect.Type) *jsonschema.Schema {
	r := jsonschema.Reflector{
		AllowAdditionalProperties: true,
	}
	schema := r.ReflectFromType(t)
	applyDV8Type(schema, t)
	return schema
}

// jsonschemaReflect reflects to Schema from a value, allowing additional properties by default.
func jsonschemaReflect(v any) *jsonschema.Schema {
	r := jsonschema.Reflector{
		AllowAdditionalProperties: true,
	}
	schema := r.Reflect(v)
	applyDV8Type(schema, reflect.TypeOf(v))
	return schema
}
