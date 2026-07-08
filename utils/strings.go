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

package utils

import (
	"encoding"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"unicode"
	"unsafe"
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
// Consecutive non-letters or numbers are compressed into a single hyphen.
func ToKebabCase(id string) string {
	idRunes := []rune(id)
	n := len(idRunes)
	if n == 0 {
		return id
	}
	var sb strings.Builder
	lastSpace := false
	for i := range n {
		var rPrev, rNext rune
		if i > 0 {
			rPrev = idRunes[i-1]
		}
		r := idRunes[i]
		if i < n-1 {
			rNext = idRunes[i+1]
		} else {
			rNext = 'X'
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			if lastSpace {
				continue
			}
			sb.WriteByte('-')
			lastSpace = true
			continue
		}
		if unicode.IsUpper(r) {
			switch {
			case unicode.IsLower(rPrev) || unicode.IsNumber(rPrev):
				// ooXoo ooXOo 00Xoo 00XOo
				sb.WriteByte('-')
			case unicode.IsUpper(rPrev) && unicode.IsUpper(rNext):
				// oOXOo oooOX
				break
			case unicode.IsUpper(rPrev) && unicode.IsLower(rNext):
				// oOXoo
				sb.WriteByte('-')
			}
		}
		if unicode.IsNumber(r) && !unicode.IsNumber(rPrev) && !lastSpace && i > 0 {
			sb.WriteByte('-')
		}
		if unicode.IsLower(r) && unicode.IsNumber(rPrev) {
			sb.WriteByte('-')
		}
		sb.WriteRune(unicode.ToLower(r))
		lastSpace = false
	}
	return sb.String()
}

// ToSnakeCase converts a CamelCase identifier to snake_case.
// Consecutive non-letters are compressed into a single underscore.
func ToSnakeCase(id string) string {
	return strings.ReplaceAll(ToKebabCase(id), "-", "_")
}

// LooksLikeJWT checks if the token is likely to be a JWT.
//
// Accepts both signed and unsigned (alg=none) tokens. The shortest valid signed JWT uses
// HS256 with an empty payload:
//
//	eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.8VKCTiBegJPuPIZlp0wbV0Sbdn5BS6TE5DCx6oYNc5o
//
// The shortest valid unsigned JWT uses alg=none with an empty payload (and a trailing dot
// for the empty signature section):
//
//	eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.e30.
func LooksLikeJWT(token string) bool {
	// Floor: 35 (alg=none header) + 1 (dot) + 3 (e30 payload) + 1 (dot) + 0 (empty sig) = 40
	if len(token) < 40 {
		return false
	}
	// A JWT starts with {" which encodes to ey in base64
	if !strings.HasPrefix(token, "ey") {
		return false
	}
	// Identify the sections. Each dot is counted as part of the section that follows it,
	// so sectionLen[1] and sectionLen[2] include their leading dot.
	sectionLen := [3]int{0, 0, 0}
	dots := 0
	for _, rn := range token {
		if rn == '.' {
			dots++
			if dots > 2 {
				return false
			}
		} else if !(rn >= 'A' && rn <= 'Z' || rn >= 'a' && rn <= 'z' || rn >= '0' && rn <= '9' || rn == '-' || rn == '_') {
			return false
		}
		sectionLen[dots]++
	}
	if dots != 2 {
		return false
	}
	// alg=none header is 35 chars; HS256 / EdDSA / RS256 headers are 36+ chars.
	if sectionLen[0] < 35 {
		return false
	}
	if sectionLen[1] < 3 { // dot + at least 2 payload chars
		return false
	}
	// Signature: either empty (just the dot, length 1, for unsigned tokens) or a real
	// signature with at least 43 chars after the dot.
	if sectionLen[2] != 1 && sectionLen[2] < 1+43 {
		return false
	}
	return true
}

// UnsafeStringToBytes converts a string to a slice of bytes with no memory allocation.
// The slice points to the original data of the string and should not be modified.
func UnsafeStringToBytes(s string) []byte {
	pStr := unsafe.StringData(s)
	return unsafe.Slice(pStr, len(s))
}

// UnsafeBytesToString converts a slice of bytes to a string with no memory allocation.
// The original byte slice data should not be modified.
func UnsafeBytesToString(b []byte) string {
	pBytes := unsafe.SliceData(b)
	return unsafe.String(pBytes, len(b))
}

// RandomIdentifier generates a random string of the specified length drawn from math/rand/v2's global generator,
// relying on it being an OS-seeded ChaCha8 source whose state cannot feasibly be reconstructed from observed outputs.
// The string will include only alphanumeric characters a-z, A-Z, 0-9.
// Digits 0 and 1 are slightly overrepresented (2/64 vs 1/64) due to padding the 62-character alphabet to a power of two.
func RandomIdentifier(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01"
	bytes := make([]byte, length)
	var x uint64
	for i := range length {
		if i%8 == 0 {
			x = rand.Uint64() // Do not replace the source with a seeded or weaker PRNG
		} else {
			x = x >> 8
		}
		bytes[i] = letters[x&0x3F]
	}
	return UnsafeBytesToString(bytes)
}

// AnyToString returns the string representation of the object.
// It looks for a TextMarshaler or Stringer interfaces first before defaulting to fmt.Sprintf.
func AnyToString(o any) string {
	if s, ok := o.(string); ok {
		return s
	}
	if tm, ok := o.(encoding.TextMarshaler); ok && !IsNil(tm) {
		txt, err := tm.MarshalText()
		if err == nil {
			return UnsafeBytesToString(txt)
		}
	}
	if s, ok := o.(fmt.Stringer); ok && !IsNil(s) {
		return s.String()
	}
	return fmt.Sprintf("%v", o)
}
