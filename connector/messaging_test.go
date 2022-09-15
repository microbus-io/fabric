package connector

import (
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEcho(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.echo.connector")

	beta := NewConnector()
	beta.SetHostName("beta.echo.connector")
	beta.Subscribe(443, "echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		_, err = w.Write(body)
		assert.NoError(t, err)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send message and validate that it's echoed back
	response, err := alpha.POST("https://beta.echo.connector/echo", []byte("Hello"))
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello"), body)
}

func TestDirectorySubscription(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.dir.connector")

	var count int32
	beta := NewConnector()
	beta.SetHostName("beta.dir.connector")
	beta.Subscribe(443, "directory/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send messages to various locations under the directory
	_, err = alpha.GET("https://beta.dir.connector/directory/")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.connector/directory/1.html")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.connector/directory/2.html")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.dir.connector/directory/sub/3.html")
	assert.NoError(t, err)

	assert.Equal(t, int32(4), count)
}

func TestQueryArgs(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.queryargs.connector")

	beta := NewConnector()
	beta.SetHostName("beta.queryargs.connector")
	beta.Subscribe(443, "arg", func(w http.ResponseWriter, r *http.Request) {
		arg := r.URL.Query().Get("arg")
		assert.Equal(t, "not_empty", arg)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send request with a query argument
	_, err = alpha.GET("https://beta.queryargs.connector/arg?arg=not_empty")
	assert.NoError(t, err)
}

func TestSubscribeBeforeAndAfterStartup(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.beforeafterstartup.connector")

	var beforeCalled, afterCalled bool
	beta := NewConnector()
	beta.SetHostName("beta.beforeafterstartup.connector")

	// Subscribe before beta is started
	beta.Subscribe(443, "before", func(w http.ResponseWriter, r *http.Request) {
		beforeCalled = true
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Subscribe after beta is started
	beta.Subscribe(443, "after", func(w http.ResponseWriter, r *http.Request) {
		afterCalled = true
	})

	// Send requests to both handlers
	_, err = alpha.GET("https://beta.beforeafterstartup.connector/before")
	assert.NoError(t, err)
	_, err = alpha.GET("https://beta.beforeafterstartup.connector/after")
	assert.NoError(t, err)

	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}
