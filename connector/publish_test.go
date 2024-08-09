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
	"github.com/microbus-io/testarossa"
)

func TestConnector_Echo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.echo.connector")

	beta := New("beta.echo.connector")
	beta.Subscribe("POST", "echo", func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		testarossa.NoError(t, err)
		_, err = w.Write(body)
		testarossa.NoError(t, err)
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()

	// Send message and validate that it's echoed back
	response, err := alpha.POST(ctx, "https://beta.echo.connector/echo", []byte("Hello"))
	testarossa.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, []byte("Hello"), body)
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
	testarossa.Equal(b, 0, errCount)
	testarossa.Equal(b, int32(b.N), echoCount.Load())

	// On 2021 MacBook Pro M1 16":
	// N=12295
	// 95594 ns/op (10460 ops/sec)
	// 20024 B/op
	// 282 allocs/op
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
	testarossa.Equal(b, 0, errCount)
	testarossa.Equal(b, int32(b.N), echoCount.Load())

	// On 2021 MacBook Pro M1 16":
	// N=1174
	// 988411 ns/op (1012 ops/sec)
	// 247735 B/op
	// 3013 allocs/op
}

func BenchmarkConnector_EchoParallelMax(b *testing.B) {
	echoParallel(b, b.N)

	// On 2021 MacBook Pro M1 16":
	// N=91160 concurrent
	// 12577 ns/op (79510 ops/sec) = approx 8x that of serial
	// 19347 B/op
	// 280 allocs/op
}

func BenchmarkConnector_EchoParallel1K(b *testing.B) {
	echoParallel(b, 1000)

	// On 2021 MacBook Pro M1 16":
	// N=94744
	// 12102 ns/op (82630 ops/sec) = approx 8x that of serial
	// 19451 B/op
	// 278 allocs/op
}

func BenchmarkConnector_EchoParallel10K(b *testing.B) {
	echoParallel(b, 10000)

	// On 2021 MacBook Pro M1 16":
	// N=107904
	// 10575 ns/op (94562 ops/sec) = approx 9x that of serial
	// 19412 B/op
	// 278 allocs/op
}

