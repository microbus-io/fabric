package frame

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

const (
	HeaderPrefix     = "Microbus-"
	HeaderMsgId      = HeaderPrefix + "Msg-Id"
	HeaderFromHost   = HeaderPrefix + "From-Host"
	HeaderFromId     = HeaderPrefix + "From-Id"
	HeaderTimeBudget = HeaderPrefix + "Time-Budget"
	HeaderCallDepth  = HeaderPrefix + "Call-Depth"
	HeaderOpCode     = HeaderPrefix + "Op-Code"
	HeaderTimestamp  = HeaderPrefix + "Timestamp"
	HeaderQueue      = HeaderPrefix + "Queue"

	OpCodeError    = "Err"
	OpCodeAck      = "Ack"
	OpCodeRequest  = "Req"
	OpCodeResponse = "Res"
)

type contextKeyType struct{}

// ContextKey is used to store the request headers in a context
var ContextKey = contextKeyType{}

// Frame is a utility class that helps with manipulating the control headers
type Frame struct {
	h http.Header
}

// Of creates a new frame for the headers of the HTTP request, response or response writer
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

// OpCode indicates the type of the control message
func (f Frame) OpCode() string {
	return f.h.Get(HeaderOpCode)
}

// SetOpCode sets the type of the control message
func (f Frame) SetOpCode(op string) {
	if op == "" {
		f.h.Del(HeaderOpCode)
	} else {
		f.h.Set(HeaderOpCode, op)
	}
}

// FromHost is the host name of the microservice that made the request or response
func (f Frame) FromHost() string {
	return f.h.Get(HeaderFromHost)
}

// SetFromHost sets the host name of the microservice that is making the request or response
func (f Frame) SetFromHost(host string) {
	if host == "" {
		f.h.Del(HeaderFromHost)
	} else {
		f.h.Set(HeaderFromHost, host)
	}
}

// FromID is the unique ID of the instance of the microservice that made the request or response
func (f Frame) FromID() string {
	return f.h.Get(HeaderFromId)
}

// SetFromID sets the unique ID of the instance of the microservice that is making the request or response
func (f Frame) SetFromID(id string) {
	if id == "" {
		f.h.Del(HeaderFromId)
	} else {
		f.h.Set(HeaderFromId, id)
	}
}

// MessageID is the unique ID given to each HTTP message and its response
func (f Frame) MessageID() string {
	return f.h.Get(HeaderMsgId)
}

// SetMessageID sets the unique ID given to each HTTP message or response
func (f Frame) SetMessageID(id string) {
	if id == "" {
		f.h.Del(HeaderMsgId)
	} else {
		f.h.Set(HeaderMsgId, id)
	}
}

// CallDepth is the depth of the call stack beginning at the original request
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

// SetCallDepth sets the depth of the call stack beginning at the original request
func (f Frame) SetCallDepth(depth int) {
	if depth == 0 {
		f.h.Del(HeaderCallDepth)
	} else {
		f.h.Set(HeaderCallDepth, strconv.Itoa(depth))
	}
}

// TimeBudget is the duration budgeted for the request to complete
func (f Frame) TimeBudget() time.Duration {
	v := f.h.Get(HeaderTimeBudget)
	if v == "" {
		return 0
	}
	ms, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return time.Millisecond * time.Duration(ms)
}

// SetTimeBudget budgets a duration for the request to complete
func (f Frame) SetTimeBudget(budget time.Duration) {
	ms := int(budget.Milliseconds())
	if ms <= 0 {
		f.h.Del(HeaderTimeBudget)
	} else {
		f.h.Set(HeaderTimeBudget, strconv.Itoa(ms))
	}
}

// Queue indicates the queue of the subscription that handled the request.
// It is used to optimize pub/sub requests
func (f Frame) Queue() string {
	return f.h.Get(HeaderQueue)
}

// SetQueue sets the queue of the subscription that handled the request.
// It is used to optimize pub/sub requests
func (f Frame) SetQueue(queue string) {
	if queue == "" {
		f.h.Del(HeaderQueue)
	} else {
		f.h.Set(HeaderQueue, queue)
	}
}
