package spec

import (
	"regexp"
	"strings"
	"unicode"
)

// conformDesc cleans up the description by removing back-quotes and extra spaces.
// It also guarantees that the description starts with a certain prefix and that it's not empty.
func conformDesc(desc string, ifEmpty string, mustStartWith string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		desc = ifEmpty
	}

	desc = strings.ReplaceAll(desc, "`", "'")

	if !strings.HasPrefix(desc, mustStartWith) {
		desc = mustStartWith + " - " + desc
	}

	split := strings.Split(desc, "\n")
	for i := range split {
		split[i] = strings.TrimRight(split[i], " \r\t")
	}
	return strings.Join(split, "\n")
}

var reUpperCaseIdentifier = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// isUpperCaseIdentifier accepts only lowerCaseIdentifiers.
func isUpperCaseIdentifier(id string) bool {
	return reUpperCaseIdentifier.MatchString(id)
}

var reLowerCaseIdentifier = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// isUpperCaseIdentifier accepts only UpperCaseIdentifiers.
func isLowerCaseIdentifier(id string) bool {
	return reLowerCaseIdentifier.MatchString(id)
}

// kebabCase converts a CamelCase identifier to kebab-case
func kebabCase(id string) string {
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
