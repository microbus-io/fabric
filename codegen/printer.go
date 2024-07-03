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

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// IndentPrinter is a printer that maintains a state of indentation
// and adjusts printing accordingly.
type IndentPrinter interface {
	Indent() int
	Unindent() int
	Error(format string, args ...any)
	Info(format string, args ...any)
	Debug(format string, args ...any)
}

// Printer is an implementation of IndentPrinter that prints to stdout and stderr.
type Printer struct {
	depth     int
	Verbose   bool
	outWriter io.Writer
	errWriter io.Writer
}

// Indent increments the indentation depth by 1.
func (p *Printer) Indent() int {
	p.depth++
	return p.depth
}

// Indent decrements the indentation depth by 1.
func (p *Printer) Unindent() int {
	p.depth--
	return p.depth
}

// Info prints a message to stderr at the current indentation depth.
func (p *Printer) Error(format string, args ...any) {
	indentation := strings.Repeat("  ", p.depth)
	w := p.errWriter
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, indentation+format+"\n", args...)
}

// Info prints a message to stdout at the current indentation depth.
func (p *Printer) Info(format string, args ...any) {
	indentation := strings.Repeat("  ", p.depth)
	w := p.outWriter
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, indentation+format+"\n", args...)
}

// Debug prints a message to stdout at the current indentation depth
// but only if the printer is in verbose mode.
func (p *Printer) Debug(format string, args ...any) {
	if !p.Verbose {
		return
	}
	indentation := strings.Repeat("  ", p.depth)
	w := p.outWriter
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, indentation+format+"\n", args...)
}
