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

// subjectOfReply is the NATS subject where a microservice subscribes to receive replies.
// For the host example.com with ID a1b2c3d4 that subject looks like r.com.example.a1b2c3d4
func subjectOfReply(hostName string, id string) string {
	return "r." + strings.ToLower(reverseHostName(hostName)) + "." + strings.ToLower(id)
}

// subjectOfSubscription is the NATS subject where a microserve subscribes to receive incoming requests for a given path.
// For the URL http://example.com:80/PATH/file.html that subject looks like 80.com.example.|.PATH.file_html .
// For a URL to that ends with a / such as https://example.com/dir/ the subject looks like 443.com.example.|.dir._
func subjectOfSubscription(hostName string, port int, path string) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(port))
	b.WriteRune('.')
	b.WriteString(strings.ToLower(reverseHostName(hostName)))
	b.WriteString(".|.")
	b.WriteString(encodePath(strings.TrimPrefix(path, "/")))
	if path == "" || strings.HasSuffix(path, "/") {
		b.WriteRune('_')
	}
	return b.String()
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
