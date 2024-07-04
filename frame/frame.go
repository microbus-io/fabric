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

package frame

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderPrefix        = "Microbus-"
	HeaderBaggagePrefix = HeaderPrefix + "Baggage-"
	HeaderMsgId         = HeaderPrefix + "Msg-Id"
	HeaderFromHost      = HeaderPrefix + "From-Host"
	HeaderFromId        = HeaderPrefix + "From-Id"
	HeaderFromVersion   = HeaderPrefix + "From-Version"
	HeaderTimeBudget    = HeaderPrefix + "Time-Budget"
	HeaderCallDepth     = HeaderPrefix + "Call-Depth"
	HeaderOpCode        = HeaderPrefix + "Op-Code"
	HeaderQueue         = HeaderPrefix + "Queue"
	HeaderFragment      = HeaderPrefix + "Fragment"
	HeaderClockShift    = HeaderPrefix + "Clock-Shift"
	HeaderLocality      = HeaderPrefix + "Locality"

	OpCodeError    = "Err"
	OpCodeAck      = "Ack"
	OpCodeRequest  = "Req"
	OpCodeResponse = "Res"
)

type contextKeyType struct{}

// contextKey is used to store the request headers in a context.
var contextKey = contextKeyType{}

// Frame is a utility class that helps with manipulating the control headers.
type Frame struct {
	h http.Header
}

// Of creates a new frame wrapping the headers of the HTTP request, response, response writer, header, or context.
func Of(x any) Frame {
	var h http.Header
	if x != nil {
		switch v := x.(type) {
		case Frame:
			h = v.h
		case *http.Request:
			h = v.Header
		case *http.Response:
			h = v.Header
		case http.ResponseWriter:
			h = v.Header()
		case http.Header:
			h = v
		case context.Context:
			h, _ = v.Value(contextKey).(http.Header)
		}
	}
	// If h==nil, frame will be read-only, returning empty values
	return Frame{h}
}

// CloneContext returns a new context with a copy of the frame of the parent context, or a new frame if it does not have one.
// Manipulating the frame of the cloned context does not impact the parent's.
func CloneContext(parent context.Context) (cloned context.Context) {
	h := http.Header{}
	for k, vv := range Of(parent).h {
		h[k] = append(h[k], vv...)
	}
	return context.WithValue(parent, contextKey, h)
}

// ContextWithFrameOf returns a new context derived from the parent, referencing the frame of x.
// If the parent includes a frame, it will be superseded by the given frame.
// Manipulating the frame of the new context impacts the original frame.
func ContextWithFrameOf(parent context.Context, x any) (ctx context.Context) {
	return context.WithValue(parent, contextKey, Of(x).h)
}

// ContextWithFrame adds a new empty frame to the context.
// If the parent includes a frame, it will be superseded by the given frame.
func ContextWithFrame(parent context.Context) (ctx context.Context) {
	return context.WithValue(parent, contextKey, http.Header{})
}

// Get returns an arbitrary header.
func (f Frame) Get(name string) string {
	return f.h.Get(name)
}

// Set sets the value of an arbitrary header.
func (f Frame) Set(name string, value string) {
	if value == "" {
		f.h.Del(name)
	} else {
		f.h.Set(name, value)
	}
}

// Header returns the underlying HTTP header backing the frame.
func (f Frame) Header() http.Header {
	return f.h
}

// XForwardedBaseURL returns the amalgamated headers X-Forwarded-Proto, -Host and -Prefix, without a trailing slash.
// The empty string is returned if the headers are not present.
func (f Frame) XForwardedBaseURL() string {
	proto := f.Header().Get("X-Forwarded-Proto")
	host := f.Header().Get("X-Forwarded-Host")
	prefix := f.Header().Get("X-Forwarded-Prefix")
	if proto == "" || host == "" {
		return ""
	}
	return strings.TrimRight(proto+"://"+host+prefix, "/")
}

