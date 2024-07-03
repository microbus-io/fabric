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

package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"strings"
)

var (
	_ = error(&TracedError{})
	_ = fmt.Stringer(&TracedError{})
	_ = fmt.Formatter(&TracedError{})
	_ = json.Marshaler(&TracedError{})
	_ = json.Unmarshaler(&TracedError{})
)

// TracedError is a standard Go error augmented with a stack trace.
type TracedError struct {
	error
	stack      []*trace
	StatusCode int
}

// Unwrap returns the underlying error.
func (e *TracedError) Unwrap() error {
	return e.error
}

// String returns a string representation of the error.
func (e *TracedError) String() string {
	var b strings.Builder
	b.WriteString(e.Error())
	if e.StatusCode != 0 && e.StatusCode != 500 {
		b.WriteString(fmt.Sprintf("\n[%d]", e.StatusCode))
	}
	for i, trace := range e.stack {
		if i == 0 {
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(trace.String())
	}
	return b.String()
}

// MarshalJSON marshals the error to JSON.
func (e *TracedError) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Error      string   `json:"error,omitempty"`
		Stack      []*trace `json:"stack,omitempty"`
		StatusCode int      `json:"statusCode,omitempty"`
	}{
		Error:      e.error.Error(),
		Stack:      e.stack,
		StatusCode: e.StatusCode,
	})
}

// UnmarshalJSON unmarshals the erro from JSON.
func (e *TracedError) UnmarshalJSON(data []byte) error {
	j := &struct {
		Error      string   `json:"error"`
		Stack      []*trace `json:"stack"`
		StatusCode int      `json:"statusCode,omitempty"`
	}{}
	err := json.Unmarshal(data, &j)
	if err != nil {
		return err
	}
	e.error = stderrors.New(j.Error)
	e.stack = j.Stack
	e.StatusCode = j.StatusCode
	return nil
}

// Format the error based on the verb and flag.
func (e *TracedError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') || s.Flag('#') {
			io.WriteString(s, e.String())
		} else {
			io.WriteString(s, e.Error())
		}
	case 's':
		io.WriteString(s, e.Error())
	}
}
