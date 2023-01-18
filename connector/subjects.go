/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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
	"strconv"
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
// For the URL http://example.com:80/PATH/file.html that subject is microbus.80.com.example.|.PATH.file_html .
// For a URL to that ends with a / such as https://example.com/dir/ the subject is microbus.443.com.example.|.dir.>
func subjectOfSubscription(plane string, hostName string, port int, path string) string {
	var b strings.Builder
	b.WriteString(plane)
	b.WriteRune('.')
	b.WriteString(strconv.Itoa(port))
	b.WriteRune('.')
	b.WriteString(strings.ToLower(reverseHostName(hostName)))
	b.WriteString(".|.")
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
// For the URL http://example.com:80/PATH/file.html that subject looks like microbus.80.com.example.|.PATH.file_html .
// For a URL to that ends with a / such as https://example.com/dir/ the subject is microbus.443.com.example.|.dir._
// so that it is captured by the corresponding subscription microbus.443.com.example.|.dir.>
func subjectOfRequest(plane string, hostName string, port int, path string) string {
	subject := subjectOfSubscription(plane, hostName, port, path)
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
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9'):
			b.WriteRune(ch)
		default:
			b.WriteRune('%')
			b.WriteString(fmt.Sprintf("%04x", int(ch)))
		}
	}
	return b.String()
}
