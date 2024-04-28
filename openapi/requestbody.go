/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiRequestBody describes a single request body.
// https://spec.openapis.org/oas/v3.1.0#request-body-object
type oapiRequestBody struct {
	Description string                    `json:"description,omitempty"`
	Required    bool                      `json:"required,omitempty"`
	Content     map[string]*oapiMediaType `json:"content,omitempty"`
}
