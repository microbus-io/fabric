/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

import "github.com/invopop/jsonschema"

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
