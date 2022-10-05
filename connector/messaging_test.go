package connector

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/stretchr/testify/assert"
)

func TestConnector_Echo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.echo.connector")

	beta := NewConnector()
	beta.SetHostName("beta.echo.connector")
	beta.Subscribe("echo", func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		_, err = w.Write(body)
		assert.NoError(t, err)
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

	// Send message and validate that it's echoed back
	response, err := alpha.POST(ctx, "https://beta.echo.connector/echo", []byte("Hello"))
	assert.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello"), body)
}

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
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

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

func TestConnector_QueryArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.queryargs.connector")

	beta := NewConnector()
	beta.SetHostName("beta.queryargs.connector")
	beta.Subscribe("arg", func(w http.ResponseWriter, r *http.Request) error {
		arg := r.URL.Query().Get("arg")
		assert.Equal(t, "not_empty", arg)
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

	// Send request with a query argument
	_, err = alpha.GET(ctx, "https://beta.queryargs.connector/arg?arg=not_empty")
	assert.NoError(t, err)
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
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

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

func TestConnector_LoadBalancing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.loadbalancing.connector")

	count1 := int32(0)
	count2 := int32(0)

	beta1 := NewConnector()
	beta1.SetHostName("beta.loadbalancing.connector")
	beta1.Subscribe("lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	beta2 := NewConnector()
	beta2.SetHostName("beta.loadbalancing.connector")
	beta2.Subscribe("lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count2, 1)
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)
	err = beta1.Startup()
	assert.NoError(t, err)
	defer beta1.Shutdown(ctx)
	err = beta2.Startup()
	assert.NoError(t, err)
	defer beta2.Shutdown(ctx)

	// Send messages
	var wg sync.WaitGroup
	for i := 0; i < 1024; i++ {
		wg.Add(1)
		go func() {
			_, err := alpha.GET(ctx, "https://beta.loadbalancing.connector/lb")
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

func TestConnector_Concurrent(t *testing.T) {
	// No parallel

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.concurrent.connector")

	beta := NewConnector()
	beta.SetHostName("beta.concurrent.connector")
	beta.Subscribe("wait", func(w http.ResponseWriter, r *http.Request) error {
		ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
		time.Sleep(time.Millisecond * time.Duration(ms))
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

	// Send messages
	var wg sync.WaitGroup
	for i := 50; i <= 500; i += 50 {
		i := i
		wg.Add(1)
		go func() {
			start := time.Now()
			_, err := alpha.GET(ctx, "https://beta.concurrent.connector/wait?ms="+strconv.Itoa(i))
			end := time.Now()
			assert.NoError(t, err)
			assert.WithinDuration(t, start.Add(time.Millisecond*time.Duration(i)), end, time.Millisecond*49)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestConnector_CallDepth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	depth := 0

	// Create the microservice
	alpha := NewConnector()
	alpha.maxCallDepth = 8
	alpha.SetHostName("alpha.calldepth.connector")
	alpha.Subscribe("next", func(w http.ResponseWriter, r *http.Request) error {
		depth++

		step, _ := strconv.Atoi(r.URL.Query().Get("step"))
		assert.Equal(t, depth, step)
		assert.Equal(t, depth, frame.Of(r).CallDepth())

		lastCall := depth == alpha.maxCallDepth
		_, err := alpha.GET(r.Context(), "https://alpha.calldepth.connector/next?step="+strconv.Itoa(step+1))
		if lastCall {
			assert.Error(t, err)
		}
		return errors.Trace(err)
	})

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)

	_, err = alpha.GET(ctx, "https://alpha.calldepth.connector/next?step=1")
	assert.Error(t, err)
	assert.Equal(t, alpha.maxCallDepth, depth)
}

func TestConnector_TimeoutDrawdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	budget := 8 * time.Second
	depth := 0

	// Create the microservice
	alpha := NewConnector()
	alpha.networkHop = time.Second
	alpha.SetHostName("alpha.timeoutdrawdown.connector")
	alpha.Subscribe("next", func(w http.ResponseWriter, r *http.Request) error {
		depth++
		_, err := alpha.GET(r.Context(), "https://alpha.timeoutdrawdown.connector/next")
		return errors.Trace(err)
	})

	// Startup the microservice
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)

	_, err = alpha.Publish(
		ctx,
		pub.GET("https://alpha.timeoutdrawdown.connector/next"),
		pub.TimeBudget(budget),
	)
	assert.Error(t, err)
	assert.True(t, depth > 1 && depth <= int(budget/alpha.networkHop))
}

func TestConnector_TimeoutNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	alpha := NewConnector()
	alpha.SetHostName("alpha.timeoutnotfound.connector")

	// Startup the microservice
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)

	// Set a time budget
	t0 := time.Now()
	_, err = alpha.Publish(
		ctx,
		pub.GET("https://alpha.timeoutnotfound.connector/nowhere"),
		pub.TimeBudget(500*time.Millisecond),
	)
	assert.Error(t, err)
	assert.True(t, time.Since(t0) >= 500*time.Millisecond && time.Since(t0) < 600*time.Millisecond)

	// Use the default time budget
	alpha.defaultTimeBudget = 500 * time.Millisecond
	t0 = time.Now()
	_, err = alpha.Publish(
		ctx,
		pub.GET("https://alpha.timeoutnotfound.connector/nowhere"),
	)
	assert.Error(t, err)
	assert.True(t, time.Since(t0) >= 500*time.Millisecond && time.Since(t0) < 600*time.Millisecond)
}

func TestConnector_TimeoutSlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	alpha := NewConnector()
	alpha.SetHostName("alpha.timeoutslow.connector")
	alpha.Subscribe("slow", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(time.Second)
		return nil
	})

	// Startup the microservice
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown(ctx)

	t0 := time.Now()
	_, err = alpha.Publish(
		ctx,
		pub.GET("https://alpha.timeoutslow.connector/slow"),
		pub.TimeBudget(time.Millisecond*500),
	)
	assert.Error(t, err)
	assert.True(t, time.Since(t0) >= 500*time.Millisecond && time.Since(t0) < 600*time.Millisecond)
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
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

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
	defer alpha.Shutdown(ctx)
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown(ctx)

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
