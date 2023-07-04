/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/clock"
	"github.com/stretchr/testify/assert"
)

func TestConnector_NoMockClockInProd(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("no.mock.clock.in.prod.connector")

	// OK before a deployment was set to PROD
	err := con.SetClock(mockClock)
	assert.NoError(t, err)

	// OK before service was started
	con.SetDeployment(PROD)
	err = con.SetClock(mockClock)
	assert.NoError(t, err)

	// Should fail to start with the mock clock set
	err = con.Startup()
	assert.Error(t, err)
}

func TestConnector_Ticker(t *testing.T) {
	t.Parallel()

	con := New("ticker.connector")

	interval := 200 * time.Millisecond
	count := 0
	step := make(chan bool)
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		step <- true
		return nil
	})

	assert.Zero(t, count)

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	<-step // at 1 intervals
	assert.Equal(t, 1, count)
	time.Sleep(interval / 2) // at 1.5 intervals
	assert.Equal(t, 1, count)
	<-step // at 2 intervals
	assert.Equal(t, 2, count)
	<-step // at 3 intervals
	assert.Equal(t, 3, count)
}

func TestConnector_TickerSkippingBeats(t *testing.T) {
	t.Parallel()

	con := New("ticker.skipping.beats.connector")

	interval := 200 * time.Millisecond
	count := 0
	step := make(chan bool)
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		step <- true
		time.Sleep(2*interval + interval/4) // 2.25 intervals
		return nil
	})

	assert.Zero(t, count)

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	<-step // at 1 intervals
	assert.Equal(t, 1, count)
	time.Sleep(interval + interval/2) // at 2.5 intervals
	assert.Equal(t, 1, count)
	time.Sleep(interval) // at 3.5 intervals
	assert.Equal(t, 1, count)

	<-step // at 4 intervals
	assert.Equal(t, 2, count)
}

func TestConnector_ClockNow(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("clock.now.connector")
	con.SetClock(mockClock)

	assert.True(t, mockClock.Now().Equal(con.Now()))
	assert.True(t, mockClock.Now().Equal(con.Clock().Now()))
}

func TestConnector_TickerPendingOps(t *testing.T) {
	t.Parallel()

	con := New("ticker.pending.ops.connector")

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

	assert.Zero(t, con.pendingOps)

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	<-step1 // at 1 intervals
	<-step2 // at 1 intervals
	assert.Equal(t, int32(2), con.pendingOps)
	<-hold1
	time.Sleep(interval / 4) // at 1.25 intervals
	assert.Equal(t, int32(1), con.pendingOps)
	<-hold2 // at 1.5 intervals
	time.Sleep(interval / 4)
	assert.Zero(t, con.pendingOps)
}

func TestConnector_TickerTimeout(t *testing.T) {
	t.Parallel()

	con := New("ticker.timeout.connector")

	interval := 400 * time.Millisecond
	start := make(chan bool)
	end := make(chan bool)
	con.StartTicker("ticker", interval, func(ctx context.Context) error {
		start <- true
		<-ctx.Done()
		end <- true
		return nil
	}, cb.TimeBudget(interval/4))

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	<-start
	t0 := time.Now()
	<-end
	dur := time.Since(t0)
	assert.True(t, dur > interval/4 && dur < interval/2, "%v", dur)
}

func TestConnector_TickerLifetimeCancellation(t *testing.T) {
	t.Parallel()

	con := New("ticker.lifetime.cancellation.connector")

	interval := 200 * time.Millisecond
	start := make(chan bool)
	end := make(chan bool)
	con.StartTicker("ticker", interval, func(ctx context.Context) error {
		start <- true
		<-ctx.Done()
		end <- true
		return nil
	}, cb.TimeBudget(time.Minute))

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	<-start
	t0 := time.Now()
	con.ctxCancel() // Cancel the lifetime context
	<-end
	dur := time.Since(t0)
	assert.True(t, dur < interval)
}

func TestConnector_TickersDisabledInTestingApp(t *testing.T) {
	t.Parallel()

	con := New("tickers.disabled.in.testing.app.connector")
	con.SetDeployment(TESTINGAPP)

	interval := 200 * time.Millisecond
	count := 0
	con.StartTicker("myticker", interval, func(ctx context.Context) error {
		count++
		return nil
	})

	assert.Zero(t, count)

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	time.Sleep(5 * interval)
	assert.Zero(t, count)
}
