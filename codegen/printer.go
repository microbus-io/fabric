package main

import (
	"fmt"
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

// StdPrinter is an implementation of IndentPrinter that prints to stdout and stderr.
type StdPrinter struct {
	depth   int
	Verbose bool
}

// Indent increments the indentation depth by 1.
func (p *StdPrinter) Indent() int {
	p.depth++
	return p.depth
}

// Indent decrements the indentation depth by 1.
func (p *StdPrinter) Unindent() int {
	p.depth--
	return p.depth
}

// Info prints a message to stderr at the current indentation depth.
func (p *StdPrinter) Error(format string, args ...any) {
	indentation := strings.Repeat("  ", p.depth)
	fmt.Fprintf(os.Stdout, indentation+format+"\r\n", args...)
}

// Info prints a message to stdout at the current indentation depth.
func (p *StdPrinter) Info(format string, args ...any) {
	indentation := strings.Repeat("  ", p.depth)
	fmt.Fprintf(os.Stdout, indentation+format+"\r\n", args...)
}

// Debug prints a message to stdout at the current indentation depth
// but only if the printer is in verbose mode.
func (p *StdPrinter) Debug(format string, args ...any) {
	if p.depth > 0 && !p.Verbose {
		return
	}
	indentation := strings.Repeat("  ", p.depth)
	fmt.Fprintf(os.Stdout, indentation+format+"\r\n", args...)
}
