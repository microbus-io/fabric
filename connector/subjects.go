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

package connector

import (
	"fmt"
	"strings"
)

// reverseHostname reverses the order of the segments in the hostname.
// www.example.com becomes com.example.www
func reverseHostname(hostname string) string {
	var sb strings.Builder
	sb.Grow(len(hostname))
	for {
		p := strings.LastIndex(hostname, ".")
		if p < 0 {
			sb.WriteString(hostname)
			break
		}
		sb.WriteString(hostname[p+1:])
		sb.WriteRune('.')
		hostname = hostname[:p]
	}
	return sb.String()
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
	var sb strings.Builder
	sb.Grow(len(plane) + len(method) + len(hostname) + len(port) + len(path) + 16)
	sb.WriteString(plane)
	sb.WriteRune('.')
	if wildcards && port == "0" {
		sb.WriteString("*")
	} else {
		sb.WriteString(port)
	}
	sb.WriteRune('.')
	sb.WriteString(strings.ToLower(reverseHostname(hostname)))
	sb.WriteString(".|.")
	method = strings.ToUpper(method)
	if wildcards && method == "ANY" {
		sb.WriteString("*")
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
			if i == len(parts)-1 && strings.HasSuffix(parts[i], "+}") {
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
