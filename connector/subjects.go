/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"fmt"
	"strings"
)

// reverseHostname reverses the order of the segments in the hostname.
// www.example.com becomes com.example.www
func reverseHostname(hostname string) string {
	segments := strings.Split(hostname, ".")
	for i := 0; i < len(segments)/2; i++ {
		j := len(segments) - i - 1
		segments[i], segments[j] = segments[j], segments[i]
	}
	return strings.Join(segments, ".")
}

// subjectOfResponse is the NATS subject where a microservice subscribes to receive responses.
// For the host example.com with ID a1b2c3d4 that subject looks like microbus.r.com.example.a1b2c3d4
func subjectOfResponses(plane string, hostname string, id string) string {
	return plane + ".r." + strings.ToLower(reverseHostname(hostname)) + "." + strings.ToLower(id)
}

// subjectOfSubscription is the NATS subject where a microservice subscribes to receive incoming requests for a given path.
// For GET http://example.com:80/path/file.html the subject is microbus.80.com.example.|.GET.path.file_html .
// For a URL that ends with a / such as POST https://example.com/dir/ the subject is microbus.443.com.example.|.POST.dir.> .
func subjectOfSubscription(plane string, method string, hostname string, port string, path string) string {
	return subjectOf(true, plane, method, hostname, port, path)
}

// subjectOfRequest is the NATS subject where a microservice published an outgoing requests for a given path.
// For GET http://example.com:80/path/file.html that subject looks like microbus.80.com.example.|.GET.path.file_html .
// For a URL that ends with a / such as POST https://example.com/dir/ the subject is microbus.443.com.example.|.POST.dir._
// so that it is captured by the corresponding subscription microbus.443.com.example.|.POST.dir.>
func subjectOfRequest(plane string, method string, hostname string, port string, path string) string {
	return subjectOf(false, plane, method, hostname, port, path)
}

// subjectOf composes the NATS subject of subscriptions and requests.
func subjectOf(wildcards bool, plane string, method string, hostname string, port string, path string) string {
	var b strings.Builder
	b.WriteString(plane)
	b.WriteRune('.')
	if wildcards && port == "0" {
		b.WriteString("*")
	} else {
		b.WriteString(port)
	}
	b.WriteRune('.')
	b.WriteString(strings.ToLower(reverseHostname(hostname)))
	b.WriteString(".|.")
	method = strings.ToUpper(method)
	if wildcards && method == "ANY" {
		b.WriteString("*")
	} else {
		b.WriteString(method)
	}
	b.WriteRune('.')
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		// Exactly the home path
		b.WriteRune('_')
		return b.String()
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		if i > 0 {
			b.WriteRune('.')
		}
		if wildcards && strings.HasPrefix(parts[i], "{") && strings.HasSuffix(parts[i], "}") {
			if i == len(parts)-1 && strings.HasSuffix(parts[i], "+}") {
				// Greedy
				b.WriteRune('>')
			} else {
				b.WriteRune('*')
			}
			continue
		}
		if wildcards && parts[i] == "*" {
			b.WriteRune('*')
			continue
		}
		escapePathPart(&b, parts[i])
	}
	return b.String()
}

// escapePathPart escapes special characters in the path to make it suitable for inclusion in the subscription subject.
func escapePathPart(b *strings.Builder, part string) {
	for _, ch := range part {
		switch {
		case ch == '.':
			b.WriteRune('_')
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-':
			b.WriteRune(ch)
		default:
			b.WriteRune('%')
			b.WriteString(fmt.Sprintf("%04x", int(ch)))
		}
	}
}
