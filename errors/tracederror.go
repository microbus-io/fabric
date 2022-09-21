package errors

import (
	"encoding/json"
	stderrors "errors"
	"runtime"
	"strings"
)

type TracedError interface {
	error

	Stack() []Trace
	Trace(trace Trace) TracedError
	String() string

	json.Marshaler
	json.Unmarshaler
}

type TracedErrorImpl struct {
	error
	stack []Trace
}

// Stack returns the current stack trace.
func (e *TracedErrorImpl) Stack() []Trace {
	return e.stack
}

// Trace adds a trace to the stack trace.
func (e *TracedErrorImpl) Trace(trace Trace) TracedError {
	e.stack = append(e.stack, trace)
	return e
}

// MarshalJSON returns a JSON encoding of a TracedError.
func (e *TracedErrorImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error string  `json:"error"`
		Stack []Trace `json:"stack"`
	}{
		Error: e.error.Error(),
		Stack: e.stack,
	})
}

// UnmarshalJSON converts a TracedError from a JSON encoding.
func (e *TracedErrorImpl) UnmarshalJSON(data []byte) error {
	jsonStruct := &struct {
		Error string  `json:"error"`
		Stack []Trace `json:"stack"`
	}{}
	err := json.Unmarshal(data, &jsonStruct)
	if err != nil {
		return err
	}
	e.error = stderrors.New(jsonStruct.Error)
	e.stack = jsonStruct.Stack
	return nil
}

// String returns a string representation of the current stack trace of TracedError.
// Traces written to the string follow the last in first out (LIFO) order.
func (e *TracedErrorImpl) String() string {
	var b strings.Builder
	stack := e.Stack()
	for i := range stack {
		trace := stack[len(stack)-i-1]
		b.WriteString(trace.String())
	}
	return b.String()
}

// New creates a new TracedError.
func New(text string, annotations ...string) TracedError {
	tracedErr := &TracedErrorImpl{
		error: stderrors.New(text),
		stack: []Trace{},
	}

	file, function, line := runtimeTrace(1)

	return tracedErr.Trace(Trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
}

// TraceError adds a trace to an existing stack trace of TracedError.
// If the error is not a TracedError, it is converted and starts a new stack trace
// and establishes the current error as the root of the stack trace.
func TraceError(err error, annotations ...string) TracedError {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err)

	file, function, line := runtimeTrace(1)

	return tracedErr.Trace(Trace{
		File:        file,
		Function:    function,
		Line:        line,
		Annotations: annotations,
	})
}

// Convert converts a Go error into a TracedError without stack tracing.
// If the error is already a TracedError, it is returned as is.
func Convert(err error) TracedError {
	if err == nil {
		return nil
	}

	var tracedErr *TracedErrorImpl
	if stderrors.As(err, &tracedErr) {
		return tracedErr
	}

	return &TracedErrorImpl{
		error: err,
		stack: []Trace{},
	}
}

// runtimeTrace traces back by the amount of levels
// to retrieve the runtime information used for tracing.
func runtimeTrace(levels int) (file string, function string, line int) {
	pc, file, line, ok := runtime.Caller(levels + 1)
	function = "?"
	runtimeFunc := runtime.FuncForPC(pc)
	if ok && runtimeFunc != nil {
		function = runtimeFunc.Name()
		p := strings.LastIndex(function, "/")
		if p >= 0 {
			function = function[p+1:]
		}
	}
	return file, function, line
}
