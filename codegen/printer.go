package main

import (
	"fmt"
	"strings"
)

type Printer struct {
	depth int
}

var printer = &Printer{}

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

// Printf prints a message to the standard output at the current indentation depth.
func (p *Printer) Printf(format string, args ...any) {
	if p.depth > 0 && !flagVerbose {
		return
	}
	indentation := strings.Repeat("  ", p.depth)
	fmt.Printf(indentation+format+"\r\n", args...)
}