func echoParallel(b *testing.B, concurrency int) {
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
	for i := range concurrency {
		tot := b.N / concurrency
		if i < b.N%concurrency {
			tot++
		} // do remainder
		go func() {
			for range tot {
				_, err := beta.POST(ctx, "https://alpha.echo.parallel.connector/echo", []byte("Hello"))
				if err != nil {
					errCount.Add(1)
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()
	b.StopTimer()
	testarossa.Equal(b, int32(0), errCount.Load())
	testarossa.Equal(b, int32(b.N), echoCount.Load())
}

func TestConnector_EchoParallelCapacity(t *testing.T) {
	t.Skip() // Dependent on strength of CPU running the test

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

	// Goroutines can take as much as 1s to start in very high load situations or slow CPUs
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
	testarossa.Zero(t, errCount.Load())
	testarossa.Equal(t, int32(n), echoCount.Load())

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
		testarossa.Equal(t, "not_empty", arg)
		return nil
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Send request with a query argument
	_, err = con.GET(ctx, "https://query.args.connector/arg?arg=not_empty")
	testarossa.NoError(t, err)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta1.Startup()
	testarossa.NoError(t, err)
	defer beta1.Shutdown()
	err = beta2.Startup()
	testarossa.NoError(t, err)
	defer beta2.Shutdown()

	// Send messages
	var wg sync.WaitGroup
	for i := 0; i < 256; i++ {
		wg.Add(1)
		go func() {
			_, err := alpha.GET(ctx, "https://beta.load.balancing.connector/lb")
			testarossa.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()

	// The requests should be more or less evenly distributed among the server microservices
	testarossa.Equal(t, int32(256), count1+count2)
	testarossa.True(t, count1 > 64)
	testarossa.True(t, count2 > 64)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
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
			testarossa.NoError(t, err)
			dur := start.Add(time.Millisecond * time.Duration(i)).Sub(end)
			testarossa.True(t, dur.Abs() <= time.Millisecond*49)
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
		testarossa.Equal(t, depth, step)
		testarossa.Equal(t, depth, frame.Of(r).CallDepth())

		_, err := con.GET(r.Context(), "https://call.depth.connector/next?step="+strconv.Itoa(step+1))
		testarossa.Error(t, err)
		testarossa.Contains(t, err.Error(), "call depth overflow")
		return errors.Trace(err)
	})

	// Startup the microservices
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	_, err = con.GET(ctx, "https://call.depth.connector/next?step=1")
	testarossa.Error(t, err)
	testarossa.Contains(t, err.Error(), "call depth overflow")
	testarossa.Equal(t, con.maxCallDepth, depth)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	budgetedCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	_, err = con.Request(
		budgetedCtx,
		pub.GET("https://timeout.drawdown.connector/next"),
	)
	testarossa.Error(t, err)
	testarossa.Equal(t, http.StatusRequestTimeout, errors.Convert(err).StatusCode)
	testarossa.True(t, depth >= 7 && depth <= 8, "%d", depth)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.context.connector/ok"),
	)
	if testarossa.NoError(t, err) {
		testarossa.False(t, deadline.IsZero())
		testarossa.True(t, time.Until(deadline) > time.Second*8, time.Until(deadline))
	}
}

func TestConnector_TimeoutNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservice
	con := New("timeout.not.found.connector")

	// Startup the microservice
	err := con.Startup()
	testarossa.NoError(t, err)
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
	testarossa.Error(t, err)
	testarossa.True(t, dur >= con.ackTimeout && dur < con.ackTimeout+time.Second)

	// Use the default time budget
	t0 = time.Now()
	_, err = con.Request(
		ctx,
		pub.GET("https://timeout.not.found.connector/nowhere"),
	)
	dur = time.Since(t0)
	testarossa.Error(t, err)
	testarossa.Equal(t, http.StatusNotFound, errors.Convert(err).StatusCode)
	testarossa.True(t, dur >= con.ackTimeout && dur < con.ackTimeout+time.Second)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	shortCtx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()
	t0 := time.Now()
	_, err = con.Request(
		shortCtx,
		pub.GET("https://timeout.slow.connector/slow"),
	)
	testarossa.Error(t, err)
	dur := time.Since(t0)
	testarossa.True(t, dur >= 500*time.Millisecond && dur < 600*time.Millisecond)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	shortCtx, cancel := context.WithTimeout(con.Lifetime(), time.Second)
	defer cancel()
	_, err = con.Request(
		shortCtx,
		pub.GET("https://context.timeout.connector/timeout"),
	)
	testarossa.Error(t, err)
	testarossa.True(t, done)
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
		testarossa.NoError(t, err)
		defer i.Shutdown()
	}

	// Make the first request
	client := named1
	t0 := time.Now()
	responded := map[string]bool{}
	ch := client.Publish(ctx, pub.GET("https://multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		res, err := i.Get()
		if testarossa.NoError(t, err) {
			body, err := io.ReadAll(res.Body)
			testarossa.NoError(t, err)
			responded[string(body)] = true
		}
	}
	dur := time.Since(t0)
	testarossa.True(t, dur >= ackTimeout && dur < ackTimeout+time.Second)
	testarossa.Equal(t, 4, len(responded))
	testarossa.True(t, responded["noqueue1"])
	testarossa.True(t, responded["noqueue2"])
	testarossa.True(t, responded["named1"] || responded["named2"])
	testarossa.False(t, responded["named1"] && responded["named2"])
	testarossa.True(t, responded["def1"] || responded["def2"])
	testarossa.False(t, responded["def1"] && responded["def2"])

	// Make the second request, should be quicker due to known responders optimization
	t0 = time.Now()
	responded = map[string]bool{}
	ch = client.Publish(ctx, pub.GET("https://multicast.connector/cast"), pub.Multicast())
	for i := range ch {
		res, err := i.Get()
		if testarossa.NoError(t, err) {
			body, err := io.ReadAll(res.Body)
			testarossa.NoError(t, err)
			responded[string(body)] = true
		}
	}
	dur = time.Since(t0)
	testarossa.True(t, dur < ackTimeout)
	testarossa.Equal(t, 4, len(responded))
	testarossa.True(t, responded["noqueue1"])
	testarossa.True(t, responded["noqueue2"])
	testarossa.True(t, responded["named1"] || responded["named2"])
	testarossa.False(t, responded["named1"] && responded["named2"])
	testarossa.True(t, responded["def1"] || responded["def2"])
	testarossa.False(t, responded["def1"] && responded["def2"])
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
	testarossa.NoError(t, err)
	defer slow.Shutdown()
	err = fast.Startup()
	testarossa.NoError(t, err)
	defer fast.Shutdown()
	err = tooSlow.Startup()
	testarossa.NoError(t, err)
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
	testarossa.True(t, dur >= 3*delay && dur < 4*delay)
	testarossa.Equal(t, 3, len(ch))
	testarossa.Equal(t, 3, cap(ch))
	for i := range ch {
		res, err := i.Get()
		if err == nil {
			body, err := io.ReadAll(res.Body)
			testarossa.NoError(t, err)
			testarossa.True(t, string(body) == "fast" || string(body) == "slow")
			respondedOK++
		} else {
			testarossa.Equal(t, http.StatusRequestTimeout, errors.Convert(err).StatusCode)
			respondedErr++
		}
	}
	testarossa.Equal(t, 2, respondedOK)
	testarossa.Equal(t, 1, respondedErr)
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
	testarossa.NoError(t, err)
	defer bad.Shutdown()
	err = good.Startup()
	testarossa.NoError(t, err)
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
	testarossa.True(t, dur >= good.ackTimeout && dur <= good.ackTimeout+time.Second)
	testarossa.Equal(t, 1, countErrs)
	testarossa.Equal(t, 1, countOKs)
}

func TestConnector_MulticastNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	con := New("multicast.not.found.connector")

	// Startup the microservice
	err := con.Startup()
	testarossa.NoError(t, err)
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
	testarossa.True(t, dur >= con.ackTimeout && dur < con.ackTimeout+time.Second)
	testarossa.Zero(t, count)
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
	testarossa.NoError(t, err)
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
			testarossa.NoError(t, err)
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
				testarossa.NoError(t, err)
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
		if testarossa.NoError(t, err) {
			countOKs++
		}
	}
	dur := time.Since(t0)
	testarossa.True(t, dur >= cons[0].ackTimeout && dur <= cons[0].ackTimeout+time.Second)
	testarossa.Equal(t, N, countOKs)
}

