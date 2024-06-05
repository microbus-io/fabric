/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package tester

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/codegen/tester/intermediate"
	"github.com/microbus-io/fabric/codegen/tester/testerapi"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *testerapi.Client
)

/*
Service implements the codegen.test microservice.

The tester is used to test the code generator's functions.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.TESTING {
		return errors.Newf("restricted to '%s' deployment", connector.TESTING)
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
StringCut tests a standard function that takes multiple input arguments and returns multiple values.
*/
func (svc *Service) StringCut(ctx context.Context, s string, sep string) (before string, after string, found bool, err error) {
	before, after, found = strings.Cut(s, sep)
	return before, after, found, nil
}

/*
PointDistance tests passing non-primitive types via query arguments.
*/
func (svc *Service) PointDistance(ctx context.Context, p1 testerapi.XYCoord, p2 testerapi.XYCoord) (d float64, err error) {
	dx := (p1.X - p2.X)
	dy := (p1.Y - p2.Y)
	d = math.Sqrt(dx*dx + dy*dy)
	return d, nil
}

/*
SubArrayRange tests sending arguments as the entire request and response bodies.
An httpRequestBody argument allows sending other arguments via query or path.
An httpResponseBody argument prevents returning additional values, except for the status code.
*/
func (svc *Service) SubArrayRange(ctx context.Context, httpRequestBody []int, min int, max int) (httpResponseBody []int, sum int, httpStatusCode int, err error) {
	for _, i := range httpRequestBody {
		if i >= min && i <= max {
			httpResponseBody = append(httpResponseBody, i)
			sum += i // Will fail to return
		}
	}
	return httpResponseBody, sum, http.StatusAccepted, nil
}

/*
WebPathArguments tests path arguments in web handlers.
*/
func (svc *Service) WebPathArguments(w http.ResponseWriter, r *http.Request) (err error) {
	u := r.URL.String()
	u += "$" // Mark EOF
	w.Write([]byte(u))
	return nil
}

/*
FunctionPathArguments tests path arguments in functions.
*/
func (svc *Service) FunctionPathArguments(ctx context.Context, named string, path2 string, suffix string) (joined string, err error) {
	return named + " " + path2 + " " + suffix, nil
}