// XForwardedFullURL returns the amalgamated headers X-Forwarded-Proto, -Host, -Prefix and -Path, without a trailing slash.
// The empty string is returned if the headers are not present.
func (f Frame) XForwardedFullURL() string {
	proto := f.Header().Get("X-Forwarded-Proto")
	host := f.Header().Get("X-Forwarded-Host")
	prefix := f.Header().Get("X-Forwarded-Prefix")
	path := f.Header().Get("X-Forwarded-Path")
	if proto == "" || host == "" {
		return ""
	}
	return strings.TrimRight(proto+"://"+host+prefix+path, "/")
}

// OpCode indicates the type of the control message.
func (f Frame) OpCode() string {
	return f.h.Get(HeaderOpCode)
}

// SetOpCode sets the type of the control message.
func (f Frame) SetOpCode(op string) {
	if op == "" {
		f.h.Del(HeaderOpCode)
	} else {
		f.h.Set(HeaderOpCode, op)
	}
}

// FromHost is the hostname of the microservice that made the request or response.
func (f Frame) FromHost() string {
	return f.h.Get(HeaderFromHost)
}

// SetFromHost sets the hostname of the microservice that is making the request or response.
func (f Frame) SetFromHost(host string) {
	if host == "" {
		f.h.Del(HeaderFromHost)
	} else {
		f.h.Set(HeaderFromHost, host)
	}
}

// FromID is the unique ID of the instance of the microservice that made the request or response.
func (f Frame) FromID() string {
	return f.h.Get(HeaderFromId)
}

// SetFromID sets the unique ID of the instance of the microservice that is making the request or response.
func (f Frame) SetFromID(id string) {
	if id == "" {
		f.h.Del(HeaderFromId)
	} else {
		f.h.Set(HeaderFromId, id)
	}
}

// FromVersion is the version number of the microservice that made the request or response.
func (f Frame) FromVersion() int {
	v := f.h.Get(HeaderFromVersion)
	if v == "" {
		return 0
	}
	ver, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return ver
}

// SetFromVersion sets the version number of the microservice that is making the request or response.
func (f Frame) SetFromVersion(version int) {
	if version == 0 {
		f.h.Del(HeaderFromVersion)
	} else {
		f.h.Set(HeaderFromVersion, strconv.Itoa(version))
	}
}

// MessageID is the unique ID given to each HTTP message and its response.
func (f Frame) MessageID() string {
	return f.h.Get(HeaderMsgId)
}

// SetMessageID sets the unique ID given to each HTTP message or response.
func (f Frame) SetMessageID(id string) {
	if id == "" {
		f.h.Del(HeaderMsgId)
	} else {
		f.h.Set(HeaderMsgId, id)
	}
}

// CallDepth is the depth of the call stack beginning at the original request.
func (f Frame) CallDepth() int {
	v := f.h.Get(HeaderCallDepth)
	if v == "" {
		return 0
	}
	depth, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return depth
}

// SetCallDepth sets the depth of the call stack beginning at the original request.
func (f Frame) SetCallDepth(depth int) {
	if depth == 0 {
		f.h.Del(HeaderCallDepth)
	} else {
		f.h.Set(HeaderCallDepth, strconv.Itoa(depth))
	}
}

