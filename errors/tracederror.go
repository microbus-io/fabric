package errors

import (
	"encoding/json"
	stderrors "errors"
	"strings"
)

// TracedError is a standard Go error augmented with a stack trace and annotations
type TracedError struct {
	error
	stack []trace
}

// String returns a string representation of the error.
// The stack trace is writted in last in first out (LIFO) order
func (e *TracedError) String() string {
	var b strings.Builder
	b.WriteString(e.Error())
	b.WriteString("\n")
	for i := range e.stack {
		trace := e.stack[len(e.stack)-i-1]
		b.WriteString(trace.String())
		b.WriteString("\n")
	}
	return b.String()
}

// MarshalJSON marshals the error to JSON
func (e *TracedError) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error string  `json:"error,omitempty"`
		Stack []trace `json:"stack,omitempty"`
	}{
		Error: e.error.Error(),
		Stack: e.stack,
	})
}

// UnmarshalJSON unmarshals the erro from JSON
func (e *TracedError) UnmarshalJSON(data []byte) error {
	j := &struct {
		Error string  `json:"error"`
		Stack []trace `json:"stack"`
	}{}
	err := json.Unmarshal(data, &j)
	if err != nil {
		return err
	}
	e.error = stderrors.New(j.Error)
	e.stack = j.Stack
	return nil
}
