/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"fmt"
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
	"github.com/microbus-io/fabric/rand"
	"github.com/microbus-io/fabric/sub"
	"github.com/stretchr/testify/assert"
)

func TestConnector_Echo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.echo.connector")

	beta := New("beta.echo.connector")
	beta.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
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
	alpha := New("alpha.echo.serial.connector")
	var echoCount atomic.Int32
	alpha.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
		echoCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
		return nil
	})

	beta := New("beta.echo.serial.connector")

	// Startup the microservice
	alpha.Startup()
	defer alpha.Shutdown()
	beta.Startup()
	defer beta.Shutdown()

	// The bottleneck is waiting on the network i/o
	var errCount int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beta.POST(ctx, "https://alpha.echo.serial.connector/echo", []byte("Hello"))
		if err != nil {
			errCount++
		}
	}
	b.StopTimer()
	assert.Zero(b, errCount)
	assert.Equal(b, int32(b.N), echoCount.Load())

	// On 2021 MacBook Pro M1 16":
	// N=10202
	// 107286 ns/op (9320 ops/sec)
	// 43195 B/op
	// 360 allocs/op
}

func BenchmarkConnector_SerialChain(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	con := New("serial.chain.connector")
	var echoCount atomic.Int32
	con.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
		if frame.Of(r).CallDepth() < 10 {
			// Go one level deeper
			res, err := con.POST(r.Context(), "https://serial.chain.connector/echo", r.Body)
			if err != nil {
				return errors.Trace(err)
			}
			body, _ := io.ReadAll(res.Body)
			w.Write(body)
		} else {
			// Echo back the request
			echoCount.Add(1)
			body, _ := io.ReadAll(r.Body)
			w.Write(body)
		}
		return nil
	})

	// Startup the microservice
	con.Startup()
	defer con.Shutdown()

	// The bottleneck is waiting on the network i/o
	var errCount int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := con.POST(ctx, "https://serial.chain.connector/echo", []byte("Hello"))
		if err != nil {
			errCount++
		}
	}
	b.StopTimer()
	assert.Zero(b, errCount)
	assert.Equal(b, int32(b.N), echoCount.Load())

	// On 2021 MacBook Pro M1 16":
	// N=703
	// 1504267 ns/op (664 ops/sec)
	// 522564 B/op
	// 3732 allocs/op
}

