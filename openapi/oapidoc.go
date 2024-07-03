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

import "github.com/invopop/jsonschema"

// oapiDoc is the root object of the OpenAPI document.
// https://spec.openapis.org/oas/v3.1.0#openapi-object
type oapiDoc struct {
	OpenAPI    string                               `json:"openapi"`
	Info       oapiInfo                             `json:"info"`
	Servers    []*oapiServer                        `json:"servers,omitempty"`
	Paths      map[string]map[string]*oapiOperation `json:"paths,omitempty"`
	Components *oapiComponents                      `json:"components,omitempty"`
}

// oapiInfo provides metadata about the API.
// https://spec.openapis.org/oas/v3.1.0#info-object
type oapiInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// oapiServer represents a server.
// https://spec.openapis.org/oas/v3.1.0#server-object
type oapiServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// oapiOperation describes a single API operation on a path.
// https://spec.openapis.org/oas/v3.1.0#operation-object
type oapiOperation struct {
	Summary     string                   `json:"summary"`
	Description string                   `json:"description,omitempty"`
	Parameters  []*oapiParameter         `json:"parameters,omitempty"`
	RequestBody *oapiRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*oapiResponse `json:"responses,omitempty"`
}

// oapiComponents holds a set of reusable objects for different aspects of the OpenAPI schema.
// https://spec.openapis.org/oas/v3.1.0#components-object
type oapiComponents struct {
	Schemas map[string]*jsonschema.Schema `json:"schemas,omitempty"`
}

// oapiParameter describes a single operation parameter.
// https://spec.openapis.org/oas/v3.1.0#parameter-object
type oapiParameter struct {
	Name        string             `json:"name"`
	In          string             `json:"in"`
	Description string             `json:"description,omitempty"`
	Schema      *jsonschema.Schema `json:"schema,omitempty"`
	Style       string             `json:"style,omitempty"`
	Explode     bool               `json:"explode,omitempty"`
	Required    bool               `json:"required,omitempty"`
}

// oapiRequestBody describes a single request body.
// https://spec.openapis.org/oas/v3.1.0#request-body-object
type oapiRequestBody struct {
	Description string                    `json:"description,omitempty"`
	Required    bool                      `json:"required,omitempty"`
	Content     map[string]*oapiMediaType `json:"content,omitempty"`
}

// oapiMediaType provides schema and examples for the media type identified by its key.
// https://spec.openapis.org/oas/v3.1.0#media-type-object
type oapiMediaType struct {
	Description string             `json:"description,omitempty"`
	Schema      *jsonschema.Schema `json:"schema,omitempty"`
}

// oapiResponse describes a single response from an API Operation.
// https://spec.openapis.org/oas/v3.1.0#response-object
type oapiResponse struct {
	Description string                    `json:"description,omitempty"`
	Content     map[string]*oapiMediaType `json:"content,omitempty"`
}
