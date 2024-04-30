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

// reverseHostName reverses the order of the segments in the host name.
// www.example.com becomes com.example.www
func reverseHostName(hostName string) string {
	segments := strings.Split(hostName, ".")
	for i := 0; i < len(segments)/2; i++ {
		j := len(segments) - i - 1
		segments[i], segments[j] = segments[j], segments[i]
	}
	return strings.Join(segments, ".")
}

// subjectOfResponse is the NATS subject where a microservice subscribes to receive responses.
// For the host example.com with ID a1b2c3d4 that subject looks like microbus.r.com.example.a1b2c3d4
func subjectOfResponses(plane string, hostName string, id string) string {
	return plane + ".r." + strings.ToLower(reverseHostName(hostName)) + "." + strings.ToLower(id)
}

// subjectOfSubscription is the NATS subject where a microservice subscribes to receive incoming requests for a given path.
// For GET http://example.com:80/path/file.html the subject is microbus.80.com.example.|.GET.path.file_html .
// For a URL that ends with a / such as POST https://example.com/dir/ the subject is microbus.443.com.example.|.POST.dir.>
func subjectOfSubscription(plane string, method string, hostName string, port string, path string) string {
	var b strings.Builder
	b.WriteString(plane)
	b.WriteRune('.')
	b.WriteString(port)
	b.WriteRune('.')
	b.WriteString(strings.ToLower(reverseHostName(hostName)))
	b.WriteString(".|.")
	b.WriteString(strings.ToUpper(method))
	b.WriteRune('.')
	if path == "" {
		// Exactly the home path
		b.WriteRune('_')
		return b.String()
	}
	b.WriteString(encodePath(strings.TrimPrefix(path, "/")))
	if strings.HasSuffix(path, "/") {
		b.WriteRune('>')
	}
	return b.String()
}

// subjectOfRequest is the NATS subject where a microservice published an outgoing requests for a given path.
// For GET http://example.com:80/path/file.html that subject looks like microbus.80.com.example.|.GET.path.file_html .
// For a URL that ends with a / such as POST https://example.com/dir/ the subject is microbus.443.com.example.|.POST.dir._
// so that it is captured by the corresponding subscription microbus.443.com.example.|.POST.dir.>
func subjectOfRequest(plane string, method string, hostName string, port string, path string) string {
	subject := subjectOfSubscription(plane, method, hostName, port, path)
	if strings.HasSuffix(subject, ">") {
		subject = strings.TrimSuffix(subject, ">") + "_"
	}
	return subject
}

// escapePath escapes special characters in the path to make it suitable for appending to the subscription subject
func encodePath(path string) string {
	var b strings.Builder
	for _, ch := range path {
		switch {
		case ch == '.':
			b.WriteRune('_')
		case ch == '/':
			b.WriteRune('.')
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '*' || ch == '-':
			b.WriteRune(ch)
		default:
			b.WriteRune('%')
			b.WriteString(fmt.Sprintf("%04x", int(ch)))
		}
	}
	return b.String()
}
