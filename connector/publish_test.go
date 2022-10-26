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
	alpha := New("alpha.echo.connector")

	beta := New("beta.echo.connector")
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

func BenchmarkConnector_EchoSerial(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	con := New("echoserial.connector")
	con.Subscribe("echo", func(w http.ResponseWriter, r *http.Request) error {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
		return nil
	})

	// Startup the microservice
	con.Startup()
	defer con.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		con.POST(ctx, "https://echoserial.connector/echo", []byte("Hello"))
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 15":
	// 78851 ns/op
	// 32869 B/op
	// 210 allocs/op
}

func BenchmarkConnector_EchoParallel(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	con := New("echo.parallel.connector")
	con.Subscribe("echo", func(w http.ResponseWriter, r *http.Request) error {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
		return nil
	})

	// Startup the microservice
	con.Startup()
	defer con.Shutdown()

	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			con.POST(ctx, "https://echo.parallel.connector/echo", []byte("Hello"))
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()

	// On 2021 MacBook Pro M1 15":
	// N=46232 concurrent
	// 20094 ns/op
	// 59724 B/op
	// 242 allocs/op
}

func TestConnector_QueryArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	con := New("query.args.connector")
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
	_, err = con.GET(ctx, "https://query.args.connector/arg?arg=not_empty")
	assert.NoError(t, err)
}

func TestConnector_LoadBalancing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.load.balancing.connector")

	count1 := int32(0)
	count2 := int32(0)

	beta1 := New("beta.load.balancing.connector")
	beta1.Subscribe("lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	beta2 := New("beta.load.balancing.connector")
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
	for i := 0; i < 256; i++ {
		wg.Add(1)
		go func() {
			_, err := alpha.GET(ctx, "https://beta.load.balancing.connector/lb")
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()

	// The requests should be more or less evenly distributed among the server microservices
	assert.Equal(t, int32(256), count1+count2)
	assert.True(t, count1 > 64)
	assert.True(t, count2 > 64)
}

func TestConnector_Concurrent(t *testing.T) {
	// No parallel

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.concurrent.connector")

	beta := New("beta.concurrent.connector")
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
			start := alpha.Now()
			_, err := alpha.GET(ctx, "https://beta.concurrent.connector/wait?ms="+strconv.Itoa(i))
			end := alpha.Now()
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
	con := New("call.depth.connector")
	con.maxCallDepth = 8
	con.Subscribe("next", func(w http.ResponseWriter, r *http.Request) error {
		depth++

		step, _ := strconv.Atoi(r.URL.Query().Get("step"))
		assert.Equal(t, depth, step)
		assert.Equal(t, depth, frame.Of(r).CallDepth())

		_, err := con.GET(r.Context(), "https://call.depth.connector/next?step="+strconv.Itoa(step+1))
		assert.Error(t, err)
		assert.Contains(t, "call depth overflow", err.Error())
		return errors.Trace(err)
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://call.depth.connector/next?step=1")
	assert.Error(t, err)
	assert.Contains(t, "call depth overflow", err.Error())
	assert.Equal(t, con.maxCallDepth, depth)
}

func TestConnector_TimeoutDrawdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	depth := 0

	// Create the microservice
	con := New("timeout.drawdown.connector")
	budget := con.networkHop * 8
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
	assert.Contains(t, "ack timeout", err.Error())
	assert.True(t, depth >= 7 && depth <= 8, "%d", depth)
}

func TestConnector_TimeoutNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("timeout.not.found.connector")

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Set a time budget in the request
	t0 := con.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
		pub.TimeBudget(2*time.Second),
	)
	dur := con.Clock().Since(t0)
	assert.Error(t, err)
	assert.True(t, dur >= AckTimeout && dur < 2*AckTimeout)

	// Use the default time budget
	t0 = con.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
	)
	dur = con.Clock().Since(t0)
	assert.Error(t, err)
	assert.Contains(t, "ack timeout", err.Error())
	assert.True(t, dur >= AckTimeout && dur < 2*AckTimeout)
}

func TestConnector_TimeoutSlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("timeout.slow.connector")
	con.Subscribe("slow", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(time.Second)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	t0 := con.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.slow.connector/slow"),
		pub.TimeBudget(time.Millisecond*500),
	)
	assert.Error(t, err)
	dur := con.Clock().Since(t0)
	assert.True(t, dur >= 500*time.Millisecond && dur < 600*time.Millisecond)
}

