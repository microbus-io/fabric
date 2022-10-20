package application

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/connector"
	"github.com/stretchr/testify/assert"
)

func TestApplication_StartStop(t *testing.T) {
	t.Parallel()

	alpha := connector.NewConnector()
	alpha.SetHostName("alpha.startstop.application")
	beta := connector.NewConnector()
	beta.SetHostName("beta.startstop.application")
	app := New(alpha, beta)

	assert.False(t, alpha.IsStarted())
	assert.False(t, beta.IsStarted())
	assert.False(t, app.IsStarted())

	err := app.Startup()
	assert.NoError(t, err)

	assert.True(t, alpha.IsStarted())
	assert.True(t, beta.IsStarted())
	assert.True(t, app.IsStarted())

	err = app.Shutdown()
	assert.NoError(t, err)

	assert.False(t, alpha.IsStarted())
	assert.False(t, beta.IsStarted())
	assert.False(t, app.IsStarted())
}

func TestApplication_Interrupt(t *testing.T) {
	t.Parallel()

	con := connector.NewConnector()
	con.SetHostName("interrupt.application")
	app := New(con)

	ch := make(chan bool)
	go func() {
		err := app.Run()
		assert.NoError(t, err)
		ch <- true
	}()

	for !app.IsStarted() {
		time.Sleep(50 * time.Microsecond)
	}
	assert.True(t, app.IsStarted())

	app.Interrupt()

	for app.IsStarted() {
		time.Sleep(50 * time.Microsecond)
	}
	assert.False(t, app.IsStarted())
}

func TestApplication_NotStartedInterrupt(t *testing.T) {
	t.Parallel()

	con := connector.NewConnector()
	con.SetHostName("not.started.interrupt.application")
	app := New(con)

	err := app.WaitForInterrupt()
	assert.Error(t, err)
	err = app.Interrupt()
	assert.Error(t, err)

	err = app.Startup()
	assert.NoError(t, err)

	go func() {
		time.Sleep(50 * time.Millisecond)
		err := app.Interrupt()
		assert.NoError(t, err)
	}()
	err = app.WaitForInterrupt()
	assert.NoError(t, err)

	err = app.Shutdown()
	assert.NoError(t, err)

	err = app.WaitForInterrupt()
	assert.Error(t, err)
	err = app.Interrupt()
	assert.Error(t, err)
}

func TestApplication_NoConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create first testing app
	alpha := connector.NewConnector()
	alpha.SetHostName("noconflict.application")
	alpha.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})
	appAlpha := NewTesting(alpha)

	// Create second testing app
	beta := connector.NewConnector()
	beta.SetHostName("noconflict.application")
	beta.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("beta"))
		return nil
	})
	appBeta := NewTesting(beta)

	// Start the apps
	err := appAlpha.Startup()
	assert.NoError(t, err)
	defer appAlpha.Shutdown()
	err = appBeta.Startup()
	assert.NoError(t, err)
	defer appBeta.Shutdown()

	// Assert different planes of communication
	assert.NotEqual(t, alpha.Plane(), beta.Plane())
	assert.Equal(t, "LOCAL", alpha.Deployment())
	assert.Equal(t, "LOCAL", beta.Deployment())

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://noconflict.application/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("alpha"), body)
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://noconflict.application/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("beta"), body)
	}
}

func TestApplication_Clock(t *testing.T) {
	t.Parallel()

	con := connector.NewConnector()
	con.SetHostName("clock.application")
	app := New(con)

	mockClock1 := clock.NewMockAtNow()
	err := app.SetClock(mockClock1)
	assert.NoError(t, err)

	err = app.Startup()
	assert.NoError(t, err)

	mockClock2 := clock.NewMockAtNow()
	err = app.SetClock(mockClock2)
	assert.NoError(t, err)

	assert.Equal(t, mockClock2, con.Clock().(*clock.ClockReference).Get())

	err = app.Shutdown()
	assert.NoError(t, err)
}
