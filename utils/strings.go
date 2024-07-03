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

package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var reUpperCaseIdentifier = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// IsUpperCaseIdentifier accepts only UpperCaseIdentifiers.
func IsUpperCaseIdentifier(id string) bool {
	return reUpperCaseIdentifier.MatchString(id)
}

var reLowerCaseIdentifier = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// IsLowerCaseIdentifier accepts only lowerCaseIdentifiers.
func IsLowerCaseIdentifier(id string) bool {
	return reLowerCaseIdentifier.MatchString(id)
}

// ToKebabCase converts a CamelCase identifier to kebab-case.
func ToKebabCase(id string) string {
	idRunes := []rune(id)
	n := len(idRunes)
	if n == 0 {
		return id
	}
	idRunes = append(idRunes, rune('x')) // Terminal
	var sb strings.Builder
	sb.WriteRune(unicode.ToLower(idRunes[0]))

	for i := 1; i < n; i++ {
		rPrev := idRunes[i-1]
		r := idRunes[i]
		rNext := idRunes[i+1]
		if unicode.IsUpper(r) {
			switch {
			case unicode.IsLower(rPrev) && unicode.IsLower(rNext):
				// ooXoo
				sb.WriteByte('-')
			case unicode.IsUpper(rPrev) && unicode.IsUpper(rNext):
				// oOXOo
				break
			case unicode.IsUpper(rPrev) && unicode.IsLower(rNext):
				if i < n-1 {
					// oOXoo
					sb.WriteByte('-')
				} else {
					// oooOX
					break
				}
			case unicode.IsLower(rPrev) && unicode.IsUpper(rNext):
				// ooXOo
				sb.WriteByte('-')
			}
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}

// ToSnakeCase converts a CamelCase identifier to snake_case.
func ToSnakeCase(id string) string {
	return strings.ReplaceAll(ToKebabCase(id), "-", "_")
}
