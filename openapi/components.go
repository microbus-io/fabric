/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

import "github.com/invopop/jsonschema"

// oapiComponents holds a set of reusable objects for different aspects of the OpenAPI schema.
// https://spec.openapis.org/oas/v3.1.0#components-object
type oapiComponents struct {
	Schemas map[string]*jsonschema.Schema `json:"schemas,omitempty"`
}