func BenchmarkConnector_EchoParallel(b *testing.B) {
	ctx := context.Background()

	// Create the microservice
	alpha := New("alpha.echo.parallel.connector")
	var echoCount atomic.Int32
	alpha.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
		echoCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
		return nil
	})

	// Goroutines can take as much as 500ms or more to start up during heavy load, which necessitates a high ack timeout
	beta := New("beta.echo.parallel.connector")
	beta.ackTimeout = time.Second

	// Startup the microservice
	alpha.Startup()
	defer alpha.Shutdown()
	beta.Startup()
	defer beta.Shutdown()

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	var errCount atomic.Int32
	for i := 0; i < b.N; i++ {
		go func() {
			_, err := beta.POST(ctx, "https://alpha.echo.parallel.connector/echo", []byte("Hello"))
			if err != nil {
				errCount.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
	assert.Zero(b, errCount.Load())
	assert.Equal(b, int32(b.N), echoCount.Load())

	// On 2021 MacBook Pro M1 16":
	// N=67580 concurrent
	// 16503 ns/op (60595 ops/sec) = approx 6x that of serial
	// 42269 B/op
	// 337 allocs/op
}

func TestConnector_EchoParallelCapacity(t *testing.T) {
	ctx := context.Background()

	// Create the microservice
	alpha := New("alpha.echo.parallel.capacity.connector")
	var echoCount atomic.Int32
	alpha.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
		echoCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
		return nil
	})

	beta := New("beta.echo.parallel.capacity.connector")

	// Startup the microservice
	alpha.Startup()
	defer alpha.Shutdown()
	beta.Startup()
	defer beta.Shutdown()

	// Goroutines can take as much as 1s to start in very high load situations
	n := 10000
	var wg sync.WaitGroup
	wg.Add(n)
	t0 := time.Now()
	var totalTime atomic.Int64
	var maxTime atomic.Int32
	var errCount atomic.Int32
	for i := 0; i < n; i++ {
		go func() {
			tts := int(time.Since(t0).Milliseconds())
			totalTime.Add(int64(tts))
			currentMax := maxTime.Load()
			if int32(tts) > currentMax {
				maxTime.Store(int32(tts))
			}
			_, err := beta.POST(ctx, "https://alpha.echo.parallel.capacity.connector/echo", []byte("Hello"))
			if err != nil {
				errCount.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Zero(t, errCount.Load())
	assert.Equal(t, int32(n), echoCount.Load())

	fmt.Printf("errs %d\n", errCount.Load())
	fmt.Printf("echo %d\n", echoCount.Load())
	fmt.Printf("avg time to start %d\n", totalTime.Load()/int64(n))
	fmt.Printf("max time to start %d\n", maxTime.Load())

	// On 2021 MacBook Pro M1 16":
	// n=10000 avg=56 max=133
	// n=20000 avg=148 max=308 ackTimeout=1s
	// n=40000 avg=318 max=569 ackTimeout=1s
	// n=60000 avg=501 max=935 ackTimeout=1s
}

func TestConnector_QueryArgs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	con := New("query.args.connector")
	con.Subscribe("GET", "arg", func(w http.ResponseWriter, r *http.Request) error {
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
	beta1.Subscribe("GET", "lb", func(w http.ResponseWriter, r *http.Request) error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	beta2 := New("beta.load.balancing.connector")
	beta2.Subscribe("GET", "lb", func(w http.ResponseWriter, r *http.Request) error {
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
	beta.Subscribe("GET", "wait", func(w http.ResponseWriter, r *http.Request) error {
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
	con := New("call.depth.connector")
	con.maxCallDepth = 8
	con.Subscribe("GET", "next", func(w http.ResponseWriter, r *http.Request) error {
		depth++

		step, _ := strconv.Atoi(r.URL.Query().Get("step"))
		assert.Equal(t, depth, step)
		assert.Equal(t, depth, frame.Of(r).CallDepth())

		_, err := con.GET(r.Context(), "https://call.depth.connector/next?step="+strconv.Itoa(step+1))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call depth overflow")
		return errors.Trace(err)
	})

	// Startup the microservices
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://call.depth.connector/next?step=1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "call depth overflow")
	assert.Equal(t, con.maxCallDepth, depth)
}

func TestConnector_TimeoutDrawdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	depth := 0

	// Create the microservice
	con := New("timeout.drawdown.connector")
	budget := con.networkHop * 8
	con.Subscribe("GET", "next", func(w http.ResponseWriter, r *http.Request) error {
		depth++
		_, err := con.GET(r.Context(), "https://timeout.drawdown.connector/next")
		return errors.Trace(err)
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	budgetedCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	_, err = con.Request(
		budgetedCtx,
		pub.GET("https://timeout.drawdown.connector/next"),
	)
	assert.Error(t, err)
	assert.Equal(t, http.StatusRequestTimeout, errors.Convert(err).StatusCode)
	assert.True(t, depth >= 7 && depth <= 8, "%d", depth)
}

func TestConnector_TimeoutContext(t *testing.T) {
	t.Parallel()

	// Create the microservice
	con := New("timeout.context.connector")
	var deadline time.Time
	con.Subscribe("GET", "ok", func(w http.ResponseWriter, r *http.Request) error {
		deadline, _ = r.Context().Deadline()
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.context.connector/ok"),
	)
	assert.NoError(t, err)
	assert.False(t, deadline.IsZero())
	assert.True(t, time.Until(deadline) > time.Second*58, time.Until(deadline))
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
	shortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	t0 := time.Now()
	_, err = con.Request(
		shortCtx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
	)
	dur := time.Since(t0)
	assert.Error(t, err)
	assert.GreaterOrEqual(t, dur, con.ackTimeout)
	assert.Less(t, dur, con.ackTimeout+time.Second)

	// Use the default time budget
	t0 = time.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
	)
	dur = time.Since(t0)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	assert.GreaterOrEqual(t, dur, con.ackTimeout)
	assert.Less(t, dur, con.ackTimeout+time.Second)
}

func TestConnector_TimeoutSlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("timeout.slow.connector")
	con.Subscribe("GET", "slow", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(time.Second)
		return nil
	})

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	shortCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()
	t0 := time.Now()
	_, err = con.Request(
		shortCtx,
		pub.GET("https://timeout.slow.connector/slow"),
	)
	assert.Error(t, err)
	dur := time.Since(t0)
	assert.GreaterOrEqual(t, dur, 500*time.Millisecond)
	assert.Less(t, dur, 600*time.Millisecond)
}

func TestConnector_ContextTimeout(t *testing.T) {
	t.Parallel()

	con := New("context.timeout.connector")

	done := false
	con.Subscribe("GET", "timeout", func(w http.ResponseWriter, r *http.Request) error {
		<-r.Context().Done()
		done = true
		return r.Context().Err()
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	shortCtx, cancel := context.WithTimeout(con.Lifetime(), time.Second)
	defer cancel()
	_, err = con.Request(
		shortCtx,
		pub.GET("https://context.timeout.connector/timeout"),
	)
	assert.Error(t, err)
	assert.True(t, done)
}

func TestConnector_Multicast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	noqueue1 := New("multicast.connector")
	noqueue1.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue1"))
		return nil
	}, sub.NoQueue())

	noqueue2 := New("multicast.connector")
	noqueue2.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("noqueue2"))
		return nil
	}, sub.NoQueue())

	named1 := New("multicast.connector")
	named1.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named1"))
		return nil
	}, sub.Queue("MyQueue"))

	named2 := New("multicast.connector")
	named2.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("named2"))
		return nil
	}, sub.Queue("MyQueue"))

	def1 := New("multicast.connector")
	def1.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def1"))
		return nil
	}, sub.DefaultQueue())

	def2 := New("multicast.connector")
	def2.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("def2"))
		return nil
	}, sub.DefaultQueue())

	ackTimeout := New("").ackTimeout

	// Startup the microservices
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
	assert.GreaterOrEqual(t, dur, ackTimeout)
	assert.Less(t, dur, ackTimeout+time.Second)
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
	assert.True(t, dur < ackTimeout)
	assert.Len(t, responded, 4)
	assert.True(t, responded["noqueue1"])
	assert.True(t, responded["noqueue2"])
	assert.True(t, responded["named1"] || responded["named2"])
	assert.False(t, responded["named1"] && responded["named2"])
	assert.True(t, responded["def1"] || responded["def2"])
	assert.False(t, responded["def1"] && responded["def2"])
}

