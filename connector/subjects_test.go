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
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestConnector_EncodePathPart(t *testing.T) {
	testCases := []string{
		"UPPERCASE", "UPPERCASE",
		"file.html", "file_html",
		"a_b_c", "a%005fb%005fc",
		"two-W0rds", "two-W0rds",
		"special!", "special%0021",
		"asteri*", "asteri%002a",
		"", "",
	}
	for i := 0; i < len(testCases); i += 2 {
		var b strings.Builder
		escapePathPart(&b, testCases[i])
		testarossa.Equal(t, testCases[i+1], b.String())
	}
}

func TestConnector_SubjectOfSubscription(t *testing.T) {
	testarossa.Equal(t, "p0.80.com.example.|.GET.PATH.to.file_html", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "80", "PATH/to/file.html"))
	testarossa.Equal(t, "p0.80.com.example.|.GET.PATH._", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "80", "PATH/"))
	testarossa.Equal(t, "p0.123.com.example.|.POST.DIR.>", subjectOfSubscription("p0", "POST", "example.com", "123", "DIR/{+}"))
	testarossa.Equal(t, "p0.123.com.example.|.PATCH.DIR.>", subjectOfSubscription("p0", "PATCH", "example.com", "123", "/DIR/{file+}"))
	testarossa.Equal(t, "p0.443.com.example.www.|.DELETE.>", subjectOfSubscription("p0", "delete", "www.example.com", "443", "/{+}"))
	testarossa.Equal(t, "p0.443.com.example.www.|.*._", subjectOfSubscription("p0", "ANY", "www.example.com", "443", ""))
	testarossa.Equal(t, "p0.*.com.example.|.GET.PATH.to.file_html", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "0", "PATH/to/file.html"))
	testarossa.Equal(t, "p0.443.com.example.|.GET.foo.*.bar.*", subjectOfSubscription("p0", "GET", "example.com", "443", "/foo/{foo}/bar/{bar}"))
	testarossa.Equal(t, "p0.*.com.example.|.*.foo.*.bar.*.>", subjectOfSubscription("p0", "ANY", "example.com", "0", "/foo/{foo}/bar/{bar}/{appendix+}"))
	testarossa.Equal(t, "p0.80.com.example.|.GET.empty._._._", subjectOfSubscription("p0", "GET", "EXAMPLE.com", "80", "empty///"))
}

func TestConnector_SubjectOfRequest(t *testing.T) {
	testarossa.Equal(t, "p0.80.com.example.|.GET.PATH.to.file_html", subjectOfRequest("p0", "GET", "EXAMPLE.com", "80", "PATH/to/file.html"))
	testarossa.Equal(t, "p0.80.com.example.|.GET.PATH._", subjectOfRequest("p0", "GET", "EXAMPLE.com", "80", "PATH/"))
	testarossa.Equal(t, "p0.123.com.example.|.PATCH.DIR._", subjectOfRequest("p0", "PATCH", "example.com", "123", "/DIR/"))
	testarossa.Equal(t, "p0.443.com.example.www.|.ANY.method", subjectOfRequest("p0", "ANY", "www.example.com", "443", "/method"))
	testarossa.Equal(t, "p0.443.com.example.www.|.DELETE._", subjectOfRequest("p0", "delete", "www.example.com", "443", "/"))
	testarossa.Equal(t, "p0.443.com.example.www.|.OPTIONS._", subjectOfRequest("p0", "OPTIONS", "www.example.com", "443", ""))
	testarossa.Equal(t, "p0.0.com.example.|.GET.PATH.to.file_html", subjectOfRequest("p0", "GET", "EXAMPLE.com", "0", "PATH/to/file.html"))
	testarossa.Equal(t, "p0.443.com.example.|.GET.foo.%007bfoo%007d.bar.%007bbar%007d", subjectOfRequest("p0", "GET", "example.com", "443", "/foo/{foo}/bar/{bar}"))
	testarossa.Equal(t, "p0.80.com.example.|.GET.empty._._._", subjectOfRequest("p0", "GET", "EXAMPLE.com", "80", "empty///"))
}

func TestConnector_subjectOfResponses(t *testing.T) {
	testarossa.Equal(t, "p0.r.com.example.1234", subjectOfResponses("p0", "example.com", "1234"))
	testarossa.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.example.com", "abcd1234"))
	testarossa.Equal(t, "p0.r.com.example.www.abcd1234", subjectOfResponses("p0", "www.EXAMPLE.com", "ABCD1234"))
}

func TestConnector_ReverseHostname(t *testing.T) {
	testarossa.Equal(t, "com.example.sub.www", reverseHostname("www.sub.example.com"))
	testarossa.Equal(t, "com.example.www", reverseHostname("www.example.com"))
	testarossa.Equal(t, "com.example", reverseHostname("example.com"))
	testarossa.Equal(t, "com", reverseHostname("com"))
	testarossa.Equal(t, "", reverseHostname(""))
}
