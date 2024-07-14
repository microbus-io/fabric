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
	"bytes"
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestCodegen_PrinterWriters(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	p := &Printer{
		Verbose:   true,
		outWriter: &outBuf,
		errWriter: &errBuf,
	}

	testarossa.NotContains(t, outBuf.String(), "Hello")
	p.Debug("Hello")
	testarossa.Contains(t, outBuf.String(), "Hello")
	testarossa.SliceLen(t, errBuf.Bytes(), 0)
	outBuf.Reset()

	testarossa.NotContains(t, outBuf.String(), "Hello")
	p.Info("Hello")
	testarossa.Contains(t, outBuf.String(), "Hello")
	testarossa.SliceLen(t, errBuf.Bytes(), 0)
	outBuf.Reset()

	testarossa.NotContains(t, errBuf.String(), "Hello")
	p.Error("Hello")
	testarossa.Contains(t, errBuf.String(), "Hello")
	testarossa.SliceLen(t, outBuf.Bytes(), 0)
	errBuf.Reset()
}

func TestCodegen_Verbose(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	p := &Printer{
		outWriter: &outBuf,
		errWriter: &errBuf,
	}

	p.Verbose = true
	p.Debug("[Debug Verbose]")
	p.Info("[Info Verbose]")
	p.Error("[Error Verbose]")
	p.Verbose = false
	p.Debug("[Debug Succinct]")
	p.Info("[Info Succinct]")
	p.Error("[Error Succinct]")

	testarossa.Contains(t, outBuf.String(), "[Debug Verbose]")
	testarossa.Contains(t, outBuf.String(), "[Info Verbose]")
	testarossa.Contains(t, errBuf.String(), "[Error Verbose]")
	testarossa.NotContains(t, outBuf.String(), "[Debug Succinct]")
	testarossa.Contains(t, outBuf.String(), "[Info Succinct]")
	testarossa.Contains(t, errBuf.String(), "[Error Succinct]")
}

func TestCodegen_Indent(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	p := &Printer{
		outWriter: &outBuf,
		errWriter: &errBuf,
	}
	p.Info("0")
	p.Indent()
	p.Info("1")
	p.Indent()
	p.Info("2")
	p.Unindent()
	p.Info("1")
	p.Unindent()
	p.Info("0")

	lines := strings.Split(outBuf.String(), "\n")
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
		testarossa.FailIf(t, line != "0" && line != "  1" && line != "    2", "Incorrect indentation", line)
	}
}
