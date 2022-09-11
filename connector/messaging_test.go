package connector

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEcho(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.echo.test")

	beta := NewConnector()
	beta.SetHostName("beta.echo.test")
	beta.Subscribe(443, "echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		_, err = w.Write(body)
		assert.NoError(t, err)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	err = beta.Startup()
	assert.NoError(t, err)

	// Send message and validate that it's echoed back
	response, err := alpha.POST("https://beta.echo.test/echo", []byte("Hello"))
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello"), body)

	// Shutdown the microservices
	err = alpha.Shutdown()
	assert.NoError(t, err)
	err = beta.Shutdown()
	assert.NoError(t, err)
}

func TestDirectorySubscription(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.dir.test")

	count := 0
	beta := NewConnector()
	beta.SetHostName("beta.dir.test")
	beta.Subscribe(443, "directory/", func(w http.ResponseWriter, r *http.Request) {
		count++
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	err = beta.Startup()
	assert.NoError(t, err)

	// Send messages to various locations under the directory
	_, err = alpha.GET("https://beta.dir.test/directory/")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.test/directory/1.html")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.test/directory/2.html")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.test/directory/sub/3.html")
	assert.NoError(t, err)

	assert.Equal(t, 4, count)

	// Shutdown the microservices
	err = alpha.Shutdown()
	assert.NoError(t, err)
	err = beta.Shutdown()
	assert.NoError(t, err)
}
