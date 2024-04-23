/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiParameter describes a single operation parameter.
// https://spec.openapis.org/oas/v3.1.0#parameter-object
type oapiParameter struct {
	Name        string      `yaml:"name"`
	In          string      `yaml:"in"`
	Description string      `yaml:"description,omitempty"`
	Schema      *oapiSchema `yaml:"schema,omitempty"`
	Style       string      `yaml:"style,omitempty"`
	Explode     bool        `yaml:"explode,omitempty"`
	Required    bool        `yaml:"required,omitempty"`
}
