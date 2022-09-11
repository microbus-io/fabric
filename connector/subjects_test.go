package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodePath(t *testing.T) {
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

func TestSubjectOfSubscription(t *testing.T) {
	assert.Equal(t, "80.com.example.|.PATH.to.file_html", subjectOfSubscription("EXAMPLE.com", 80, "PATH/to/file.html"))
	assert.Equal(t, "123.com.example.|.DIR.>", subjectOfSubscription("example.com", 123, "DIR/"))
	assert.Equal(t, "123.com.example.|.DIR.>", subjectOfSubscription("example.com", 123, "/DIR/"))
	assert.Equal(t, "443.com.example.www.|.>", subjectOfSubscription("www.example.com", 443, "/"))
	assert.Equal(t, "443.com.example.www.|._", subjectOfSubscription("www.example.com", 443, ""))
}

func TestSubjectOfRequest(t *testing.T) {
	assert.Equal(t, "80.com.example.|.PATH.to.file_html", subjectOfRequest("EXAMPLE.com", 80, "PATH/to/file.html"))
	assert.Equal(t, "123.com.example.|.DIR._", subjectOfRequest("example.com", 123, "DIR/"))
	assert.Equal(t, "123.com.example.|.DIR._", subjectOfRequest("example.com", 123, "/DIR/"))
	assert.Equal(t, "443.com.example.www.|._", subjectOfRequest("www.example.com", 443, "/"))
	assert.Equal(t, "443.com.example.www.|._", subjectOfRequest("www.example.com", 443, ""))
}

func TestSubjectOfReply(t *testing.T) {
	assert.Equal(t, "r.com.example.1234", subjectOfReply("example.com", "1234"))
	assert.Equal(t, "r.com.example.www.abcd1234", subjectOfReply("www.example.com", "abcd1234"))
	assert.Equal(t, "r.com.example.www.abcd1234", subjectOfReply("www.EXAMPLE.com", "ABCD1234"))
}

func TestReverseHostName(t *testing.T) {
	assert.Equal(t, "com.example.sub.www", reverseHostName("www.sub.example.com"))
	assert.Equal(t, "com.example.www", reverseHostName("www.example.com"))
	assert.Equal(t, "com.example", reverseHostName("example.com"))
	assert.Equal(t, "com", reverseHostName("com"))
	assert.Equal(t, "", reverseHostName(""))
}