func TestConnector_ContextTimeout(t *testing.T) {
	t.Parallel()

	con := New("context.timeout.connector")

	done := false
	con.Subscribe("timeout", func(w http.ResponseWriter, r *http.Request) error {
		<-r.Context().Done()
		done = true
		return r.Context().Err()
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.Request(
		con.Lifetime(),
		pub.GET("https://context.timeout.connector/timeout"),
		pub.TimeBudget(time.Second),
	)
	assert.Error(t, err)
	assert.True(t, done)
}

func TestConnector_Multicast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	noqueue1 := New("multicast.connector")
	noqueue1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue1"))
		return nil
	}, sub.NoQueue())

	noqueue2 := New("multicast.connector")
	noqueue2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue2"))
		return nil
	}, sub.NoQueue())

	named1 := New("multicast.connector")
	named1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named1"))
		return nil
	}, sub.Queue("MyQueue"))

	named2 := New("multicast.connector")
	named2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named2"))
		return nil
	}, sub.Queue("MyQueue"))

	def1 := New("multicast.connector")
	def1.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def1"))
		return nil
	}, sub.DefaultQueue())

	def2 := New("multicast.connector")
	def2.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def2"))
		return nil
	}, sub.DefaultQueue())

	// Startup the microservices
	for _, i := range []*Connector{noqueue1, noqueue2, named1, named2, def1, def2} {
		err := i.Startup()
		assert.NoError(t, err)
		defer i.Shutdown()
	}

	// Make the first request
	client := named1
	t0 := client.Now()
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
	dur := client.Clock().Since(t0)
	assert.True(t, dur >= AckTimeout && dur < AckTimeout*2)
	assert.Len(t, responded, 4)
	assert.True(t, responded["noqueue1"])
	assert.True(t, responded["noqueue2"])
	assert.True(t, responded["named1"] || responded["named2"])
	assert.False(t, responded["named1"] && responded["named2"])
	assert.True(t, responded["def1"] || responded["def2"])
	assert.False(t, responded["def1"] && responded["def2"])

	// Make the second request, should be quicker due to known responders optimization
	t0 = client.Now()
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
	dur = client.Clock().Since(t0)
	assert.True(t, dur < AckTimeout)
	assert.Len(t, responded, 4)
	assert.True(t, responded["noqueue1"])
	assert.True(t, responded["noqueue2"])
	assert.True(t, responded["named1"] || responded["named2"])
	assert.False(t, responded["named1"] && responded["named2"])
	assert.True(t, responded["def1"] || responded["def2"])
	assert.False(t, responded["def1"] && responded["def2"])
}

func TestConnector_MulticastDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	slow := New("multicast.delay.connector")
	delay := AckTimeout
	slow.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(delay * 2)
		w.Write([]byte("slow"))
		return nil
	}, sub.NoQueue())

	fast := New("multicast.delay.connector")
	fast.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("fast"))
		return nil
	}, sub.NoQueue())

	tooSlow := New("multicast.delay.connector")
	tooSlow.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(delay * 4)
		w.Write([]byte("too slow"))
		return nil
	}, sub.NoQueue())

	// Startup the microservice
	err := slow.Startup()
	assert.NoError(t, err)
	defer slow.Shutdown()
	err = fast.Startup()
	assert.NoError(t, err)
	defer fast.Shutdown()
	err = tooSlow.Startup()
	assert.NoError(t, err)
	defer tooSlow.Shutdown()

	// Send the message
	var respondedOK, respondedErr int
	t0 := slow.Now()
	ch := slow.Publish(
		ctx,
		pub.GET("https://multicast.delay.connector/cast"),
		pub.Multicast(),
		pub.TimeBudget(delay*3),
	)
	for i := range ch {
		res, err := i.Get()
		if err == nil {
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			dur := slow.Clock().Since(t0)
			if string(body) == "fast" {
				assert.True(t, dur < delay)
			} else if string(body) == "slow" {
				assert.True(t, dur >= 2*delay && dur < 3*delay)
			}
			respondedOK++
		} else {
			assert.Contains(t, err.Error(), "timeout")
			respondedErr++
			assert.Equal(t, 2, respondedOK)
			dur := slow.Clock().Since(t0)
			assert.True(t, dur >= 3*delay && dur < 4*delay)
		}
	}
	assert.Equal(t, 2, respondedOK)
	assert.Equal(t, 1, respondedErr)
}

func TestConnector_MulticastError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	bad := New("multicast.error.connector")
	bad.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("bad situation")
	}, sub.NoQueue())

	good := New("multicast.error.connector")
	good.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("good situation"))
		return nil
	}, sub.NoQueue())

	// Startup the microservice
	err := bad.Startup()
	assert.NoError(t, err)
	defer bad.Shutdown()
	err = good.Startup()
	assert.NoError(t, err)
	defer good.Shutdown()

	// Send the message
	var countErrs, countOKs int
	t0 := bad.Now()
	ch := bad.Publish(ctx, pub.GET("https://multicast.error.connector/cast"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		if err != nil {
			countErrs++
		} else {
			countOKs++
		}
	}
	dur := bad.Clock().Since(t0)
	assert.True(t, dur >= AckTimeout && dur < 2*AckTimeout)
	assert.Equal(t, 1, countErrs)
	assert.Equal(t, 1, countOKs)
}

