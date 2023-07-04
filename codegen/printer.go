/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
