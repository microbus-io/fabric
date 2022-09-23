package errors

import (
	"encoding/json"
	stderrors "errors"
	"strings"
)

type TracedError interface {
	error
	String() string
	json.Marshaler
	json.Unmarshaler
}

type tracedErrorImpl struct {
	error
	stack []trace
}

// Stack returns the current stack trace.
func (e *tracedErrorImpl) Stack() []trace {
	return e.stack
}

// Push adds a trace to the stack trace.
func (e *tracedErrorImpl) Push(trace trace) TracedError {
	e.stack = append(e.stack, trace)
	return e
}

// String returns a string representation of the current stack trace of TracedError.
// Traces written to the string follow the last in first out (LIFO) order.
func (e *tracedErrorImpl) String() string {
	var b strings.Builder
	b.WriteString("\n")
	stack := e.Stack()
	for i := range stack {
		trace := stack[len(stack)-i-1]
		b.WriteString(trace.String())
	}
	return b.String()
}

// MarshalJSON returns a JSON encoding of a TracedError.
func (e *tracedErrorImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error string  `json:"error"`
		Stack []trace `json:"stack"`
	}{
		Error: e.error.Error(),
		Stack: e.stack,
	})
}

// UnmarshalJSON converts a TracedError from a JSON encoding.
func (e *tracedErrorImpl) UnmarshalJSON(data []byte) error {
	jsonStruct := &struct {
		Error string  `json:"error"`
		Stack []trace `json:"stack"`
	}{}
	err := json.Unmarshal(data, &jsonStruct)
	if err != nil {
		return err
	}
	e.error = stderrors.New(jsonStruct.Error)
	e.stack = jsonStruct.Stack
	return nil
}
