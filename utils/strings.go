/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var reUpperCaseIdentifier = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// IsUpperCaseIdentifier accepts only lowerCaseIdentifiers.
func IsUpperCaseIdentifier(id string) bool {
	return reUpperCaseIdentifier.MatchString(id)
}

var reLowerCaseIdentifier = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// IsLowerCaseIdentifier accepts only UpperCaseIdentifiers.
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