// TimeBudget is the duration budgeted for the request to complete.
// A value of 0 indicates no time budget.
func (f Frame) TimeBudget() time.Duration {
	v := f.h.Get(HeaderTimeBudget)
	if v == "" {
		return 0
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms < 0 {
		return 0
	}
	return time.Millisecond * time.Duration(ms)
}

// SetTimeBudget budgets a duration for the request to complete.
// A value of 0 indicates no time budget.
func (f Frame) SetTimeBudget(budget time.Duration) {
	ms := int(budget.Milliseconds())
	if ms <= 0 {
		f.h.Del(HeaderTimeBudget)
	} else {
		f.h.Set(HeaderTimeBudget, strconv.Itoa(ms))
	}
}

// Queue indicates the queue of the subscription that handled the request.
// It is used by the client to optimize multicast requests.
func (f Frame) Queue() string {
	return f.h.Get(HeaderQueue)
}

// SetQueue sets the queue of the subscription that handled the request.
// It is used by the client to optimize multicast requests.
func (f Frame) SetQueue(queue string) {
	if queue == "" {
		f.h.Del(HeaderQueue)
	} else {
		f.h.Set(HeaderQueue, queue)
	}
}

// Fragment returns the index of the fragment of large messages out of the total number of fragments.
// Fragments are indexed starting at 1.
func (f Frame) Fragment() (index int, max int) {
	v := f.h.Get(HeaderFragment)
	if v == "" {
		return 1, 1
	}
	parts := strings.Split(v, "/")
	if len(parts) != 2 {
		return 1, 1
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return 1, 1
	}
	max, err = strconv.Atoi(parts[1])
	if err != nil {
		return 1, 1
	}
	return index, max
}

// Fragment sets the index of the fragment of large messages out of the total number of fragments.
// Fragments are indexed starting at 1.
func (f Frame) SetFragment(index int, max int) {
	if index < 1 || max < 1 || (index == 1 && max == 1) {
		f.h.Del(HeaderFragment)
	} else {
		f.h.Set(HeaderFragment, strconv.Itoa(index)+"/"+strconv.Itoa(max))
	}
}

// ClockShift returns the time offset set in the frame.
// Time offsets are used during testing to offset the clock of a transaction.
// A positive offset moves the clock into the future.
// A negative offset moves the clock into the past.
func (f Frame) ClockShift() time.Duration {
	s := f.h.Get(HeaderClockShift)
	if s == "" {
		return 0
	}
	d, _ := time.ParseDuration(s)
	return d
}

// SetClockShift sets the time offset in the frame.
// Time offsets are used during testing to offset the clock of a transaction.
// A positive offset moves the clock into the future.
// A negative offset moves the clock into the past.
func (f Frame) SetClockShift(offset time.Duration) {
	if offset == 0 {
		f.h.Del(HeaderClockShift)
	} else {
		f.h.Set(HeaderClockShift, offset.String())
	}
}

// Baggage is an arbitrary header that is passed through to downstream microservices.
func (f Frame) Baggage(name string) string {
	return f.h.Get(HeaderBaggagePrefix + name)
}

// SetBaggage sets an arbitrary header that is passed through to downstream microservices.
func (f Frame) SetBaggage(name string, value string) {
	if value == "" {
		f.h.Del(HeaderBaggagePrefix + name)
	} else {
		f.h.Set(HeaderBaggagePrefix+name, value)
	}
}

// Languages parses the Accept-Language header and returns the listed languages in order of their q value.
func (f Frame) Languages() []string {
	qOrder := map[string]float64{}
	var result []string

	// da, en-gb;q=0.8, en;q=0.7
	full := f.h.Get("Accept-Language")
	segments := strings.Split(full, ",")
	for s, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		p := strings.Index(seg, ";")
		if p < 0 {
			// da
			result = append(result, seg)
			qOrder[seg] = 1.0 - float64(s)/1e6
		} else {
			// en-gb;q=0.8
			lang := strings.TrimSpace(seg[:p])
			if lang != "" {
				qStr := strings.TrimLeft(seg[p+1:], " q=")
				q, _ := strconv.ParseFloat(qStr, 64)
				result = append(result, lang)
				qOrder[lang] = q - float64(s)/1e6
			}
		}
	}
	if len(result) > 1 {
		sort.Slice(result, func(i, j int) bool {
			return qOrder[result[i]] > qOrder[result[j]]
		})
	}
	return result
}

// SetLanguages sets the Accept-Language header with the list of languages.
func (f Frame) SetLanguages(language ...string) {
	if len(language) == 0 {
		f.h.Del("Accept-Language")
	} else {
		f.h.Set("Accept-Language", strings.Join(language, ", "))
	}
}

// Locality indicates the geographic locality of the microservice that handled the request.
// It is used by the client to optimize routing of unicast requests.
func (f Frame) Locality() string {
	return f.h.Get(HeaderLocality)
}

// SetLocality sets the geographic locality of the microservice that handled the request.
// It is used by the client to optimize routing of unicast requests.
func (f Frame) SetLocality(locality string) {
	if locality == "" {
		f.h.Del(HeaderLocality)
	} else {
		f.h.Set(HeaderLocality, locality)
	}
}
