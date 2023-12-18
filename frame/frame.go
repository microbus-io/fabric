/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package frame

import (
	"context"
	"net/http"
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
	HeaderTimestamp     = HeaderPrefix + "Timestamp"
	HeaderQueue         = HeaderPrefix + "Queue"
	HeaderFragment      = HeaderPrefix + "Fragment"

	OpCodeError    = "Err"
	OpCodeAck      = "Ack"
	OpCodeRequest  = "Req"
	OpCodeResponse = "Res"
)

type contextKeyType struct{}

// ContextKey is used to store the request headers in a context.
var ContextKey = contextKeyType{}

// Frame is a utility class that helps with manipulating the control headers.
type Frame struct {
	h http.Header
}

// Of creates a new frame for the headers of the HTTP request, response or response writer.
func Of(r any) Frame {
	var h http.Header
	switch v := r.(type) {
	case *http.Request:
		h = v.Header
	case *http.Response:
		h = v.Header
	case http.ResponseWriter:
		h = v.Header()
	case http.Header:
		h = v
	case context.Context:
		h, _ = v.Value(ContextKey).(http.Header)
		if h == nil {
			h = make(http.Header)
		}
	}
	return Frame{h}
}

// Clone returns a new context with the frame of the original context.
// Manipulating the frame of the result context does not impact the original context.
// It is the equivalent of Copy(ctx, ctx)
func Clone(ctx context.Context) (result context.Context) {
	return Copy(ctx, ctx)
}

// Copy takes the frame of the source context and adds it to the destination context.
// The result is a new context derived from the destination, but with the frame of the source.
// Manipulating the frame of the new context does not impact the source or destination contexts.
func Copy(dest context.Context, src context.Context) (result context.Context) {
	f := Of(src)
	h := make(http.Header)
	for k, v := range f.h {
		h[k] = v
	}
	return context.WithValue(dest, ContextKey, h)
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

// XForwarded returns the amalgamated URL from the X-Forwarded headers without a trailing slash.
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

// FromHost is the host name of the microservice that made the request or response.
func (f Frame) FromHost() string {
	return f.h.Get(HeaderFromHost)
}

// SetFromHost sets the host name of the microservice that is making the request or response.
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
