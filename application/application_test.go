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

package application

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/coreservices/configurator"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/testarossa"
)

func TestApplication_StartStop(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.start.stop.application")
	beta := connector.New("beta.start.stop.application")
	app := NewTesting()
	app.Add(alpha, beta)

	testarossa.False(t, alpha.IsStarted())
	testarossa.False(t, beta.IsStarted())

	err := app.Startup()
	testarossa.NoError(t, err)

	testarossa.True(t, alpha.IsStarted())
	testarossa.True(t, beta.IsStarted())

	err = app.Shutdown()
	testarossa.NoError(t, err)

	testarossa.False(t, alpha.IsStarted())
	testarossa.False(t, beta.IsStarted())
}

func TestApplication_Interrupt(t *testing.T) {
	t.Parallel()

	con := connector.New("interrupt.application")
	app := NewTesting()
	app.Add(con)

	ch := make(chan bool)
	go func() {
		err := app.Startup()
		testarossa.NoError(t, err)
		go func() {
			app.WaitForInterrupt()
			err := app.Shutdown()
			testarossa.NoError(t, err)
			ch <- true
		}()
		ch <- true
	}()

	<-ch
	testarossa.True(t, con.IsStarted())
	app.Interrupt()
	<-ch
	testarossa.False(t, con.IsStarted())
}

func TestApplication_NoConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create first testing app
	alpha := connector.New("no.conflict.application")
	alpha.Subscribe("GET", "id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("alpha"))
		return nil
	})
	appAlpha := NewTesting()
	appAlpha.Add(alpha)

	// Create second testing app
	beta := connector.New("no.conflict.application")
	beta.Subscribe("GET", "id", func(w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte("beta"))
		return nil
	})
	appBeta := NewTesting()
	appBeta.Add(beta)

	// Start the apps
	err := appAlpha.Startup()
	testarossa.NoError(t, err)
	defer appAlpha.Shutdown()
	err = appBeta.Startup()
	testarossa.NoError(t, err)
	defer appBeta.Shutdown()

	// Assert different planes of communication
	testarossa.NotEqual(t, alpha.Plane(), beta.Plane())
	testarossa.Equal(t, connector.TESTING, alpha.Deployment())
	testarossa.Equal(t, connector.TESTING, beta.Deployment())

	// Alpha should never see beta
	for i := 0; i < 32; i++ {
		response, err := alpha.GET(ctx, "https://no.conflict.application/id")
		testarossa.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, "alpha", string(body))
	}

	// Beta should never see alpha
	for i := 0; i < 32; i++ {
		response, err := beta.GET(ctx, "https://no.conflict.application/id")
		testarossa.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		testarossa.NoError(t, err)
		testarossa.Equal(t, "beta", string(body))
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
	beta.Subscribe("GET", "/ok", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	beta.SetOnStartup(func(ctx context.Context) error {
		time.Sleep(startupTimeout / 2)
		return nil
	})

	app := NewTesting()
	app.Add(alpha, beta)
	app.startupTimeout = startupTimeout
	t0 := time.Now()
	err := app.Startup()
	dur := time.Since(t0)
	testarossa.NoError(t, err)
	testarossa.True(t, failCount > 0)
	testarossa.True(t, dur >= startupTimeout/2)

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

	app := NewTesting()
	app.Add(alpha, beta)
	app.startupTimeout = startupTimeout
	t0 := time.Now()
	err := app.Startup()
	dur := time.Since(t0)
	testarossa.Error(t, err)
	testarossa.True(t, failCount > 0)
	testarossa.True(t, dur >= startupTimeout)
	testarossa.True(t, beta.IsStarted())
	testarossa.False(t, alpha.IsStarted())

	err = app.Shutdown()
	testarossa.NoError(t, err)
	testarossa.False(t, beta.IsStarted())
	testarossa.False(t, alpha.IsStarted())
}

func TestApplication_Remove(t *testing.T) {
	t.Parallel()

	alpha := connector.New("alpha.remove.application")
	beta := connector.New("beta.remove.application")

	app := NewTesting()
	app.AddAndStartup(alpha, beta)
	testarossa.True(t, alpha.IsStarted())
	testarossa.True(t, beta.IsStarted())
	testarossa.Equal(t, alpha.Plane(), beta.Plane())

	app.Remove(beta)
	testarossa.True(t, alpha.IsStarted())
	testarossa.True(t, beta.IsStarted())
	testarossa.Equal(t, alpha.Plane(), beta.Plane())

	err := app.Shutdown()
	testarossa.NoError(t, err)
	testarossa.False(t, alpha.IsStarted())
	testarossa.True(t, beta.IsStarted()) // Should remain up because no longer under management of the app

	err = beta.Shutdown()
	testarossa.NoError(t, err)
	testarossa.False(t, beta.IsStarted())
}

func TestApplication_Run(t *testing.T) {
	t.Parallel()

	con := connector.New("run.application")
	config := configurator.NewService()
	app := NewTesting()
	app.Add(config)
	app.Add(con)

	go func() {
		err := app.Run()
		testarossa.NoError(t, err)
	}()

	time.Sleep(2 * time.Second)
	testarossa.True(t, con.IsStarted())
	testarossa.True(t, config.IsStarted())

	app.Interrupt()

	time.Sleep(time.Second)
	testarossa.False(t, con.IsStarted())
	testarossa.False(t, config.IsStarted())
}
