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

	"github.com/microbus-io/dv8"
	"github.com/microbus-io/testarossa"
)

func TestPerson_ValidateQuery(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert := testarossa.For(t)

	// Prepare a valid query
	validQuery := Query{
		// HINT: Initialize the query's fields with valid values
		Example: "Valid value",
	}
	err := dv8.Validate(ctx, &validQuery)
	assert.NoError(err)

	// HINT: Check validation of individual query fields
	t.Run("first_name_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidQuery := validQuery
		invalidQuery.FirstName = strings.Repeat("X", 65)
		assert.Error(dv8.Validate(ctx, &invalidQuery))
	})
	t.Run("last_name_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidQuery := validQuery
		invalidQuery.LastName = strings.Repeat("X", 65)
		assert.Error(dv8.Validate(ctx, &invalidQuery))
	})
	t.Run("email_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidQuery := validQuery
		invalidQuery.Email = strings.Repeat("x", 257)
		assert.Error(dv8.Validate(ctx, &invalidQuery))
	})
	t.Run("example_too_long", func(t *testing.T) {
		assert := testarossa.For(t)
		invalidQuery := validQuery
		invalidQuery.Example = strings.Repeat("X", 1024) // Too long
		assert.Error(dv8.Validate(ctx, &invalidQuery))
	})
}
