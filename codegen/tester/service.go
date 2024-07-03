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

package tester

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
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
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.Newf("service disallowed in '%s' deployment", svc.Deployment())
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
StringCut tests a function that takes primitive input arguments and returns primitive values.
*/
func (svc *Service) StringCut(ctx context.Context, s string, sep string) (before string, after string, found bool, err error) {
	before, after, found = strings.Cut(s, sep)
	return before, after, found, nil
}

/*
PointDistance tests a function that takes non-primitive input arguments.
*/
func (svc *Service) PointDistance(ctx context.Context, p1 testerapi.XYCoord, p2 *testerapi.XYCoord) (d float64, err error) {
	dx := (p1.X - p2.X)
	dy := (p1.Y - p2.Y)
	d = math.Sqrt(dx*dx + dy*dy)
	return d, nil
}

/*
ShiftPoint tests passing pointers of non-primitive types.
*/
func (svc *Service) ShiftPoint(ctx context.Context, p *testerapi.XYCoord, x float64, y float64) (shifted *testerapi.XYCoord, err error) {
	return &testerapi.XYCoord{X: p.X + x, Y: p.Y + y}, nil
}

/*
SubArrayRange tests sending arguments as the entire request and response bodies.
An httpRequestBody argument allows sending other arguments via query or path.
An httpResponseBody argument prevents returning additional values, except for the status code.
*/
func (svc *Service) SubArrayRange(ctx context.Context, httpRequestBody []int, min int, max int) (httpResponseBody []int, httpStatusCode int, err error) {
	for _, i := range httpRequestBody {
		if i >= min && i <= max {
			httpResponseBody = append(httpResponseBody, i)
		}
	}
	return httpResponseBody, http.StatusAccepted, nil
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
	parts := strings.Split(r.URL.Path, "/")
	path1 := parts[2]
	path2 := parts[4]
	path3 := strings.Join(parts[6:], "/")
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

/*
MultiValueHeaders tests a passing in and returning headers with multiple values.
*/
func (svc *Service) MultiValueHeaders(w http.ResponseWriter, r *http.Request) (err error) {
	if len(r.Header["Multi-In"]) <= 1 {
		return errors.New("multi value not received")
	}
	w.Header().Add("Multi-Out", "Out1")
	w.Header().Add("Multi-Out", "Out2")
	return nil
}

/*
PathArgumentsPriority tests the priority of path arguments in functions.
*/
func (svc *Service) PathArgumentsPriority(ctx context.Context, foo string) (echo string, err error) {
	return foo, nil
}

/*
DirectoryServer tests service resources given a greedy path argument.
*/
func (svc *Service) DirectoryServer(w http.ResponseWriter, r *http.Request) (err error) {
	_, path, _ := strings.Cut(r.URL.Path, "/directory-server/")
	path, _ = url.JoinPath("static", path)
	if !strings.HasPrefix(path, "static/") {
		return errors.Newc(http.StatusNotFound, "")
	}
	err = svc.ServeResFile(path, w, r)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
LinesIntersection tests nested non-primitive types.
*/
func (svc *Service) LinesIntersection(ctx context.Context, l1 testerapi.XYLine, l2 *testerapi.XYLine) (b bool, err error) {
	onSegment := func(p testerapi.XYCoord, q testerapi.XYCoord, r testerapi.XYCoord) bool {
		return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
			q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
	}
	orientation := func(p testerapi.XYCoord, q testerapi.XYCoord, r testerapi.XYCoord) int {
		val := (q.Y-p.Y)*(r.X-q.X) -
			(q.X-p.X)*(r.Y-q.Y)
		switch {
		case val == 0:
			return 0 // collinear
		case val > 0:
			return 1 // Clockwise
		default:
			return 2 // Counterclockwise
		}
	}

	// Find the four orientations needed for general and special cases
	o1 := orientation(l1.Start, l1.End, l2.Start)
	o2 := orientation(l1.Start, l1.End, l2.End)
	o3 := orientation(l2.Start, l2.End, l1.Start)
	o4 := orientation(l2.Start, l2.End, l1.End)

	// General case
	if o1 != o2 && o3 != o4 {
		return true, nil
	}

	// Special Cases
	// l1.Start, l1.End and l2.Start are collinear and l2.Start lies on segment l1
	if o1 == 0 && onSegment(l1.Start, l2.Start, l1.End) {
		return true, nil
	}
	// l1.Start, l1.End and l2.End are collinear and l2.End lies on segment l1
	if o2 == 0 && onSegment(l1.Start, l2.End, l1.End) {
		return true, nil
	}
	// l2.Start, l2.End and l1.Start are collinear and l1.Start lies on segment l2
	if o3 == 0 && onSegment(l2.Start, l1.Start, l2.End) {
		return true, nil
	}
	// l2.Start, l2.End and l1.End are collinear and l1.End lies on segment l2
	if o4 == 0 && onSegment(l2.Start, l1.End, l2.End) {
		return true, nil
	}
	return false, nil
}

/*
OnDiscovered tests listening to events.
*/
func (svc *Service) OnDiscoveredSink(ctx context.Context, p testerapi.XYCoord, n int) (q testerapi.XYCoord, m int, err error) {
	if n > 0 {
		return p, n + 1, nil
	}
	if n < 0 {
		return testerapi.XYCoord{X: -p.X, Y: -p.Y}, n + 1, nil
	}
	return testerapi.XYCoord{}, 0, errors.New("n can't be zero")
}

/*
Hello prints hello in the language best matching the request's Accept-Language header.
*/
func (svc *Service) Hello(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	hello, _ := svc.LoadResString(ctx, "hello")
	w.Write([]byte(hello))
	return nil
}

/*
WhatTimeIsIt tests shifting the clock.
*/
func (svc *Service) WhatTimeIsIt(ctx context.Context) (t time.Time, err error) {
	return svc.Now(ctx), nil
}
