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

package connector

import (
	"net/url"
	"strings"
)

const (
	idPrefix       = "id-"
	localityPrefix = "loc-"
)

// maxSubjectLength caps the length of a request's NATS subject. NATS recommends keeping subjects
// short (well under 256 characters); the cap is generous to accommodate long hostname+path
// combinations while still bounding attacker-lengthened URLs before they reach the bus.
const maxSubjectLength = 1024

// subjectHexDigits supplies the lowercase hex alphabet used for percent-encoded
// bytes in NATS subject segments. Lowercase blends with the lowercase hostname
// segments and the rest of the subject; url.PathUnescape accepts both cases on
// the decode path so this is a producer-side aesthetic choice, not a contract
// constraint.
const subjectHexDigits = "0123456789abcdef"

// escapeHostname encodes a hostname for placement in a single NATS subject segment.
//
// Encoding rules:
//   - '.' becomes '_' (the segment-separator workaround the framework has used since
//     v1; preserves readability for typical service identities like "payments.core").
//   - URL-special characters that the route validator allows (e.g. '$', '!', '~')
//     are percent-encoded as %XX with uppercase hex digits.
//   - '_' and the canonical alphanumerics + '-' pass through unchanged.
//
// The asymmetry between '.' (gets the legacy `_` flattening) and other specials
// (get `%XX`) is intentional: typical hostnames carry only `.` and stay readable
// in subjects, while the rare special-char route hostname remains representable.
//
// Note: '_' in the input is preserved (no encoding), so a route hostname with a
// literal underscore can collide with one whose dot maps to underscore. The
// framework forbids '_' in service identities to avoid this collision; route
// hostnames inherit the legacy ambiguity for compatibility.
func escapeHostname(hostname string) string {
	if !needsHostnameEscape(hostname) {
		return hostname
	}
	var sb strings.Builder
	sb.Grow(len(hostname) + 8)
	appendHostnameEscaped(&sb, hostname)
	return sb.String()
}

// unescapeHostname is the inverse of escapeHostname. It first reverses the
// '_' → '.' substitution, then percent-decodes any %XX sequences. Returns the
// input as-is on a malformed escape - the framework only consumes its own
// output, so failure indicates a non-framework subject.
func unescapeHostname(escaped string) string {
	if strings.IndexByte(escaped, '_') < 0 && strings.IndexByte(escaped, '%') < 0 {
		return escaped
	}
	s := strings.ReplaceAll(escaped, "_", ".")
	decoded, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}

// splitSubject parses a NATS subject into its first six segments. srcHostname
// and hostname are returned percent-decoded; the other strings are sub-slices
// of subject. A malformed subject (fewer than six segments) produces zero
// values for the missing fields.
func splitSubject(subject string) (plane, trust, port, srcHostname, hostname, idOrLocality string) {
	s := subject
	var i int
	if i = strings.IndexByte(s, '.'); i < 0 {
		return
	}
	plane, s = s[:i], s[i+1:]
	if i = strings.IndexByte(s, '.'); i < 0 {
		return
	}
	trust, s = s[:i], s[i+1:]
	if i = strings.IndexByte(s, '.'); i < 0 {
		return
	}
	port, s = s[:i], s[i+1:]
	if i = strings.IndexByte(s, '.'); i < 0 {
		return
	}
	srcHostname, s = unescapeHostname(s[:i]), s[i+1:]
	if i = strings.IndexByte(s, '.'); i < 0 {
		return
	}
	hostname, s = unescapeHostname(s[:i]), s[i+1:]
	if i = strings.IndexByte(s, '.'); i < 0 {
		idOrLocality = s
	} else {
		idOrLocality = s[:i]
	}
	return
}

// cutIDOrLocality strips a reserved id- or loc- prefix from the hostname's
// first segment and returns it as the slot value. Returns the hostname
// unchanged with an empty slot when no reserved prefix is present.
func cutIDOrLocality(hostname string) (host, idOrLocality string) {
	if i := strings.IndexByte(hostname, '.'); i > 0 {
		first := hostname[:i]
		lower := strings.ToLower(first)
		if strings.HasPrefix(lower, idPrefix) || strings.HasPrefix(lower, localityPrefix) {
			return hostname[i+1:], lower
		}
	}
	return hostname, ""
}

// escapeLocality wraps a hyphen-form locality prefix (e.g. "us-west") in its
// slot form (e.g. "loc-us-west"). Returns an empty string for an empty input.
func escapeLocality(locality string) string {
	if locality == "" {
		return ""
	}
	return localityPrefix + strings.ToLower(locality)
}

// subjectOfResponseSub is the subject a microservice subscribes to in order to
// receive responses to its outgoing requests. Source is wildcarded.
func subjectOfResponseSub(plane string, hostname string, id string) string {
	flatHost := escapeHostname(strings.ToLower(hostname))
	lowID := strings.ToLower(id)
	var sb strings.Builder
	sb.Grow(len(plane) + len(flatHost) + len(lowID) + 12)
	sb.WriteString(plane)
	sb.WriteString(".reply._.*.")
	sb.WriteString(flatHost)
	sb.WriteRune('.')
	sb.WriteString(lowID)
	return sb.String()
}

