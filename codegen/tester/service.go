/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package tester

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"

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
	if svc.Deployment() != connector.TESTING && svc.Deployment() != connector.LOCAL {
		return errors.Newf("not allowed in '%s' deployment", svc.Deployment())
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
func (svc *Service) PointDistance(ctx context.Context, p1 testerapi.XYCoord, p2 *testerapi.XYCoord) (d float64, err error) {
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
	return fmt.Sprintf("%v %v %v", named, path2, suffix), nil
}

/*
NonStringPathArguments tests path arguments that are not strings.
*/
func (svc *Service) NonStringPathArguments(ctx context.Context, named int, path2 bool, suffix float64) (joined string, err error) {
	return fmt.Sprintf("%v %v %v", named, path2, suffix), nil
}

/*
UnnamedFunctionPathArguments tests path arguments that are not named.
*/
func (svc *Service) UnnamedFunctionPathArguments(ctx context.Context, path1 string, path2 string, path3 string) (joined string, err error) {
	return fmt.Sprintf("%v %v %v", path1, path2, path3), nil
}

/*
UnnamedWebPathArguments tests path arguments that are not named.
*/
func (svc *Service) UnnamedWebPathArguments(w http.ResponseWriter, r *http.Request) (err error) {
	q := r.URL.Query()
	path1 := q.Get("path1")
	path2 := q.Get("path2")
	path3 := q.Get("path3")
	w.Write([]byte(fmt.Sprintf("%v %v %v", path1, path2, path3)))
	return nil
}

/*
SumTwoIntegers tests returning a status code from a function.
*/
func (svc *Service) SumTwoIntegers(ctx context.Context, x int, y int) (sum int, httpStatusCode int, err error) {
	if x+y > 0 {
		return x + y, http.StatusAccepted, nil
	} else {
		return x + y, http.StatusNotAcceptable, nil
	}
}

/*
Echo tests a typical web handler.
*/
func (svc *Service) Echo(w http.ResponseWriter, r *http.Request) (err error) {
	// Verify that the frame in the context points to the header of the request
	frameHdr := frame.Of(r.Context()).Header()
	frameHdr.Add("Magic-Header", "Harry Potter")
	for k, vv1 := range frameHdr {
		vv2, ok := r.Header[k]
		if !ok || len(vv1) != len(vv2) {
			return errors.New("headers not same")
		}
		for i := 0; i < len(vv1); i++ {
			if vv1[i] != vv2[i] {
				return errors.New("headers not same")
			}
		}
	}
	r.Write(w)
	return nil
}
