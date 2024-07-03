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
	"flag"
	"os"
	"strings"

	"github.com/microbus-io/fabric/env"
)

// main is executed when "go generate" is run in the current working directory.
func main() {
	// Load flags from environment variable because can't pass arguments to code-generator
	var flagForce bool
	var flagVerbose bool
	envVal := env.Get("MICROBUS_CODEGEN")
	if envVal == "" {
		envVal = env.Get("CODEGEN")
	}
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.BoolVar(&flagForce, "f", false, "Force processing even if no change detected")
	flags.BoolVar(&flagVerbose, "v", false, "Verbose output")
	_ = flags.Parse(strings.Split(envVal, " "))

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
