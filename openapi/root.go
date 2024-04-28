/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiRoot is the root object of the OpenAPI document.
// https://spec.openapis.org/oas/v3.1.0#openapi-object
type oapiRoot struct {
	OpenAPI    string                               `json:"openapi"`
	Info       oapiInfo                             `json:"info"`
	Servers    []*oapiServer                        `json:"servers,omitempty"`
	Paths      map[string]map[string]*oapiOperation `json:"paths,omitempty"`
	Components *oapiComponents                      `json:"components,omitempty"`
}
