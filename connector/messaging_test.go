package connector

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestLoadBalancing(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.loadbalancing.connector")

	count1 := int32(0)
	count2 := int32(0)

	beta1 := NewConnector()
	beta1.SetHostName("beta.loadbalancing.connector")
	beta1.Subscribe(443, "lb", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count1, 1)
	})

	beta2 := NewConnector()
	beta2.SetHostName("beta.loadbalancing.connector")
	beta2.Subscribe(443, "lb", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count2, 1)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta1.Startup()
	assert.NoError(t, err)
	defer beta1.Shutdown()
	err = beta2.Startup()
	assert.NoError(t, err)
	defer beta2.Shutdown()

	// Send messages
	var wg sync.WaitGroup
	for i := 0; i < 1024; i++ {
		wg.Add(1)
		go func() {
			_, err := alpha.GET("https://beta.loadbalancing.connector/lb")
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()

	// The requests should be more or less evenly distributed among the server microservices
	assert.Equal(t, int32(1024), count1+count2)
	assert.True(t, count1 > 256)
	assert.True(t, count2 > 256)
}

func TestConcurrent(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.concurrent.connector")

	beta := NewConnector()
	beta.SetHostName("beta.concurrent.connector")
	beta.Subscribe(443, "wait", func(w http.ResponseWriter, r *http.Request) {
		ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
		time.Sleep(time.Millisecond * time.Duration(ms))
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send messages
	var wg sync.WaitGroup
	for i := 50; i <= 500; i += 50 {
		i := i
		wg.Add(1)
		go func() {
			start := time.Now()
			_, err := alpha.GET("https://beta.concurrent.connector/wait?ms=" + strconv.Itoa(i))
			end := time.Now()
			assert.NoError(t, err)
			assert.WithinDuration(t, start.Add(time.Millisecond*time.Duration(i)), end, time.Millisecond*49)
			wg.Done()
		}()
	}
	wg.Wait()
}