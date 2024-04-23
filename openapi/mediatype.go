/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiMediaType provides schema and examples for the media type identified by its key.
// https://spec.openapis.org/oas/v3.1.0#media-type-object
type oapiMediaType struct {
	Description string      `yaml:"description,omitempty"`
	Schema      *oapiSchema `yaml:"schema,omitempty"`
}
