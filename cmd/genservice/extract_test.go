/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"os"
	"path/filepath"
	"testing"

	"github.com/microbus-io/testarossa"
)

// writeDefinition writes a minimal api-package definition.go into a temp dir and returns the dir.
func writeDefinition(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	src := "package svcapi\n\nimport \"github.com/microbus-io/fabric/define\"\n\n" +
		"const Hostname = \"my.service\"\n\n" + body + "\n"
	err := os.WriteFile(filepath.Join(dir, "definition.go"), []byte(src), 0o644)
	testarossa.For(t).NoError(err)
	return dir
}

// TestParseServiceRejectsBacktickConfigDefault pins that a config whose Default value contains a
// backtick is rejected, because the value is embedded into a Go raw string literal in the generated
// code where a backtick would close the literal early and yield uncompilable Go.
func TestParseServiceRejectsBacktickConfigDefault(t *testing.T) {
	assert := testarossa.For(t)
	// A double-quoted Go string may hold a literal backtick.
	body := "var MyConfig = define.Config{ Default: \"wei" + "`" + "rd\" }"
	_, err := parseService(writeDefinition(t, body))
	if assert.Error(err) {
		assert.Contains(err.Error(), "MyConfig")
		assert.Contains(err.Error(), "backtick")
	}
}

// TestParseServiceAcceptsCleanConfigDefault pins that an ordinary config default parses.
func TestParseServiceAcceptsCleanConfigDefault(t *testing.T) {
	assert := testarossa.For(t)
	body := "var MyConfig = define.Config{ Default: \"normal\" }"
	_, err := parseService(writeDefinition(t, body))
	assert.NoError(err)
}

// TestParseServiceRejectsBacktickGodoc pins the sibling rule for a feature's godoc.
func TestParseServiceRejectsBacktickGodoc(t *testing.T) {
	assert := testarossa.For(t)
	body := "// MyConfig holds a wei" + "`" + "rd value.\nvar MyConfig = define.Config{ Default: \"x\" }"
	_, err := parseService(writeDefinition(t, body))
	if assert.Error(err) {
		assert.Contains(err.Error(), "MyConfig")
		assert.Contains(err.Error(), "backtick")
	}
}

// TestParseServiceRejectsNonLiteralRequiredClaims pins that a RequiredClaims factored into a const is
// rejected with an actionable error rather than silently producing an ungated endpoint.
func TestParseServiceRejectsNonLiteralRequiredClaims(t *testing.T) {
	assert := testarossa.For(t)
	body := "const gate = \"roles.customer\"\n" +
		"var Secret = define.Function{ Host: Hostname, Method: \"GET\", Route: \"/secret\", RequiredClaims: gate }"
	_, err := parseService(writeDefinition(t, body))
	if assert.Error(err) {
		assert.Contains(err.Error(), "Secret")
		assert.Contains(err.Error(), "RequiredClaims")
		assert.Contains(err.Error(), "string literal")
	}
}

// TestParseServiceAcceptsLiteralRequiredClaims pins that the inlined literal form parses cleanly.
func TestParseServiceAcceptsLiteralRequiredClaims(t *testing.T) {
	assert := testarossa.For(t)
	body := "var Secret = define.Function{ Host: Hostname, Method: \"GET\", Route: \"/secret\", RequiredClaims: \"roles.customer\" }"
	_, err := parseService(writeDefinition(t, body))
	assert.NoError(err)
}

// TestParseServiceFieldJSONNames pins the state key a task output is written under: the json tag's name, else the
// Go name, and none for a field encoding/json does not marshal.
func TestParseServiceFieldJSONNames(t *testing.T) {
	assert := testarossa.For(t)
	dir := writeDefinition(t, "type StepOut struct {\n"+
		"\tCountOut int `json:\"count,omitzero\"`\n"+
		"\tPlain    int\n"+
		"\tBare     int `json:\",omitzero\"`\n"+
		"\tOther    int `yaml:\"other\"`\n"+
		"\tSkipped  int `json:\"-\"`\n"+
		"\tDash     int `json:\"-,\"`\n"+
		"\thidden   int\n"+
		"}\n")
	svc, err := parseService(dir)
	if !assert.NoError(err) {
		return
	}
	got := map[string]string{}
	for _, f := range svc.structs["StepOut"].fields {
		got[f.goName] = f.jsonName
	}
	assert.Expect(
		got["CountOut"], "count",
		got["Plain"], "Plain",
		got["Bare"], "Bare",
		got["Other"], "Other",
		got["Skipped"], "",
		got["Dash"], "-",
		got["hidden"], "",
	)
}

