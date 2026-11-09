/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package yellowpagesapi

// Query selects persons. Zero-valued filters are ignored, so an empty query selects every person.
type Query struct {
	FirstName string `json:"firstName,omitzero" dv8:"trim,len<=64"`
	LastName  string `json:"lastName,omitzero" dv8:"trim,len<=64"`
	Email     string `json:"email,omitzero" dv8:"trim,len<=256"`
	Limit     int    `json:"limit,omitzero" dv8:"val>=0,val<=1000" jsonschema_description:"At most 1000; 0 for 100"`
	Offset    int    `json:"offset,omitzero" dv8:"val>=0"`
}
