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

// MarshalJSON returns JSON marshal of TracedError.
func (e *TracedErrorImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error string  `json:"error"`
		Stack []Trace `json:"stack"`
	}{
		Error: e.error.Error(),
		Stack: e.stack,
	})
}

// UnmarshalJSON returns the TracedError from a JSON marshal.
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
func (e *TracedErrorImpl) String() string {
	var b strings.Builder
	for _, t := range e.Stack() {
		b.WriteString("\n")
		b.WriteString(t.String())
	}
	return b.String()
}

// New creates a new TracedError.
func New(text string, annotation string) TracedError {
	tracedErr := &TracedErrorImpl{
		error: stderrors.New(text),
		stack: []Trace{},
	}

	file, function, line := runtimeTrace(1)

	return tracedErr.Trace(Trace{
		File:       file,
		Function:   function,
		Line:       line,
		Annotation: annotation,
	})
}

// TraceError adds to existing stack trace of TracedError
// or starts a new TracedError if needed.
func TraceError(err error, annotation string) TracedError {
	if err == nil {
		return nil
	}
	tracedErr := Convert(err)

	level := 1
	file, function, line := runtimeTrace(level)
	for strings.HasPrefix(function, "runtime.") {
		level++
		file, function, line = runtimeTrace(level)
	}

	return tracedErr.Trace(Trace{
		File:       file,
		Function:   function,
		Line:       line,
		Annotation: annotation,
	})
}

// Convert converts an error into a TracedError.
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
// to retrieve the runtime information for tracing.
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
