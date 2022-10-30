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
	indentation := strings.Repeat(" ", p.depth*4)
	fmt.Printf(indentation+format+"\r\n", args...)
}
