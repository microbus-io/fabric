/*
Copyright 2023 Microbus LLC and various contributors

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
