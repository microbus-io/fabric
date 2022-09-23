package errors

import (
	"encoding/json"
	stderrors "errors"
	"strings"
)

type tracedError struct {
	error
	stack []trace
}

// stacktrace returns the current stack trace.
func (e *tracedError) Stack() []trace {
	return e.stack
}

// push adds a trace to the stack trace.
func (e *tracedError) Push(trace trace) {
	e.stack = append(e.stack, trace)
}

// String returns a string representation of the current stack trace of the traced error.
// Traces written to the string follow the last in first out (LIFO) order.
func (e *tracedError) String() string {
	var b strings.Builder
	b.WriteString("\n")
	stack := e.Stack()
	for i := range stack {
		trace := stack[len(stack)-i-1]
		b.WriteString(trace.String())
	}
	b.WriteString("error: ")
	b.WriteString(e.Error())
	return b.String()
}

// MarshalJSON returns a JSON encoding of a traced error.
func (e *tracedError) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error string  `json:"error"`
		Stack []trace `json:"stack"`
	}{
		Error: e.error.Error(),
		Stack: e.stack,
	})
}

// UnmarshalJSON converts a traced error from a JSON encoding.
func (e *tracedError) UnmarshalJSON(data []byte) error {
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
