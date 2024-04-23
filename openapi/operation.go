/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiOperation describes a single API operation on a path.
// https://spec.openapis.org/oas/v3.1.0#operation-object
type oapiOperation struct {
	Summary     string                   `yaml:"summary"`
	Description string                   `yaml:"description,omitempty"`
	Parameters  []*oapiParameter         `yaml:"parameters,omitempty"`
	RequestBody *oapiRequestBody         `yaml:"requestBody,omitempty"`
	Responses   map[string]*oapiResponse `yaml:"responses,omitempty"`
}
