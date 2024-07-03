/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
