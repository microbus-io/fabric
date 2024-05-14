/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package errors

import (
	"fmt"
)

// trace is a single stack trace location
type trace struct {
	File     string `json:"file"`
	Function string `json:"function"`
	Line     int    `json:"line"`
}

// String returns a string representation of the trace
func (t *trace) String() string {
	return fmt.Sprintf("- %s\n  %s:%d", t.Function, t.File, t.Line)
}