// subjectOfResponse is the subject a microservice publishes a response to.
// Argument order mirrors the on-wire segment order.
func subjectOfResponse(plane, srcHostname, hostname, id string) string {
	flatSrc := escapeHostname(strings.ToLower(srcHostname))
	flatHost := escapeHostname(strings.ToLower(hostname))
	lowID := strings.ToLower(id)
	var sb strings.Builder
	sb.Grow(len(plane) + len(flatSrc) + len(flatHost) + len(lowID) + 12)
	sb.WriteString(plane)
	sb.WriteString(".reply._.")
	sb.WriteString(flatSrc)
	sb.WriteRune('.')
	sb.WriteString(flatHost)
	sb.WriteRune('.')
	sb.WriteString(lowID)
	return sb.String()
}

// SubjectOfRequestSub is the subject a microservice subscribes to in order to
// receive incoming requests for a given route. Source is wildcarded; port 0 and
// method "ANY" become wildcards. Exposed as the runtime's contract: external
// tools (notably cmd/genacl) compute their NATS subject patterns to match
// this exact output.
func SubjectOfRequestSub(plane, port, hostname, idOrLocality, method, path string) string {
	return subjectOf(true, plane, port, "", hostname, idOrLocality, method, path)
}

// SubjectOfRequest is the subject a microservice publishes an outgoing request to.
// Argument order mirrors the on-wire segment order. Exposed alongside
// SubjectOfRequestSub for cross-tool wire-format verification.
func SubjectOfRequest(plane, port, srcHostname, hostname, idOrLocality, method, path string) string {
	return subjectOf(false, plane, port, srcHostname, hostname, idOrLocality, method, path)
}

// subjectOf composes the NATS subject of subscriptions and requests. Argument
// order mirrors the on-wire segment order. The wildcards flag selects between
// the subscription and publish forms (subscriptions wildcard source, port 0,
// and method ANY).
func subjectOf(wildcards bool, plane, port, srcHostname, hostname, idOrLocality, method, path string) string {
	var sb strings.Builder
	sb.Grow(len(plane) + len(port) + len(srcHostname) + len(hostname) + len(idOrLocality) + len(method) + len(path) + 24)
	sb.WriteString(plane)
	sb.WriteRune('.')
	if port == "666" {
		sb.WriteString("danger")
	} else {
		sb.WriteString("safe")
	}
	sb.WriteRune('.')
	if wildcards && port == "0" {
		sb.WriteRune('*') // Any port
	} else {
		sb.WriteString(port)
	}
	sb.WriteRune('.')
	if wildcards {
		sb.WriteRune('*') // Any source
	} else if srcHostname != "" {
		sb.WriteString(escapeHostname(strings.ToLower(srcHostname)))
	} else {
		sb.WriteRune('_')
	}
	sb.WriteRune('.')
	sb.WriteString(escapeHostname(strings.ToLower(hostname)))
	sb.WriteRune('.')
	if idOrLocality == "" {
		sb.WriteRune('_')
	} else {
		sb.WriteString(strings.ToLower(idOrLocality))
	}
	sb.WriteRune('.')
	method = strings.ToUpper(method)
	if wildcards && method == "ANY" {
		sb.WriteRune('*') // Any method
	} else {
		sb.WriteString(method)
	}
	sb.WriteRune('.')
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		// Exactly the root path, which could come with or without a slash
		sb.WriteRune('_')
		return sb.String()
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		if i > 0 {
			sb.WriteRune('.')
		}
		if wildcards && strings.HasPrefix(parts[i], "{") && strings.HasSuffix(parts[i], "}") {
			if i == len(parts)-1 && strings.HasSuffix(parts[i], "...}") {
				// Greedy
				sb.WriteRune('>')
			} else {
				sb.WriteRune('*')
			}
			continue
		}
		if wildcards && parts[i] == "*" {
			sb.WriteRune('*')
			continue
		}
		if parts[i] == "" {
			sb.WriteRune('_')
		} else {
			escapePathPart(&sb, parts[i])
		}
	}
	return sb.String()
}

// escapePathPart percent-encodes a single path segment for inclusion in a NATS
// subject. Every byte outside [A-Za-z0-9-] becomes %XX with uppercase hex
// digits, including '.', '_', and any URL-special character. Path segments are
// case-sensitive: uppercase letters pass through unchanged.
//
// Empty segments and wildcards (`{name}`, `{name...}`, bare `*`) are handled
// by the caller, not here.
func escapePathPart(b *strings.Builder, part string) {
	for i := 0; i < len(part); i++ {
		c := part[i]
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '-':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(subjectHexDigits[c>>4])
			b.WriteByte(subjectHexDigits[c&0xF])
		}
	}
}

// needsHostnameEscape reports whether the hostname requires escaping for use
// as a NATS subject segment (i.e., contains any byte outside [A-Za-z0-9_-]).
func needsHostnameEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			continue
		}
		return true
	}
	return false
}

// appendHostnameEscaped writes the hostname-encoded form of s to b.
func appendHostnameEscaped(b *strings.Builder, s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.':
			b.WriteByte('_')
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(subjectHexDigits[c>>4])
			b.WriteByte(subjectHexDigits[c&0xF])
		}
	}
}
