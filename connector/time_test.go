package connector

import (
	"context"
	"testing"
	"time"

	"github.com/microbus-io/fabric/cb"
	"github.com/microbus-io/fabric/clock"
	"github.com/stretchr/testify/assert"
)

func TestConnector_MockClockInProd(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("mock.clock.in.prod.connector")

	// OK before a deployment was set to PROD
	err := con.SetClock(mockClock)
	assert.NoError(t, err)

	// Should fail when deployment is set to PROD
	con.SetDeployment(PROD)
	err = con.SetClock(mockClock)
	assert.Error(t, err)

	// Should fail to start with the mock clock set
	err = con.Startup()
	assert.Error(t, err)
}

func TestConnector_Ticker(t *testing.T) {
	t.Parallel()

	mockClock1 := clock.NewMockAtNow()
	mockClock2 := clock.NewMockAtNow()

	con := New("ticker.connector")
	con.SetClock(mockClock1)

	count := 0
	con.StartTicker("myticker", time.Minute, func(ctx context.Context) error {
		count++
		return nil
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Zero(t, count)
	mockClock1.Add(time.Second + time.Minute)
	assert.Equal(t, 1, count)
	mockClock1.Add(2 * time.Minute)
	assert.Equal(t, 3, count)

	con.SetClock(mockClock2)
	mockClock2.Add(2 * time.Minute)
	assert.Equal(t, 5, count)
}

func TestConnector_TickerSkippingBeats(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("ticker.skipping.beats.connector")
	con.SetClock(mockClock)

	count := 0
	con.StartTicker("myticker", time.Minute, func(ctx context.Context) error {
		count++
		mockClock.Sleep(10*time.Second + 2*time.Minute)
		return nil
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	mockClock.Add(30 * time.Second)
	assert.Zero(t, count)
	// Ticker starts at 1:00
	mockClock.Add(1 * time.Minute) // 1:30
	assert.Equal(t, 1, count)
	mockClock.Add(1 * time.Minute) // 2:30
	assert.Equal(t, 1, count)
	// Ticker finished at 3:10
	mockClock.Add(1 * time.Minute) // 3:30
	assert.Equal(t, 1, count)
	// Ticker starts at 4:00
	mockClock.Add(1 * time.Minute) // 4:30
	assert.Equal(t, 2, count)
	mockClock.Add(1 * time.Minute) // 5:30
	assert.Equal(t, 2, count)
	// Ticker finished at 6:10
	mockClock.Add(1 * time.Minute) // 6:30
	assert.Equal(t, 2, count)
}

func TestConnector_Now(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("now.connector")
	con.SetClock(mockClock)

	assert.Equal(t, mockClock.Now(), con.Now())
	assert.Equal(t, mockClock.Now(), con.Clock().Now())
}

func TestConnector_TickerPendingOps(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("ticker.pending.ops.connector")
	con.SetClock(mockClock)

	con.StartTicker("myticker1", time.Minute, func(ctx context.Context) error {
		mockClock.Sleep(10 * time.Second)
		return nil
	})
	con.StartTicker("myticker2", time.Minute, func(ctx context.Context) error {
		mockClock.Sleep(20 * time.Second)
		return nil
	})

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Zero(t, con.pendingOps)
	mockClock.Add(59 * time.Second) // 0:59
	assert.Zero(t, con.pendingOps)
	mockClock.Add(2 * time.Second) // 1:01
	assert.Equal(t, int32(2), con.pendingOps)
	mockClock.Add(10 * time.Second) // 1:11
	assert.Equal(t, int32(1), con.pendingOps)
	mockClock.Add(10 * time.Second) // 1:21
	assert.Zero(t, con.pendingOps)
}

func TestConnector_StopTicker(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockAtNow()

	con := New("stop.ticker.connector")
	con.SetClock(mockClock)

	countAfter := 0
	con.StartTicker("after", time.Minute, func(ctx context.Context) error {
		countAfter++
		return nil
	})
	countBefore := 0
	con.StartTicker("before", time.Minute, func(ctx context.Context) error {
		countBefore++
		return nil
	})

	// Stop ticker before startup
	con.StopTicker("before")

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	assert.Zero(t, countAfter)
	assert.Zero(t, countBefore)
	mockClock.Add(time.Second + time.Minute) // 1:01
	assert.Equal(t, 1, countAfter)
	assert.Zero(t, countBefore)
	mockClock.Add(time.Minute) // 2:01
	assert.Equal(t, 2, countAfter)
	assert.Zero(t, countBefore)

	// Stop ticker after startup
	con.StopTicker("after")

	mockClock.Add(time.Minute) // 3:01
	assert.Equal(t, 2, countAfter)
	assert.Zero(t, countBefore)
	mockClock.Add(time.Hour) // 1:03:01
	assert.Equal(t, 2, countAfter)
	assert.Zero(t, countBefore)
}

func TestConnector_TickerTimeout(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMock()

	con := New("ticker.timeout.connector")
	con.SetClock(mockClock)

	var top, bottom bool
	con.StartTicker("ticker", time.Minute, func(ctx context.Context) error {
		top = true
		<-ctx.Done()
		bottom = true
		return nil
	}, cb.TimeBudget(time.Second*10))

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	mockClock.Add(time.Minute + 5*time.Second)
	assert.True(t, top)
	assert.False(t, bottom)
	mockClock.Add(10 * time.Second)
	assert.True(t, top)
	assert.True(t, bottom)
}

func TestConnector_TickerLifetimeCancellation(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMock()

	con := New("ticker.lifetime.cancellation.connector")
	con.SetClock(mockClock)

	var top, bottom bool
	step := make(chan bool)
	con.StartTicker("ticker", time.Minute, func(ctx context.Context) error {
		top = true
		step <- true
		<-ctx.Done()
		bottom = true
		step <- true
		return nil
	}, cb.TimeBudget(time.Minute))

	err := con.Startup()
	assert.NoError(t, err)
	defer con.Shutdown()

	mockClock.Add(time.Minute + 5*time.Second)
	assert.True(t, top)
	assert.False(t, bottom)
	<-step
	con.ctxCancel()
	<-step
	assert.True(t, top)
	assert.True(t, bottom)
}
