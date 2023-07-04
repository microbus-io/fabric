/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package errors

import (
	"strconv"
	"strings"
)

// trace is a single stack trace location
type trace struct {
	File        string   `json:"file"`
	Function    string   `json:"function"`
	Line        int      `json:"line"`
	Annotations []string `json:"annotations,omitempty"`
}

// String returns a string representation of the trace
func (t *trace) String() string {
	var b strings.Builder
	b.WriteString(t.Function)
	b.WriteString("\n\t")
	b.WriteString(t.File)
	b.WriteString(":")
	b.WriteString(strconv.Itoa(t.Line))
	for _, a := range t.Annotations {
		b.WriteString("\n\t")
		b.WriteString(a)
	}
	return b.String()
}
