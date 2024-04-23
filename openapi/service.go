/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

import (
	"mime"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/fabric/sub"
)

// Service is populated with the microservice's specs in order to genreate its OpenAPI document.
type Service struct {
	ServiceName string
	Description string
	Version     int
	Endpoints   []*Endpoint
	RemoteURI   string
}

// MarshalYAML produces the YAML representation of the OpenAPI (Swagger) of the service.
func (s *Service) MarshalYAML() (interface{}, error) {
	root := oapiRoot{
		OpenAPI: "3.0.0",
		Info: oapiInfo{
			Title:       s.ServiceName,
			Description: s.Description,
			Version:     strconv.Itoa(s.Version),
		},
		Paths: map[string]map[string]*oapiOperation{},
		Components: &oapiComponents{
			Schemas: map[string]*oapiSchema{},
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
			root.Servers[0].URL = s.RemoteURI[:p+1]
		}
	}

	for _, ep := range s.Endpoints {
		subscr, err := sub.NewSub(s.ServiceName, ep.Path, nil)
		if err == nil {
			if subscr.Host == "" {
				subscr.Host = s.ServiceName
			}
			ep.Path = "/" + subscr.Canonical()
			// if subscr.Port == 443 {
			// 	ep.Path = strings.Replace(ep.Path, ":443", "", 1)
			// }
		}
		// Catch all subscriptions
		if strings.HasSuffix(ep.Path, "/") {
			ep.Path += "{subpath}"
		}
		root.Paths[ep.Path] = map[string]*oapiOperation{}
	}

	components := map[string]*oapiSchema{}
	for _, ep := range s.Endpoints {
		inType := reflect.TypeOf(ep.InputArgs)
		inSchema := schemaOf(inType)
		inSchema.Ref += ep.Name + "_in"
		inComponentSchema := schemaOfStruct(inType)

		outType := reflect.TypeOf(ep.OutputArgs)
		outSchema := schemaOf(outType)
		outSchema.Ref += ep.Name + "_out"
		outComponentSchema := schemaOfStruct(outType)

		// Collect referenced components
		components[ep.Name+"_in"] = inComponentSchema
		components[ep.Name+"_out"] = outComponentSchema
		collectComponentsIn(components, inType)
		collectComponentsIn(components, outType)

		// GET path (no body, args in query string)
		getPath := &oapiOperation{
			Summary:     cleanEndpointSummary(ep.Summary),
			Description: ep.Description,
			Parameters:  []*oapiParameter{},
		}
		for i := 0; i < inType.NumField(); i++ {
			field := inType.Field(i)
			name := fieldName(field)
			if name == "" {
				continue
			}

			parameter := &oapiParameter{
				In:     "query",
				Name:   name,
				Schema: schemaOf(field.Type),
			}
			if parameter.Schema.Type == "" { // Non-primitive type
				parameter.Style = "deepObject"
				parameter.Explode = true
				// See https://swagger.io/docs/specification/serialization/
			}
			getPath.Parameters = append(getPath.Parameters, parameter)
		}

		// POST path (args in body)
		postPath := &oapiOperation{
			Summary:     cleanEndpointSummary(ep.Summary),
			Description: ep.Description,
			RequestBody: &oapiRequestBody{
				Required: true,
				Content: map[string]*oapiMediaType{
					"application/json": {
						Schema: inSchema,
					},
				},
			},
		}

		// Catch-all subscriptions
		if strings.HasSuffix(ep.Path, "/{subpath}") {
			parameter := &oapiParameter{
				In:   "path",
				Name: "subpath",
				Schema: &oapiSchema{
					Type: "string",
				},
				Description: "Sub-path for catch-all subscription",
				Required:    true,
			}
			getPath.Parameters = append(getPath.Parameters, parameter)
			postPath.Parameters = append(postPath.Parameters, parameter)
		}

		// Response is same for GET and POST
		responses := map[string]*oapiResponse{
			// "403": &swaggerResponse{
			// 	Description: "Unauthorized",
			// 	Content: map[string]*swaggerContent{
			// 		"text/plain": &swaggerContent{
			// 			Schema: &swaggerSchema{
			// 				Type: "string",
			// 			},
			// 		},
			// 	},
			// },
			// "5XX": &swaggerResponse{
			// 	Description: "Error",
			// 	Content: map[string]*swaggerContent{
			// 		"text/plain": &swaggerContent{
			// 			Schema: &swaggerSchema{
			// 				Type: "string",
			// 			},
			// 		},
			// 	},
			// },
		}
		if ep.Type == "function" {
			responses["200"] = &oapiResponse{
				Description: "OK",
				Content: map[string]*oapiMediaType{
					"application/json": {
						Schema: outSchema,
					},
				},
			}
		}
		if ep.Type == "web" {
			responses["200"] = &oapiResponse{
				Description: "OK",
			}
			p := strings.LastIndex(ep.Path, ".")
			if p >= 0 {
				contentType := mime.TypeByExtension(ep.Path[p:])
				if contentType != "" {
					responses["200"].Content = map[string]*oapiMediaType{
						contentType: {},
					}
				}
			}
		}
		getPath.Responses = responses
		postPath.Responses = responses

		// Method
		method := ep.Method
		if method == "" {
			if ep.Type == "web" {
				method = "GET"
			} else {
				method = "POST"
			}
		}
		if methodHasBody(method) {
			root.Paths[ep.Path][strings.ToLower(method)] = postPath
		} else {
			root.Paths[ep.Path][strings.ToLower(method)] = getPath
		}
	}

	// Components
	for name, schema := range components {
		root.Components.Schemas[name] = schema
	}
	root.Components.Schemas["Nullable"] = &oapiSchema{
		Type: "null",
	}

	return root, nil
}

func collectComponentsIn(collector map[string]*oapiSchema, t reflect.Type) {
	if t == reflect.TypeOf(time.Time{}) {
		return
	}

	switch t.Kind() {
	case reflect.Struct:
		name := typeFullName(t)
		if name != "" {
			collector[name] = schemaOfStruct(t)
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := fieldName(field)
			if name == "" {
				continue
			}
			collectComponentsIn(collector, field.Type)
		}
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Slice:
		collectComponentsIn(collector, t.Elem())
	}
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