func BenchmarkConnector_NATSDirectPublishing(b *testing.B) {
	con := New("nats.direct.publishing.connector")

	err := con.Startup()
	testarossa.NoError(b, err)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()
	err = gamma.Startup()
	testarossa.NoError(t, err)
	defer gamma.Shutdown()

	check := func() (count int, quick bool) {
		responded := map[string]bool{}
		t0 := time.Now()
		ch := alpha.Publish(ctx, pub.GET("https://known.responders.connector/cast"), pub.Multicast())
		for i := range ch {
			res, err := i.Get()
			if testarossa.NoError(t, err) {
				responded[frame.Of(res).FromID()] = true
			}
		}
		dur := time.Since(t0)
		return len(responded), dur < alpha.ackTimeout
	}

	// First request should be slower, consecutive requests should be quick
	count, quick := check()
	testarossa.Equal(t, 3, count)
	testarossa.False(t, quick)
	count, quick = check()
	testarossa.Equal(t, 3, count)
	testarossa.True(t, quick)
	count, quick = check()
	testarossa.Equal(t, 3, count)
	testarossa.True(t, quick)

	// Add a new microservice
	err = delta.Startup()
	testarossa.NoError(t, err)

	// Should most likely get slow again once the new instance is discovered,
	// consecutive requests should be quick
	for count != 4 || !quick {
		count, quick = check()
	}
	count, quick = check()
	testarossa.Equal(t, 4, count)
	testarossa.True(t, quick)

	// Remove a microservice
	delta.Shutdown()

	// Should get slow again, consecutive requests should be quick
	count, quick = check()
	testarossa.Equal(t, 3, count)
	testarossa.False(t, quick)
	count, quick = check()
	testarossa.Equal(t, 3, count)
	testarossa.True(t, quick)
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
	testarossa.NoError(t, err)
	defer con.Shutdown()

	t0 := time.Now()
	go func() {
		_, err = con.Request(
			con.Lifetime(),
			pub.GET("https://lifetime.cancellation.connector/something"),
		)
		testarossa.Error(t, err)
		step <- true
	}()
	<-step
	con.ctxCancel()
	<-step
	testarossa.True(t, done)
	dur := time.Since(t0)
	testarossa.True(t, dur < time.Second)
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
			testarossa.NoError(t, err)
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
	testarossa.True(t, time.Since(t0) > time.Duration(n*100)*time.Millisecond)
	testarossa.Equal(t, n, int(responses.Load()))
	testarossa.Equal(t, n, len(ch))
	testarossa.Equal(t, n, cap(ch))

	// If asking for first response only, it should return immediately when it is produced
	responses.Store(0)
	t0 = time.Now()
	ch = cons[0].Publish(
		ctx,
		pub.GET("https://channel.capacity.connector/multicast"),
		pub.Unicast(),
	)
	testarossa.True(t, time.Since(t0) > 100*time.Millisecond && time.Since(t0) < 200*time.Millisecond)
	testarossa.Equal(t, 1, int(responses.Load()))
	testarossa.Equal(t, 1, len(ch))
	testarossa.Equal(t, 1, cap(ch))

	// The remaining handlers are still called and should finish
	time.Sleep(time.Duration(n*100) * time.Millisecond)
	testarossa.Equal(t, n, int(responses.Load()))
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
			testarossa.NoError(t, err)
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
	testarossa.NoError(t, err)
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
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()
	err = gamma.Startup()
	testarossa.NoError(t, err)
	defer gamma.Shutdown()

	// Send message and validate that it's echoed back
	_, err = alpha.Request(ctx,
		pub.GET("https://beta.baggage.connector/noop"),
		pub.Baggage("Suitcase", "Clothes"),
		pub.Header("Accept-Language", "en-US"),
		pub.Header("X-Forwarded-For", "1.2.3.4"),
	)
	testarossa.NoError(t, err)
	testarossa.True(t, betaCalled)
	testarossa.True(t, gammaCalled)
	testarossa.Equal(t, "Clothes", betaBaggage)
	testarossa.Equal(t, "Clothes", gammaBaggage)
	testarossa.Equal(t, "en-US", betaLanguage)
	testarossa.Equal(t, "en-US", gammaLanguage)
	testarossa.Equal(t, "1.2.3.4", betaXFwd)
	testarossa.Equal(t, "1.2.3.4", gammaXFwd)
}

func TestConnector_MultiValueHeader(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create the microservices
	alpha := New("alpha.multi.value.header.connector")

	beta := New("beta.multi.value.header.connector")
	beta.Subscribe("GET", "receive", func(w http.ResponseWriter, r *http.Request) error {
		testarossa.SliceLen(t, r.Header["Multi-Value-In"], 3)
		w.Header().Add("Multi-Value-Out", "1")
		w.Header().Add("Multi-Value-Out", "2")
		w.Header().Add("Multi-Value-Out", "3")
		return nil
	})

	// Startup the microservices
	err := alpha.Startup()
	testarossa.NoError(t, err)
	defer alpha.Shutdown()
	err = beta.Startup()
	testarossa.NoError(t, err)
	defer beta.Shutdown()

	// Send message and validate that it's echoed back
	response, err := alpha.Request(ctx,
		pub.GET("https://beta.multi.value.header.connector/receive"),
		pub.AddHeader("Multi-Value-In", "1"),
		pub.AddHeader("Multi-Value-In", "2"),
		pub.AddHeader("Multi-Value-In", "3"),
	)
	testarossa.NoError(t, err)
	testarossa.SliceLen(t, response.Header["Multi-Value-Out"], 3)
}
