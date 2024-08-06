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
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/testarossa"
)

func TestConnector_ClockOffset(t *testing.T) {
	t.Parallel()

	// Create the microservices
	alpha := New("alpha.clock.offset.connector")

	var betaTime time.Time
	var betaShift time.Duration
	beta := New("beta.clock.offset.connector")
	beta.Subscribe("GET", "shift", func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		betaTime = beta.Now(ctx)
		betaShift = frame.Of(ctx).ClockShift()
		beta.GET(r.Context(), "https://gamma.clock.offset.connector/shift")
		return nil
	})

	var gammaTime time.Time
	var gammaShift time.Duration
	gamma := New("gamma.clock.offset.connector")
	gamma.Subscribe("GET", "shift", func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		gammaTime = beta.Now(ctx)
		gammaShift = frame.Of(ctx).ClockShift()
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

	// Shift the time in the context one minute in the past
	ctx := frame.ContextWithFrame(context.Background())
	frame.Of(ctx).SetClockShift(-time.Minute)

	// Send message and validate that beta receives the offset time
	realTime := time.Now()
	time.Sleep(10 * time.Millisecond)
	alphaTime := alpha.Now(ctx) // Offset by -1m
	testarossa.True(t, alphaTime.Before(realTime))
	_, err = alpha.GET(ctx, "https://beta.clock.offset.connector/shift")
	testarossa.NoError(t, err)
	testarossa.True(t, betaTime.Before(realTime))
	testarossa.True(t, gammaTime.Before(realTime))
	testarossa.Equal(t, -time.Minute, betaShift)
	testarossa.Equal(t, -time.Minute, gammaShift)

	// Shift the time in the context one hour in the future
	ctx = frame.ContextWithFrame(context.Background())
	frame.Of(ctx).SetClockShift(time.Hour)

	// Send message and validate that beta receives the offset time
	realTime = time.Now()
	alphaTime = alpha.Now(ctx) // Offset by +1h
	testarossa.True(t, alphaTime.After(realTime.Add(time.Minute)))
	_, err = alpha.GET(ctx, "https://beta.clock.offset.connector/shift")
	testarossa.NoError(t, err)
	testarossa.True(t, betaTime.After(realTime.Add(time.Minute)))
	testarossa.True(t, gammaTime.After(realTime.Add(time.Minute)))
	testarossa.Equal(t, time.Hour, betaShift)
	testarossa.Equal(t, time.Hour, gammaShift)
}

func TestConnector_Ticker(t *testing.T) {
	t.Parallel()

	con := New("ticker.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 200 * time.Millisecond
	count := 0
	step := make(chan bool)
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		step <- true
		return nil
	})

	testarossa.Zero(t, count)

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-step // at 1 intervals
	testarossa.Equal(t, 1, count)
	time.Sleep(interval / 2) // at 1.5 intervals
	testarossa.Equal(t, 1, count)
	<-step // at 2 intervals
	testarossa.Equal(t, 2, count)
	<-step // at 3 intervals
	testarossa.Equal(t, 3, count)
}

func TestConnector_TickerSkippingBeats(t *testing.T) {
	t.Parallel()

	con := New("ticker.skipping.beats.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 200 * time.Millisecond
	count := 0
	step := make(chan bool)
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		step <- true
		time.Sleep(2*interval + interval/4) // 2.25 intervals
		return nil
	})

	testarossa.Zero(t, count)

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-step // at 1 intervals
	testarossa.Equal(t, 1, count)
	time.Sleep(interval + interval/2) // at 2.5 intervals
	testarossa.Equal(t, 1, count)
	time.Sleep(interval) // at 3.5 intervals
	testarossa.Equal(t, 1, count)

	<-step // at 4 intervals
	testarossa.Equal(t, 2, count)
}

func TestConnector_TickerPendingOps(t *testing.T) {
	t.Parallel()

	con := New("ticker.pending.ops.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 200 * time.Millisecond
	step1 := make(chan bool)
	hold1 := make(chan bool)
	step2 := make(chan bool)
	hold2 := make(chan bool)
	con.StartTicker("myticker1", interval, func(ctx context.Context) error {
		step1 <- true
		hold1 <- true
		return nil
	})
	con.StartTicker("myticker2", interval, func(ctx context.Context) error {
		step2 <- true
		hold2 <- true
		return nil
	})

	testarossa.Zero(t, con.pendingOps)

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-step1 // at 1 intervals
	<-step2 // at 1 intervals
	testarossa.Equal(t, int32(2), con.pendingOps)
	<-hold1
	time.Sleep(interval / 4) // at 1.25 intervals
	testarossa.Equal(t, int32(1), con.pendingOps)
	<-hold2 // at 1.5 intervals
	time.Sleep(interval / 4)
	testarossa.Zero(t, con.pendingOps)
}

func TestConnector_TickerTimeout(t *testing.T) {
	t.Parallel()

	con := New("ticker.timeout.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 400 * time.Millisecond
	start := make(chan bool)
	end := make(chan bool)
	con.StartTicker("ticker", interval, func(ctx context.Context) error {
		start <- true
		ctx, cancel := context.WithTimeout(ctx, interval/4)
		defer cancel()
		<-ctx.Done()
		end <- true
		return nil
	})

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-start
	t0 := time.Now()
	<-end
	dur := time.Since(t0)
	testarossa.True(t, dur > interval/4 && dur < interval/2, "%v", dur)
}

func TestConnector_TickerLifetimeCancellation(t *testing.T) {
	t.Parallel()

	con := New("ticker.lifetime.cancellation.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 200 * time.Millisecond
	start := make(chan bool)
	end := make(chan bool)
	con.StartTicker("ticker", interval, func(ctx context.Context) error {
		start <- true
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		<-ctx.Done()
		end <- true
		return nil
	})

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-start
	t0 := time.Now()
	con.ctxCancel() // Cancel the lifetime context
	<-end
	dur := time.Since(t0)
	testarossa.True(t, dur < interval)
}

func TestConnector_TickersDisabledInTestingApp(t *testing.T) {
	t.Parallel()

	con := New("tickers.disabled.in.testing.app.connector")

	interval := 200 * time.Millisecond
	count := 0
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		return nil
	})

	testarossa.Zero(t, count)

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	time.Sleep(5 * interval)
	testarossa.Zero(t, count)
}

func TestConnector_TickerStop(t *testing.T) {
	t.Parallel()

	con := New("ticker.stop.connector")
	con.SetDeployment(LAB) // Tickers are disabled in TESTING

	interval := 200 * time.Millisecond
	count := 0
	enter := make(chan bool)
	exit := make(chan bool)
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		enter <- true
		exit <- true
		return nil
	})

	testarossa.Zero(t, count)

	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	<-enter
	testarossa.Equal(t, 1, count)
	con.StopTicker("myticker")
	<-exit

	time.Sleep(2 * interval)
	testarossa.Equal(t, 1, count)

	// Restart
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		enter <- true
		exit <- true
		return nil
	})

	<-enter
	testarossa.Equal(t, 2, count)
	<-exit
}
