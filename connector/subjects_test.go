/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_EncodePath(t *testing.T) {
	testCases := []string{
		"/UPPERCASE/file.html", `.UPPERCASE.file_html`,
		"Hello/two-W0rds", `Hello.two-W0rds`,
		"123/abc/ABC/", `123.abc.ABC.`,
		"special!/character", `special%0021.character`,
		"", ``,
	}
	for i := 0; i < len(testCases); i += 2 {
		assert.Equal(t, testCases[i+1], encodePath(testCases[i]))
	}
}

func TestConnector_SubjectOfSubscription(t *testing.T) {
	assert.Equal(t, "p0.80.com.example.|.GET.PATH.to.file_html", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "80", "PATH/to/file.html"))
	assert.Equal(t, "p0.123.com.example.|.POST.DIR.>", subjectOfSubscription("p0", "POST", "example.com", "123", "DIR/"))
	assert.Equal(t, "p0.123.com.example.|.PATCH.DIR.>", subjectOfSubscription("p0", "PATCH", "example.com", "123", "/DIR/"))
	assert.Equal(t, "p0.443.com.example.www.|.DELETE.>", subjectOfSubscription("p0", "delete", "www.example.com", "443", "/"))
	assert.Equal(t, "p0.443.com.example.www.|.*._", subjectOfSubscription("p0", "ANY", "www.example.com", "443", ""))
	assert.Equal(t, "p0.*.com.example.|.GET.PATH.to.file_html", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "0", "PATH/to/file.html"))
}

func TestConnector_SubjectOfRequest(t *testing.T) {
	assert.Equal(t, "p0.80.com.example.|.GET.PATH.to.file_html", subjectOfRequest("p0", "GET", "EXAMPLE.com", "80", "PATH/to/file.html"))
	assert.Equal(t, "p0.123.com.example.|.POST.DIR._", subjectOfRequest("p0", "POST", "example.com", "123", "DIR/"))
	assert.Equal(t, "p0.123.com.example.|.PATCH.DIR._", subjectOfRequest("p0", "PATCH", "example.com", "123", "/DIR/"))
	assert.Equal(t, "p0.443.com.example.www.|.DELETE._", subjectOfRequest("p0", "delete", "www.example.com", "443", "/"))
	assert.Equal(t, "p0.443.com.example.www.|.OPTIONS._", subjectOfRequest("p0", "OPTIONS", "www.example.com", "443", ""))
	assert.Equal(t, "p0.0.com.example.|.GET.PATH.to.file_html", subjectOfRequest("p0", "GET", "EXAMPLE.com", "0", "PATH/to/file.html"))
}

func TestConnector_subjectOfResponses(t *testing.T) {
	assert.Equal(t, "p0.r.com.example.1234", subjectOfResponses("p0", "example.com", "1234"))
	assert.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.example.com", "abcd1234"))
	assert.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.EXAMPLE.com", "ABCD1234"))
}

func TestConnector_ReverseHostname(t *testing.T) {
	assert.Equal(t, "com.example.sub.www", reverseHostname("www.sub.example.com"))
	assert.Equal(t, "com.example.www", reverseHostname("www.example.com"))
	assert.Equal(t, "com.example", reverseHostname("example.com"))
	assert.Equal(t, "com", reverseHostname("com"))
	assert.Equal(t, "", reverseHostname(""))
}