func TestConnector_MulticastPartialTimeout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	delay := time.Millisecond * 500

	// Create the microservices
	slow := New("multicast.partial.timeout.connector")
	slow.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(delay * 2)
		w.Write([]byte("slow"))
		return nil
	}, sub.NoQueue())

	fast := New("multicast.partial.timeout.connector")
	fast.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("fast"))
		return nil
	}, sub.NoQueue())

	tooSlow := New("multicast.partial.timeout.connector")
	tooSlow.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
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
	shortCtx, cancel := context.WithTimeout(ctx, delay*3)
	defer cancel()
	var respondedOK, respondedErr int
	t0 := time.Now()
	ch := slow.Publish(
		shortCtx,
		pub.GET("https://multicast.partial.timeout.connector/cast"),
		pub.Multicast(),
	)
	dur := time.Since(t0)
	assert.GreaterOrEqual(t, dur, 3*delay)
	assert.Less(t, dur, 4*delay)
	assert.Equal(t, 3, len(ch))
	assert.Equal(t, 3, cap(ch))
	for i := range ch {
		res, err := i.Get()
		if err == nil {
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.True(t, string(body) == "fast" || string(body) == "slow")
			respondedOK++
		} else {
			assert.Equal(t, http.StatusRequestTimeout, errors.Convert(err).StatusCode)
			respondedErr++
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
	bad.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("bad situation")
	}, sub.NoQueue())

	good := New("multicast.error.connector")
	good.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
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
	t0 := time.Now()
	ch := bad.Publish(ctx, pub.GET("https://multicast.error.connector/cast"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		if err != nil {
			countErrs++
		} else {
			countOKs++
		}
	}
	dur := time.Since(t0)
	assert.GreaterOrEqual(t, dur, good.ackTimeout)
	assert.Less(t, dur, good.ackTimeout+time.Second)
	assert.Equal(t, 1, countErrs)
	assert.Equal(t, 1, countOKs)
}