// TestParseServiceRejectsStringOption pins that a task argument may not carry the json ",string" option, since the
// state holds each field as its bare value.
func TestParseServiceRejectsStringOption(t *testing.T) {
	assert := testarossa.For(t)
	for _, tc := range []struct{ in, out string }{
		{out: "\tCount int64 `json:\"count,omitzero,string\"`\n"},
		{in: "\tCount int64 `json:\"count,string\"`\n"},
	} {
		dir := writeDefinition(t, "var Step = define.Task{Host: Hostname, Method: \"POST\", Route: \":428/step\", In: StepIn{}, Out: StepOut{}}\n"+
			"type StepIn struct {\n"+tc.in+"}\n"+
			"type StepOut struct {\n"+tc.out+"}\n")
		_, err := parseService(dir)
		if assert.Error(err, tc.in+tc.out) {
			assert.Contains(err.Error(), ",string")
		}
	}
}

// TestParseServiceRejectsEmbeddedArguments pins that a task or workflow In or Out struct may not embed another
// struct, whose fields would not be read or written as state fields.
func TestParseServiceRejectsEmbeddedArguments(t *testing.T) {
	assert := testarossa.For(t)
	dir := writeDefinition(t, "var Flow = define.Workflow{Host: Hostname, Method: \"GET\", Route: \":428/flow\", In: FlowIn{}, Out: FlowOut{}}\n"+
		"type Common struct {\n\tTenant string `json:\"tenant,omitzero\"`\n}\n"+
		"type FlowIn struct {\n\tCommon\n}\n"+
		"type FlowOut struct{}\n")
	_, err := parseService(dir)
	if assert.Error(err) {
		assert.Contains(err.Error(), "embeds Common")
	}
}

// TestParseServiceRejectsReservedStateNames pins that a task or workflow argument may not map to a state field
// whose name ends with "!" or "+", which the engine refuses to write.
func TestParseServiceRejectsReservedStateNames(t *testing.T) {
	assert := testarossa.For(t)
	for _, tag := range []string{"items+", "summary!"} {
		dir := writeDefinition(t, "var Flow = define.Workflow{Host: Hostname, Method: \"GET\", Route: \":428/flow\", In: FlowIn{}, Out: FlowOut{}}\n"+
			"type FlowIn struct{}\n"+
			"type FlowOut struct {\n\tValue string `json:\""+tag+"\"`\n}\n")
		_, err := parseService(dir)
		if assert.Error(err, tag) {
			assert.Contains(err.Error(), tag)
		}
	}
}

// TestParseServiceRejectsUnmarshaledArguments pins that a task or workflow argument must marshal to JSON, since
// an unexported or json:"-" field cannot be carried in the state as an input or an output.
func TestParseServiceRejectsUnmarshaledArguments(t *testing.T) {
	assert := testarossa.For(t)
	for _, tc := range []struct{ in, out string }{
		{in: "\tSecret string `json:\"-\"`\n"},
		{in: "\tsecret string\n"},
		{out: "\tSecret string `json:\"-\"`\n"},
		{out: "\tsecret string\n"},
	} {
		dir := writeDefinition(t, "var Flow = define.Workflow{Host: Hostname, Method: \"GET\", Route: \":428/flow\", In: FlowIn{}, Out: FlowOut{}}\n"+
			"type FlowIn struct {\n"+tc.in+"}\n"+
			"type FlowOut struct {\n\tDone bool `json:\"done,omitzero\"`\n"+tc.out+"}\n")
		_, err := parseService(dir)
		if assert.Error(err, tc.in+tc.out) {
			assert.Contains(err.Error(), "cannot be carried in the state")
		}
	}
}
