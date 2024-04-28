/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

import (
	"encoding/json"
	"mime"
	"regexp"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
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
	root := oapiRoot{
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
			root.Servers[0].URL = s.RemoteURI[:p+1]
		}
	}

	resolveRefs := func(schema *jsonschema.Schema, endpoint string) {
		// Resolve all $def references to the components section of the document
		for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			if strings.HasPrefix(pair.Value.Ref, "#/$defs/") {
				// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
				pair.Value.Ref = "#/components/schemas/" + endpoint + "_" + pair.Value.Ref[8:]
			}
		}

		// Move $defs into the components section of the document
		// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
		for defKey, defSchema := range schema.Definitions {
			// Resolve all nested $def references to the components section of the document
			for pair := defSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
				if strings.HasPrefix(pair.Value.Ref, "#/$defs/") {
					// #/$defs/ABC ===> #/components/schemas/ENDPOINT_ABC
					pair.Value.Ref = "#/components/schemas/" + endpoint + "_" + pair.Value.Ref[8:]
				}
			}
			root.Components.Schemas[endpoint+"_"+defKey] = defSchema
		}
		schema.Definitions = nil
	}

	for _, ep := range s.Endpoints {
		var op *oapiOperation
		var method string
		if ep.Type == "function" {
			schemaIn := jsonschema.Reflect(ep.InputArgs)
			resolveRefs(schemaIn, ep.Name+"_IN")
			schemaIn.Version = "" // Avoid rendering the $schema property
			schemaOut := jsonschema.Reflect(ep.OutputArgs)
			resolveRefs(schemaOut, ep.Name+"_OUT")
			schemaOut.Version = "" // Avoid rendering the $schema property
			for pair := schemaOut.Properties.Oldest(); pair != nil; pair = pair.Next() {
				if strings.HasPrefix(pair.Value.Ref, "#/") {
					pair.Value.Ref = "#/components/schemas/" + ep.Name + "_IN" + pair.Value.Ref[1:]
				}
			}

			root.Components.Schemas[ep.Name+"_IN"] = schemaIn
			root.Components.Schemas[ep.Name+"_OUT"] = schemaOut

			method = "post"
			op = &oapiOperation{
				Summary:     cleanEndpointSummary(ep.Summary),
				Description: ep.Description,
				RequestBody: &oapiRequestBody{
					Required: true,
					Content: map[string]*oapiMediaType{
						"application/json": {
							Schema: &jsonschema.Schema{
								Ref: "#/components/schemas/" + ep.Name + "_IN",
							},
						},
					},
				},
				Responses: map[string]*oapiResponse{
					"200": {
						Description: "OK",
						Content: map[string]*oapiMediaType{
							"application/json": {
								Schema: &jsonschema.Schema{
									Ref: "#/components/schemas/" + ep.Name + "_OUT",
								},
							},
						},
					},
				},
			}
		}
		if ep.Type == "web" {
			method = "get"
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

		subscr, err := sub.NewSub(s.ServiceName, ep.Path, nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if subscr.Host == "" {
			subscr.Host = s.ServiceName
		}
		path := "/" + subscr.Canonical()
		// if subscr.Port == 443 {
		// 	path = strings.Replace(path, ":443", "", 1)
		// }

		// Catch all subscriptions
		if strings.HasSuffix(ep.Path, "/") {
			path += "{suffix}"

			op.Parameters = append(op.Parameters, &oapiParameter{
				In:   "path",
				Name: "suffix",
				Schema: &jsonschema.Schema{
					Type: "string",
				},
				Description: "Suffix of path",
				Required:    true,
			})
		}
		root.Paths[path] = map[string]*oapiOperation{
			method: op,
		}
	}
	return json.Marshal(root)
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