func TestConnector_MulticastNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	con := New("multicast.not.found.connector")

	// Startup the microservice
	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	// Send the message
	var count int
	t0 := time.Now()
	ch := con.Publish(ctx, pub.GET("https://multicast.not.found.connector/nowhere"), pub.Multicast())
	for i := range ch {
		i.Get()
		count++
	}
	dur := time.Since(t0)
	assert.GreaterOrEqual(t, dur, con.ackTimeout)
	assert.Less(t, dur, con.ackTimeout+time.Second)
	assert.Equal(t, 0, count)
}

func TestConnector_MassMulticast(t *testing.T) {
	// No parallel

	ctx := context.Background()
	randomPlane := rand.AlphaNum64(12)
	N := 128

	// Create the client microservice
	client := New("client.mass.multicast.connector")
	client.SetDeployment(TESTING)
	client.SetPlane(randomPlane)

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
			cons[i].SetDeployment(TESTING)
			cons[i].SetPlane(randomPlane)
			cons[i].Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
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
	t0 := time.Now()
	ch := client.Publish(ctx, pub.GET("https://mass.multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		_, err := i.Get()
		if assert.NoError(t, err) {
			countOKs++
		}
	}
	dur := time.Since(t0)
	assert.GreaterOrEqual(t, dur, cons[0].ackTimeout)
	assert.Less(t, dur, cons[0].ackTimeout+time.Second)
	assert.Equal(t, N, countOKs)
}

func BenchmarkConnector_NATSDirectPublishing(b *testing.B) {
	con := New("nats.direct.publishing.connector")

	err := con.Startup()
	assert.NoError(b, err)
	defer con.Shutdown()

	body := make([]byte, 512*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		con.natsConn.Publish("somewhere", body)
	}
	b.StopTimer()

	// On 2021 MacBook Pro M1 16":
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
	alpha.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	beta := New("known.responders.connector")
	beta.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	gamma := New("known.responders.connector")
	gamma.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}, sub.NoQueue())

	delta := New("known.responders.connector")
	delta.Subscribe("GET", "cast", func(w http.ResponseWriter, r *http.Request) error {
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
		t0 := time.Now()
		ch := alpha.Publish(ctx, pub.GET("https://known.responders.connector/cast"), pub.Multicast())
		for i := range ch {
			res, err := i.Get()
			if assert.NoError(t, err) {
				responded[frame.Of(res).FromID()] = true
			}
		}
		dur := time.Since(t0)
		return len(responded), dur < alpha.ackTimeout
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
	con.Subscribe("GET", "something", func(w http.ResponseWriter, r *http.Request) error {
		step <- true
		<-r.Context().Done()
		done = true
		return r.Context().Err()
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	t0 := time.Now()
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
	dur := time.Since(t0)
	assert.True(t, dur < time.Second)
}

func TestConnector_ChannelCapacity(t *testing.T) {
	t.Parallel()

	n := 8

	// Create microservices that respond in a staggered timeline
	var responses atomic.Int32
	var wg sync.WaitGroup
	cons := make([]*Connector, n)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			cons[i] = New("channel.capacity.connector")
			cons[i].SetDeployment(TESTING)
			cons[i].Subscribe("GET", "multicast", func(w http.ResponseWriter, r *http.Request) error {
				time.Sleep(time.Duration(100*i+100) * time.Millisecond)
				responses.Add(1)
				return nil
			}, sub.NoQueue())
			err := cons[i].Startup()
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()
	defer func() {
		for i := 0; i < n; i++ {
			cons[i].Shutdown()
		}
	}()

	ctx := context.Background()

	// All responses should come in at once after all handlers finished
	responses.Store(0)
	t0 := time.Now()
	cons[0].multicastChanCap = n / 2 // Limited multicast channel capacity should not block
	ch := cons[0].Publish(
		ctx,
		pub.GET("https://channel.capacity.connector/multicast"),
	)
	assert.Greater(t, time.Since(t0), time.Duration(n*100)*time.Millisecond)
	assert.Equal(t, n, int(responses.Load()))
	assert.Equal(t, n, len(ch))
	assert.Equal(t, n, cap(ch))

	// If asking for first response only, it should return immediately when it is produced
	responses.Store(0)
	t0 = time.Now()
	ch = cons[0].Publish(
		ctx,
		pub.GET("https://channel.capacity.connector/multicast"),
		pub.Unicast(),
	)
	assert.Greater(t, time.Since(t0), 100*time.Millisecond)
	assert.Less(t, time.Since(t0), 200*time.Millisecond)
	assert.Equal(t, 1, int(responses.Load()))
	assert.Equal(t, 1, len(ch))
	assert.Equal(t, 1, cap(ch))

	// The remaining handlers are still called and should finish
	time.Sleep(time.Duration(n*100) * time.Millisecond)
	assert.Equal(t, n, int(responses.Load()))
}

func TestConnector_UnicastToNoQueue(t *testing.T) {
	t.Parallel()

	n := 8
	var wg sync.WaitGroup
	wg.Add(n)
	cons := make([]*Connector, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			cons[i] = New("unicast.to.no.queue.connector")
			cons[i].SetDeployment(TESTING)
			cons[i].Subscribe("GET", "no-queue", func(w http.ResponseWriter, r *http.Request) error {
				return nil
			}, sub.NoQueue())
			err := cons[i].Startup()
			assert.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()
	defer func() {
		for i := 0; i < n; i++ {
			cons[i].Shutdown()
		}
	}()

	_, err := cons[0].Request(
		cons[0].Lifetime(),
		pub.GET("https://unicast.to.no.queue.connector/no-queue"),
	)
	assert.NoError(t, err)
}

func TestConnector_Baggage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.baggage.connector")

	betaCalled := false
	betaBaggage := ""
	betaLanguage := ""
	betaXFwd := ""
	beta := New("beta.baggage.connector")
	beta.Subscribe("GET", "noop", func(w http.ResponseWriter, r *http.Request) error {
		betaCalled = true
		betaBaggage = frame.Of(r).Baggage("Suitcase")
		betaLanguage = r.Header.Get("Accept-Language")
		betaXFwd = r.Header.Get("X-Forwarded-For")
		beta.GET(r.Context(), "https://gamma.baggage.connector/noop")
		return nil
	})

	gammaCalled := false
	gammaBaggage := ""
	gammaLanguage := ""
	gammaXFwd := ""
	gamma := New("gamma.baggage.connector")
	gamma.Subscribe("GET", "noop", func(w http.ResponseWriter, r *http.Request) error {
		gammaCalled = true
		gammaBaggage = frame.Of(r).Baggage("Suitcase")
		gammaLanguage = r.Header.Get("Accept-Language")
		gammaXFwd = r.Header.Get("X-Forwarded-For")
		return nil
	})

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

	// Send message and validate that it's echoed back
	_, err = alpha.Request(ctx,
		pub.GET("https://beta.baggage.connector/noop"),
		pub.Baggage("Suitcase", "Clothes"),
		pub.Header("Accept-Language", "en-US"),
		pub.Header("X-Forwarded-For", "1.2.3.4"),
	)
	assert.NoError(t, err)
	assert.True(t, betaCalled)
	assert.True(t, gammaCalled)
	assert.Equal(t, "Clothes", betaBaggage)
	assert.Equal(t, "Clothes", gammaBaggage)
	assert.Equal(t, "en-US", betaLanguage)
	assert.Equal(t, "en-US", gammaLanguage)
	assert.Equal(t, "1.2.3.4", betaXFwd)
	assert.Equal(t, "1.2.3.4", gammaXFwd)
}
