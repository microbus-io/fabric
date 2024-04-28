/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiServer represents a server.
// https://spec.openapis.org/oas/v3.1.0#server-object
type oapiServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}
