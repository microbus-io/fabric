package application

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/connector"
	"github.com/stretchr/testify/assert"
)

func TestApplication_StartStop(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.startstop.application")
	beta := connector.New("beta.startstop.application")
	app := New(alpha, beta)

	assert.False(t, alpha.IsStarted())
	assert.False(t, beta.IsStarted())

	err := app.Startup()
	assert.NoError(t, err)

	assert.True(t, alpha.IsStarted())
	assert.True(t, beta.IsStarted())

	err = app.Shutdown()
	assert.NoError(t, err)

	assert.False(t, alpha.IsStarted())
	assert.False(t, beta.IsStarted())
}

func TestApplication_Interrupt(t *testing.T) {
	t.Parallel()

	con := connector.New("interrupt.application")
	app := New(con)

	ch := make(chan bool)
	go func() {
		err := app.Startup()
		assert.NoError(t, err)
		go func() {
			app.WaitForInterrupt()
			err := app.Shutdown()
			assert.NoError(t, err)
			ch <- true
		}()
		ch <- true
	}()

	<-ch
	assert.True(t, con.IsStarted())
	app.Interrupt()
	<-ch
	assert.False(t, con.IsStarted())
}

func TestApplication_NoConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create first testing app
	alpha := connector.New("noconflict.application")
	alpha.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})
	appAlpha := NewTesting(alpha)

	// Create second testing app
	beta := connector.New("noconflict.application")
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
