/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapi

// Endpoint describes a single endpoint of a microservice, such as an RPC function.
type Endpoint struct {
	Type        string
	Name        string
	Path        string
	Summary     string
	Description string
	InputArgs   interface{}
	OutputArgs  interface{}
	Method      string
}