func TestConnector_MassMulticast(t *testing.T) {
	// No parallel

	ctx := context.Background()
	N := 128

	// Create the client microservice
	client := New("client.mass.multicast.connector")

	err := client.Startup()
	assert.NoError(t, err)
	defer client.Shutdown()

	// Create the server microservices in parallel
	var wg sync.WaitGroup
	cons := make([]*Connector, N)
	for i := 0; i < N; i++ {
		i := i
		wg.Add(1)
		go func() {
			cons[i] = New("mass.multicast.connector")
			cons[i].Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
				w.Write([]byte("ok"))
				return nil
			}, sub.NoQueue())

			err := cons[i].Startup()
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()
	defer func() {
		var wg sync.WaitGroup
		for i := 0; i < N; i++ {
			i := i
			wg.Add(1)
			go func() {
				err := cons[i].Shutdown()
				assert.NoError(t, err)
				wg.Done()
			}()
		}
		wg.Wait()
	}()

	// Send the message
	var countOKs int
	t0 := client.Now()
	ch := client.Publish(ctx, pub.GET("https://mass.multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		if assert.NoError(t, err) {
			countOKs++
		}
	}
	dur := client.Clock().Since(t0)
	assert.True(t, dur >= AckTimeout && dur < 2*AckTimeout)
	assert.Equal(t, N, countOKs)
}

func BenchmarkConnector_NATSDirectPublishing(b *testing.B) {
	con := New("nats.direct.connector")

	err := con.Startup()
	assert.NoError(b, err)
	defer con.Shutdown()

	body := make([]byte, 512*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		con.natsConn.Publish("somewhere", body)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 15":
	// 128B: 82 ns/op
	// 256B: 104 ns/op
	// 512B: 153 ns/op
	// 1KB: 247 ns/op
	// 2KB: 410 ns/op
	// 4KB: 746 ns/op
	// 8KB: 1480 ns/op
	// 16KB: 2666 ns/op
	// 32KB: 5474 ns/op
	// 64KB: 9173 ns/op
	// 128KB: 16307 ns/op
	// 256KB: 32700 ns/op
	// 512KB: 63429 ns/op
}

func TestConnector_KnownResponders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("known.responders.connector")
	alpha.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	beta := New("known.responders.connector")
	beta.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	gamma := New("known.responders.connector")
	gamma.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	delta := New("known.responders.connector")
	delta.Subscribe("cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	// Startup the microservices
	err := alpha.Startup()
	assert.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	assert.NoError(t, err)
	defer beta.Shutdown()
	err = gamma.Startup()
	assert.NoError(t, err)
	defer gamma.Shutdown()

	check := func() (count int, quick bool) {
		responded := map[string]bool{}
		t0 := alpha.Now()
		ch := alpha.Publish(ctx, pub.GET("https://known.responders.connector/cast"), pub.Multicast())
		for i := range ch {
			res, err := i.Get()
			if assert.NoError(t, err) {
				responded[frame.Of(res).FromID()] = true
			}
		}
		dur := alpha.Clock().Since(t0)
		return len(responded), dur < AckTimeout
	}

	// First request should be slower, consecutive requests should be quick
	count, quick := check()
	assert.Equal(t, 3, count)
	assert.False(t, quick)
	count, quick = check()
	assert.Equal(t, 3, count)
	assert.True(t, quick)
	count, quick = check()
	assert.Equal(t, 3, count)
	assert.True(t, quick)

	// Add a new microservice
	err = delta.Startup()
	assert.NoError(t, err)

	// Should most likely get slow again once the new instance is discovered,
	// consecutive requests should be quick
	for count != 4 || !quick {
		count, quick = check()
	}
	count, quick = check()
	assert.Equal(t, 4, count)
	assert.True(t, quick)

	// Remove a microservice
	delta.Shutdown()

	// Should get slow again, consecutive requests should be quick
	count, quick = check()
	assert.Equal(t, 3, count)
	assert.False(t, quick)
	count, quick = check()
	assert.Equal(t, 3, count)
	assert.True(t, quick)
}

func TestConnector_LifetimeCancellation(t *testing.T) {
	t.Parallel()

	con := New("lifetime.cancellation.connector")

	done := false
	step := make(chan bool)
	con.Subscribe("something", func(w http.ResponseWriter, r *http.Request) error {
		step <- true
		<-r.Context().Done()
		done = true
		return r.Context().Err()
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	t0 := con.Now()
	go func() {
		_, err = con.Request(
			con.Lifetime(),
			pub.GET("https://lifetime.cancellation.connector/something"),
		)
		assert.Error(t, err)
		step <- true
	}()
	<-step
	con.ctxCancel()
	<-step
	assert.True(t, done)
	dur := con.Clock().Since(t0)
	assert.True(t, dur < time.Second)
}
