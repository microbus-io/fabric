/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiResponse describes a single response from an API Operation.
// https://spec.openapis.org/oas/v3.1.0#response-object
type oapiResponse struct {
	Description string                    `json:"description,omitempty"`
	Content     map[string]*oapiMediaType `json:"content,omitempty"`
}
