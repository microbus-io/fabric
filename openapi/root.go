/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// oapiRoot is the root object of the OpenAPI document.
// https://spec.openapis.org/oas/v3.1.0#openapi-object
type oapiRoot struct {
	OpenAPI    string                               `yaml:"openapi"`
	Info       oapiInfo                             `yaml:"info"`
	Servers    []*oapiServer                        `yaml:"servers,omitempty"`
	Paths      map[string]map[string]*oapiOperation `yaml:"paths,omitempty"`
	Components *oapiComponents                      `yaml:"components,omitempty"`
}
