/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"html"
	"io"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/pub"
	"github.com/stretchr/testify/assert"
)

func TestConnector_ReadResFile(t *testing.T) {
	t.Parallel()

	// Create the microservices
	con := New("read.res.file.connector")
	con.SetResDirFS("testdata")

	assert.Equal(t, "<html>{{ . }}</html>\n", string(con.MustReadResFile("res.txt")))
	assert.Equal(t, "<html>{{ . }}</html>\n", con.MustReadResTextFile("res.txt"))

	assert.Nil(t, con.MustReadResFile("nothing.txt"))
	assert.Equal(t, "", con.MustReadResTextFile("nothing.txt"))

	v, err := con.ExecuteResTemplate("res.txt", "<body></body>")
	assert.NoError(t, err)
	assert.Equal(t, "<html><body></body></html>\n", v)

	v, err = con.ExecuteResTemplate("res.html", "<body></body>")
	assert.NoError(t, err)
	assert.Equal(t, "<html>"+html.EscapeString("<body></body>")+"</html>\n", v)
}

func TestConnector_LoadResString(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.load.res.string.connector")

	beta := New("beta.load.res.string.connector")
	beta.Subscribe("localized", func(w http.ResponseWriter, r *http.Request) error {
		s, _ := beta.LoadResString(r.Context(), "hello")
		w.Write([]byte(s))
		return nil
	})
	beta.SetResDirFS("testdata")

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
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
		if assert.NoError(t, err) {
			body, err := io.ReadAll(response.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte(testCases[i+1]), body)
			}
		}
	}
}
