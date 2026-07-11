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
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/dv8"
	"github.com/microbus-io/testarossa"
)

func TestPerson_ValidateObject(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Prepare a valid object
	validObject := Person{
		// HINT: Initialize the object's fields with valid values
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     "alice@example.com",
		Birthday:  time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Example:   "Valid value",
	}
	err := dv8.Validate(ctx, &validObject)
	assert.NoError(err)

	// HINT: Check validation of individual object fields
	t.Run("first_name_required", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.FirstName = ""
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("first_name_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.FirstName = strings.Repeat("X", 65)
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("last_name_required", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.LastName = ""
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("last_name_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.LastName = strings.Repeat("X", 65)
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("email_required", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.Email = ""
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("email_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.Email = strings.Repeat("x", 250) + "@ab.com"
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("birthday_in_future", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.Birthday = time.Now().Add(24 * time.Hour)
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
	t.Run("example_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidObject := validObject
		invalidObject.Example = strings.Repeat("X", 1024) // Too long
		assert.Error(dv8.Validate(ctx, &invalidObject))
	})
}
