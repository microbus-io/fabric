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
	"context"
	"html"
	"io"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/testarossa"
)

func TestConnector_ReadResFile(t *testing.T) {
	t.Parallel()

	// Create the microservices
	con := New("read.res.file.connector")
	con.SetResFSDir("testdata")

	testarossa.Equal(t, "<html>{{ . }}</html>\n", string(con.MustReadResFile("res.txt")))
	testarossa.Equal(t, "<html>{{ . }}</html>\n", con.MustReadResTextFile("res.txt"))

	testarossa.Nil(t, con.MustReadResFile("nothing.txt"))
	testarossa.Equal(t, "", con.MustReadResTextFile("nothing.txt"))

	v, err := con.ExecuteResTemplate("res.txt", "<body></body>")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "<html><body></body></html>\n", v)

	v, err = con.ExecuteResTemplate("res.html", "<body></body>")
	testarossa.NoError(t, err)
	testarossa.Equal(t, "<html>"+html.EscapeString("<body></body>")+"</html>\n", v)
}

func TestConnector_LoadResString(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.load.res.string.connector")

	beta := New("beta.load.res.string.connector")
	beta.Subscribe("GET", "localized", func(w http.ResponseWriter, r *http.Request) error {
		s, _ := beta.LoadResString(r.Context(), "hello")
		w.Write([]byte(s))
		return nil
	})
	beta.SetResFSDir("testdata")

	// Startup the microservices
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()

	// Send message and validate the correct language
	testCases := []string{
		"", "Hello",
		"en", "Hello",
		"en-CA", "Hello",
		"en-AU", "G'day",
		"fr", "Hello",
		"it", "Ciao",
	}
	for i := 0; i < len(testCases); i += 2 {
		response, err := alpha.Request(ctx, pub.GET("https://beta.load.res.string.connector/localized"), pub.Header("Accept-Language", testCases[i]))
		if testarossa.NoError(t, err) {
			body, err := io.ReadAll(response.Body)
			if testarossa.NoError(t, err) {
				testarossa.SliceEqual(t, []byte(testCases[i+1]), body)
			}
		}
	}
}
