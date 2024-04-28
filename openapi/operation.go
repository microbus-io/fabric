/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiOperation describes a single API operation on a path.
// https://spec.openapis.org/oas/v3.1.0#operation-object
type oapiOperation struct {
	Summary     string                   `json:"summary"`
	Description string                   `json:"description,omitempty"`
	Parameters  []*oapiParameter         `json:"parameters,omitempty"`
	RequestBody *oapiRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*oapiResponse `json:"responses,omitempty"`
}
