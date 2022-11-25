package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microbus-io/fabric/services/metrics/metricsapi"
)

var (
	_ *testing.T
	_ assert.TestingT
	_ *metricsapi.Client
)

// Initialize starts up the testing app.
func Initialize() error {
    // TODO: Initialize testing app
	
	// Include all downstream microservices in the testing app
	// Use .With(...) to initialize with appropriate config values
	App.Include(
		Svc,
		// downstream.NewService().With(),
	)

	err := App.Startup()
	if err != nil {
		return err
	}

	// You may call any of the microservices after the app is started

	return nil
}

// Terminate shuts down the testing app.
func Terminate() error {
	err := App.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func TestMetrics_Collect(t *testing.T) {
	// TODO: Test Collect
	t.Parallel()
	/*
		Collect(ctx, POST(body), ContentType(mime), QueryArg(n, v), Header(n, v)).
			Name(testName).
			StatusOK(t).
			StatusCode(t, statusCode).
			BodyContains(t, bodyContains).
			BodyNotContains(t, bodyNotContains).
			HeaderContains(t, headerName, valueContains).
			NoError(t).
			Error(t, errContains).
			Assert(t, func(t, httpResponse, err))
	*/
	// ctx := Context()
}
