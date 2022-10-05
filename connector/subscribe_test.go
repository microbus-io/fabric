package connector

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/microbus-io/fabric/errors"
	"github.com/stretchr/testify/assert"
)

func TestConnector_DirectorySubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.dir.connector")

	var count int32
	beta := NewConnector()
	beta.SetHostName("beta.dir.connector")
	beta.Subscribe("directory/", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send messages to various locations under the directory
	_, err = alpha.GET(ctx, "https://beta.dir.connector/directory/")
	assert.NoError(t, err)
	_, err = alpha.GET(ctx, "https://beta.dir.connector/directory/1.html")
	assert.NoError(t, err)
	_, err = alpha.GET(ctx, "https://beta.dir.connector/directory/2.html")
	assert.NoError(t, err)
	_, err = alpha.GET(ctx, "https://beta.dir.connector/directory/sub/3.html")
	assert.NoError(t, err)

	assert.Equal(t, int32(4), count)
}

func TestConnector_ErrorAndPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.error.connector")

	beta := NewConnector()
	beta.SetHostName("beta.error.connector")
	beta.Subscribe("err", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("it's bad")
	})
	beta.Subscribe("panic", func(w http.ResponseWriter, r *http.Request) error {
		panic("it's really bad")
	})
	beta.Subscribe("oserr", func(w http.ResponseWriter, r *http.Request) error {
		err := errors.Trace(os.ErrNotExist)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		return err
	})
	beta.Subscribe("stillalive", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send messages
	_, err = alpha.GET(ctx, "https://beta.error.connector/err")
	assert.Error(t, err)
	assert.Equal(t, "it's bad", err.Error())

	_, err = alpha.GET(ctx, "https://beta.error.connector/panic")
	assert.Error(t, err)
	assert.Equal(t, "it's really bad", err.Error())

	_, err = alpha.GET(ctx, "https://beta.error.connector/oserr")
	assert.Error(t, err)
	assert.Equal(t, "file does not exist", err.Error())
	assert.False(t, errors.Is(err, os.ErrNotExist)) // Cannot reconstitute error type

	_, err = alpha.GET(ctx, "https://beta.error.connector/stillalive")
	assert.NoError(t, err)
}

func TestConnector_DifferentPlanes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("diffplanes.connector")
	alpha.SetPlane("alpha")
	alpha.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})

	beta := NewConnector()
	beta.SetHostName("diffplanes.connector")
	beta.SetPlane("beta")
	beta.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("beta"))
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://diffplanes.connector/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("alpha"), body)
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://diffplanes.connector/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("beta"), body)
	}
}

func TestConnector_SubscribeBeforeAndAfterStartup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.beforeafterstartup.connector")

	var beforeCalled, afterCalled bool
	beta := NewConnector()
	beta.SetHostName("beta.beforeafterstartup.connector")

	// Subscribe before beta is started
	beta.Subscribe("before", func(w http.ResponseWriter, r *http.Request) error {
		beforeCalled = true
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Subscribe after beta is started
	beta.Subscribe("after", func(w http.ResponseWriter, r *http.Request) error {
		afterCalled = true
		return nil
	})

	// Send requests to both handlers
	_, err = alpha.GET(ctx, "https://beta.beforeafterstartup.connector/before")
	assert.NoError(t, err)
	_, err = alpha.GET(ctx, "https://beta.beforeafterstartup.connector/after")
	assert.NoError(t, err)

	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}
