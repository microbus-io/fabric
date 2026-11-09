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

// Person is a person in the yellow pages.
type Person struct {
	Key       int       `json:"key,omitzero" jsonschema_description:"Key is the person's key, assigned on creation"`
	FirstName string    `json:"firstName,omitzero" dv8:"trim,notzero,len<=64"`
	LastName  string    `json:"lastName,omitzero" dv8:"trim,notzero,len<=64"`
	Email     string    `json:"email,omitzero" dv8:"trim,notzero,len<=256"`
	Birthday  time.Time `json:"birthday,omitzero"`
	CreatedAt time.Time `json:"createdAt,omitzero" jsonschema_description:"CreatedAt is set by the database"`
	UpdatedAt time.Time `json:"updatedAt,omitzero" jsonschema_description:"UpdatedAt is set by the database"`
}

// Validate checks the invariants that span more than one field. It runs after the dv8 field tags are enforced.
func (p *Person) Validate(ctx context.Context) error {
	if p == nil {
		return errors.New("nil person")
	}
	if !p.Birthday.IsZero() && p.Birthday.After(time.Now()) {
		return errors.New("birthday must be in the past")
	}
	return nil
}
