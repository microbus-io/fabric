package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector_EncodePath(t *testing.T) {
	testCases := []string{
		"/UPPERCASE/file.html", `.UPPERCASE.file_html`,
		"Hello/two-W0rds", `Hello.two%002dW0rds`,
		"123/abc/ABC/", `123.abc.ABC.`,
		"", ``,
	}
	for i := 0; i < len(testCases); i += 2 {
		assert.Equal(t, encodePath(testCases[i]), testCases[i+1])
	}
}

func TestConnector_SubjectOfSubscription(t *testing.T) {
	assert.Equal(t, "p0.80.com.example.|.PATH.to.file_html", subjectOfSubscription("p0", "EXAMPLE.com", 80, "PATH/to/file.html"))
	assert.Equal(t, "p0.123.com.example.|.DIR.>", subjectOfSubscription("p0", "example.com", 123, "DIR/"))
	assert.Equal(t, "p0.123.com.example.|.DIR.>", subjectOfSubscription("p0", "example.com", 123, "/DIR/"))
	assert.Equal(t, "p0.443.com.example.www.|.>", subjectOfSubscription("p0", "www.example.com", 443, "/"))
	assert.Equal(t, "p0.443.com.example.www.|._", subjectOfSubscription("p0", "www.example.com", 443, ""))
}

func TestConnector_SubjectOfRequest(t *testing.T) {
	assert.Equal(t, "p0.80.com.example.|.PATH.to.file_html", subjectOfRequest("p0", "EXAMPLE.com", 80, "PATH/to/file.html"))
	assert.Equal(t, "p0.123.com.example.|.DIR._", subjectOfRequest("p0", "example.com", 123, "DIR/"))
	assert.Equal(t, "p0.123.com.example.|.DIR._", subjectOfRequest("p0", "example.com", 123, "/DIR/"))
	assert.Equal(t, "p0.443.com.example.www.|._", subjectOfRequest("p0", "www.example.com", 443, "/"))
	assert.Equal(t, "p0.443.com.example.www.|._", subjectOfRequest("p0", "www.example.com", 443, ""))
}

func TestConnector_subjectOfResponses(t *testing.T) {
	assert.Equal(t, "p0.r.com.example.1234", subjectOfResponses("p0", "example.com", "1234"))
	assert.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.example.com", "abcd1234"))
	assert.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.EXAMPLE.com", "ABCD1234"))
}

func TestConnector_ReverseHostName(t *testing.T) {
	assert.Equal(t, "com.example.sub.www", reverseHostName("www.sub.example.com"))
	assert.Equal(t, "com.example.www", reverseHostName("www.example.com"))
	assert.Equal(t, "com.example", reverseHostName("example.com"))
	assert.Equal(t, "com", reverseHostName("com"))
	assert.Equal(t, "", reverseHostName(""))
}
