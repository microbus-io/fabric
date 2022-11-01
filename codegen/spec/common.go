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
	var sb strings.Builder
	for i, r := range id {
		if i > 0 && unicode.IsUpper(r) {
			sb.WriteByte('-')
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}
