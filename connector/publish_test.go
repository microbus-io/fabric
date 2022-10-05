package connector

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
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
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()

	// Send message and validate that it's echoed back
	response, err := alpha.POST(ctx, "https://beta.echo.connector/echo", []byte("Hello"))
	assert.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello"), body)
}

func TestConnector_QueryArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	con := NewConnector()
	con.SetHostName("queryargs.connector")
	con.Subscribe("arg", func(w http.ResponseWriter, r *http.Request) error {
		arg := r.URL.Query().Get("arg")
		assert.Equal(t, "not_empty", arg)
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send request with a query argument
	_, err = con.GET(ctx, "https://queryargs.connector/arg?arg=not_empty")
	assert.NoError(t, err)
}

func TestConnector_LoadBalancing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := NewConnector()
	alpha.SetHostName("alpha.load.balancing.connector")

	count1 := int32(0)
	count2 := int32(0)

	beta1 := NewConnector()
	beta1.SetHostName("beta.load.balancing.connector")
	beta1.Subscribe("lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	beta2 := NewConnector()
	beta2.SetHostName("beta.load.balancing.connector")
	beta2.Subscribe("lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count2, 1)
		return nil
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
			_, err := alpha.GET(ctx, "https://beta.load.balancing.connector/lb")
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
	con := NewConnector()
	con.maxCallDepth = 8
	con.SetHostName("call.depth.connector")
	con.Subscribe("next", func(w http.ResponseWriter, r *http.Request) error {
		depth++

		step, _ := strconv.Atoi(r.URL.Query().Get("step"))
		assert.Equal(t, depth, step)
		assert.Equal(t, depth, frame.Of(r).CallDepth())

		lastCall := depth == con.maxCallDepth
		_, err := con.GET(r.Context(), "https://call.depth.connector/next?step="+strconv.Itoa(step+1))
		if lastCall {
			assert.Error(t, err)
		}
		return errors.Trace(err)
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://call.depth.connector/next?step=1")
	assert.Error(t, err)
	assert.Equal(t, con.maxCallDepth, depth)
}

func TestConnector_TimeoutDrawdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	budget := 8 * time.Second
	depth := 0

	// Create the microservice
	con := NewConnector()
	con.networkHop = time.Second
	con.SetHostName("timeout.drawdown.connector")
	con.Subscribe("next", func(w http.ResponseWriter, r *http.Request) error {
		depth++
		_, err := con.GET(r.Context(), "https://timeout.drawdown.connector/next")
		return errors.Trace(err)
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.drawdown.connector/next"),
		pub.TimeBudget(budget),
	)
	assert.Error(t, err)
	assert.True(t, depth > 1 && depth <= int(budget/con.networkHop))
}

func TestConnector_TimeoutNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := NewConnector()
	con.SetHostName("timeout.not.found.connector")

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Set a time budget in the request
	t0 := time.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
		pub.TimeBudget(2*time.Second),
	)
	dur := time.Since(t0)
	assert.Error(t, err)
	assert.True(t, dur >= con.networkHop && dur <= con.networkHop*2)

	// Use the default time budget
	t0 = time.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
	)
	dur = time.Since(t0)
	assert.Error(t, err)
	assert.True(t, dur >= con.networkHop && dur <= con.networkHop*2)
}

func TestConnector_TimeoutSlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := NewConnector()
	con.SetHostName("timeout.slow.connector")
	con.Subscribe("slow", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(time.Second)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	t0 := time.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.slow.connector/slow"),
		pub.TimeBudget(time.Millisecond*500),
	)
	assert.Error(t, err)
	assert.True(t, time.Since(t0) >= 500*time.Millisecond && time.Since(t0) < 600*time.Millisecond)
}

func TestConnector_Multicast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	noqueue1 := NewConnector()
	noqueue1.SetHostName("multicast.connector")
	noqueue1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue1"))
		return nil
	}, sub.NoQueue())

	noqueue2 := NewConnector()
	noqueue2.SetHostName("multicast.connector")
	noqueue2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue2"))
		return nil
	}, sub.NoQueue())

	named1 := NewConnector()
	named1.SetHostName("multicast.connector")
	named1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named1"))
		return nil
	}, sub.Queue("MyQueue"))

	named2 := NewConnector()
	named2.SetHostName("multicast.connector")
	named2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named2"))
		return nil
	}, sub.Queue("MyQueue"))

	def1 := NewConnector()
	def1.SetHostName("multicast.connector")
	def1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def1"))
		return nil
	}, sub.DefaultQueue())

	def2 := NewConnector()
	def2.SetHostName("multicast.connector")
	def2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def2"))
		return nil
	}, sub.DefaultQueue())

	// Startup the microservice
	for _, i := range []*Connector{noqueue1, noqueue2, named1, named2, def1, def2} {
		err := i.Startup()
		assert.NoError(t, err)
		defer i.Shutdown()
	}

	// Make the first request
	client := named1
	t0 := time.Now()
	responded := map[string]bool{}
	ch := client.Publish(ctx, pub.GET("https://multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		res, err := i.Get()
		if assert.NoError(t, err) {
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			responded[string(body)] = true
		}
	}
	dur := time.Since(t0)
	assert.True(t, dur >= client.networkHop && dur < client.networkHop*2)
	assert.Len(t, responded, 4)
	assert.True(t, responded["noqueue1"])
	assert.True(t, responded["noqueue2"])
	assert.True(t, responded["named1"] || responded["named2"])
	assert.False(t, responded["named1"] && responded["named2"])
	assert.True(t, responded["def1"] || responded["def2"])
	assert.False(t, responded["def1"] && responded["def2"])

	// Make the second request, should be quicker due to known responders optimization
	t0 = time.Now()
	responded = map[string]bool{}
	ch = client.Publish(ctx, pub.GET("https://multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		res, err := i.Get()
		if assert.NoError(t, err) {
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			responded[string(body)] = true
		}
	}
	dur = time.Since(t0)
	assert.True(t, dur < client.networkHop)
	assert.Len(t, responded, 4)
	assert.True(t, responded["noqueue1"])
	assert.True(t, responded["noqueue2"])
	assert.True(t, responded["named1"] || responded["named2"])
	assert.False(t, responded["named1"] && responded["named2"])
	assert.True(t, responded["def1"] || responded["def2"])
	assert.False(t, responded["def1"] && responded["def2"])
}
