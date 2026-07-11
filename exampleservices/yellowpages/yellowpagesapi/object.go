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

import (
	"context"
	"time"

	"github.com/microbus-io/errors"
)

// Person represents a person persisted in a SQL database.
// Field-level constraints are declared as dv8 tags on the fields.
type Person struct {
	Key            PersonKey `json:"key,omitzero"`
	Revision       int       `json:"revision,omitzero"`
	CreatedAt      time.Time `json:"createdAt,omitzero"`
	UpdatedAt      time.Time `json:"updatedAt,omitzero"`
	ReservedBefore time.Time `json:"reservedBefore,omitzero"`

	// HINT: Define the fields of the object here, with dv8 tags for field-level constraints
	FirstName string    `json:"firstName,omitzero" dv8:"trim,notzero,len<=64"`
	LastName  string    `json:"lastName,omitzero" dv8:"trim,notzero,len<=64"`
	Email     string    `json:"email,omitzero" dv8:"trim,notzero,len<=256"`
	Birthday  time.Time `json:"birthday,omitzero"`
	Example   string    `json:"example,omitzero" jsonschema:"-" dv8:"trim,len<=256"` // Do not remove the example
}

// Validate validates invariants that span multiple fields of the object.
// It is called automatically when the object is validated, after the dv8 field tags are enforced.
func (obj *Person) Validate(ctx context.Context) error {
	if obj == nil {
		return errors.New("nil object")
	}
	// HINT: Validate invariants that span multiple fields here as required
	if !obj.Birthday.IsZero() && obj.Birthday.After(time.Now()) {
		return errors.New("Birthday must be in the past")
	}
	return nil
}
