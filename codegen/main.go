package main

import (
	"flag"
	"os"
	"strings"
)

// main is executed when "go generate" is run in the current working directory.
func main() {
	// Load flags from environment variable because can't pass arguments to code-generator
	var flagForce bool
	var flagVerbose bool
	env := os.Getenv("MICROBUS_CODEGEN")
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.BoolVar(&flagForce, "f", false, "Force processing even if no change detected")
	flags.BoolVar(&flagVerbose, "v", false, "Verbose output")
	_ = flags.Parse(strings.Split(env, " "))

	// Run generator
	gen := NewGenerator()
	gen.Force = flagForce
	gen.Printer = &Printer{
		Verbose: flagVerbose,
	}
	err := gen.Run()
	if err != nil {
		if flagVerbose {
			gen.Printer.Error("%+v", err)
		} else {
			gen.Printer.Error("%v", err)
		}
		os.Exit(-1)
	}
}
