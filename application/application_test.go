package application

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/services/configurator"
	"github.com/stretchr/testify/assert"
)

func TestApplication_StartStop(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.start.stop.application")
	beta := connector.New("beta.start.stop.application")
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
	alpha := connector.New("no.conflict.application")
	alpha.Subscribe("id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})
	appAlpha := NewTesting(alpha)

	// Create second testing app
	beta := connector.New("no.conflict.application")
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
	assert.Equal(t, connector.TESTINGAPP, alpha.Deployment())
	assert.Equal(t, connector.TESTINGAPP, beta.Deployment())

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://no.conflict.application/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("alpha"), body)
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://no.conflict.application/id")
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("beta"), body)
	}
}

func TestApplication_DependencyStart(t *testing.T) {
	t.Parallel()

	startupTimeout := time.Second * 2

	// Alpha is dependent on beta to start
	failCount := 0
	alpha := connector.New("alpha.dependency.start.application")
	alpha.SetOnStartup(func(ctx context.Context) error {
		_, err := alpha.Request(ctx, pub.GET("https://beta.dependency.start.application/ok"))
		if err != nil {
			failCount++
			return errors.Trace(err)
		}
		return nil
	})

	// Beta takes a bit of time to start
	beta := connector.New("beta.dependency.start.application")
	beta.Subscribe("/ok", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	beta.SetOnStartup(func(ctx context.Context) error {
		time.Sleep(startupTimeout / 2)
		return nil
	})

	app := New(alpha, beta)
	app.startupTimeout = startupTimeout
	t0 := time.Now()
	err := app.Startup()
	dur := time.Since(t0)
	assert.NoError(t, err)
	assert.True(t, failCount > 0)
	assert.True(t, dur >= startupTimeout/2)

	app.Shutdown()
}

func TestApplication_FailStart(t *testing.T) {
	t.Parallel()

	startupTimeout := time.Second

	// Alpha fails to start
	failCount := 0
	alpha := connector.New("alpha.fail.start.application")
	alpha.SetOnStartup(func(ctx context.Context) error {
		failCount++
		return errors.New("oops")
	})

	// Beta starts without a hitch
	beta := connector.New("beta.fail.start.application")

	app := New(alpha, beta)
	app.startupTimeout = startupTimeout
	t0 := time.Now()
	err := app.Startup()
	dur := time.Since(t0)
	assert.Error(t, err)
	assert.True(t, failCount > 0)
	assert.True(t, dur >= startupTimeout)
	assert.True(t, beta.IsStarted())
	assert.False(t, alpha.IsStarted())

	err = app.Shutdown()
	assert.NoError(t, err)
	assert.False(t, beta.IsStarted())
	assert.False(t, alpha.IsStarted())
}

func TestApplication_JoinInclude(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.join.include.application")
	beta := connector.New("beta.join.include.application")
	gamma := connector.New("gamma.join.include.application")

	app := NewTesting(alpha)
	app.Include(beta)
	app.Join(gamma)

	assert.Equal(t, alpha.Plane(), beta.Plane())
	assert.Equal(t, alpha.Plane(), gamma.Plane())

	err := app.Startup()
	assert.NoError(t, err)
	assert.True(t, alpha.IsStarted())
	assert.True(t, beta.IsStarted())
	assert.False(t, gamma.IsStarted())

	err = gamma.Startup()
	assert.NoError(t, err)
	assert.True(t, gamma.IsStarted())

	err = app.Shutdown()
	assert.NoError(t, err)
	assert.False(t, alpha.IsStarted())
	assert.False(t, beta.IsStarted())
	assert.True(t, gamma.IsStarted())

	err = gamma.Shutdown()
	assert.NoError(t, err)
	assert.False(t, gamma.IsStarted())
}

func TestApplication_Services(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.services.application")
	beta1 := connector.New("beta.services.application")
	beta2 := connector.New("beta.services.application")

	app := New(alpha, beta1, beta2)
	assert.Equal(t, []connector.Service{alpha, beta1, beta2}, app.Services())
	assert.Equal(t, []connector.Service{beta1, beta2}, app.ServicesByHost("beta.services.application"))
}

func TestApplication_Run(t *testing.T) {
	t.Parallel()

	con := connector.New("run.application")
	config := configurator.NewService()
	app := New(con, config)

	go func() {
		err := app.Run()
		assert.NoError(t, err)
	}()

	time.Sleep(time.Second)
	assert.True(t, con.IsStarted())
	assert.True(t, config.IsStarted())

	app.Interrupt()

	time.Sleep(time.Second)
	assert.False(t, con.IsStarted())
	assert.False(t, config.IsStarted())
}
