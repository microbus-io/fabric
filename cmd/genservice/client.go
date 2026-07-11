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
	"bytes"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Import paths the generated client may reference.
const (
	impContext     = "context"
	impJSON        = "encoding/json"
	impIter        = "iter"
	impHTTP        = "net/http"
	impReflect     = "reflect"
	impStrconv     = "strconv"
	impTesting     = "testing"
	impDV8         = "github.com/microbus-io/dv8"
	impWorkflow    = "github.com/microbus-io/dwarf/workflow"
	impErrors      = "github.com/microbus-io/errors"
	impTestarossa  = "github.com/microbus-io/testarossa"
	impCfg         = "github.com/microbus-io/fabric/cfg"
	impConnector   = "github.com/microbus-io/fabric/connector"
	impHTTPX       = "github.com/microbus-io/fabric/httpx"
	impPub         = "github.com/microbus-io/fabric/pub"
	impService     = "github.com/microbus-io/fabric/service"
	impSub         = "github.com/microbus-io/fabric/sub"
	impUtils       = "github.com/microbus-io/fabric/utils"
	impApplication = "github.com/microbus-io/fabric/application"
	impForeman     = "github.com/microbus-io/fabric/coreservices/foreman"
	impForemanAPI  = "github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
)

// clientModel is the in-memory tree handed to client.txt: the package, computed imports, var guards,
// and the feature views grouped by kind. The template decides, from the Has*/Need* predicates, which
// client structs and helpers to emit, and ranges over the feature slices to emit the methods.
type clientModel struct {
	Header  string
	Package string
	Imports importSet

	Funcs          []*featureView
	Webs           []*featureView
	Tasks          []*featureView
	Workflows      []*featureView
	OutboundEvents []*featureView
}

type importSet struct {
	Std []string
	Ext []string
}

func (m *clientModel) HasFunc() bool     { return len(m.Funcs) > 0 }
func (m *clientModel) HasWeb() bool      { return len(m.Webs) > 0 }
func (m *clientModel) HasTask() bool     { return len(m.Tasks) > 0 }
func (m *clientModel) HasWorkflow() bool { return len(m.Workflows) > 0 }
func (m *clientModel) HasEvent() bool    { return len(m.OutboundEvents) > 0 }

// NeedClient reports whether the Client/MulticastClient proxies are required (functions or web).
func (m *clientModel) NeedClient() bool { return m.HasFunc() || m.HasWeb() }

// NeedExecutor reports whether the Executor proxy is required (tasks or workflows). The Subgraph proxy is
// gated separately on HasWorkflow, since it carries only workflow methods (a task is not independently invocable).
func (m *clientModel) NeedExecutor() bool { return m.HasTask() || m.HasWorkflow() }

// NeedResponse reports whether multicastResponse and marshalPublish are required (functions or events).
func (m *clientModel) NeedResponse() bool { return m.HasFunc() || m.HasEvent() }

// featureView is one feature with the fragments the templates interpolate into method bodies.
type featureView struct {
	Name        string
	Doc         string // raw godoc text, or "" when undocumented
	DocComment  string // "// ...\n" lines, or "" when undocumented
	In          string // In struct type name
	Out         string // Out struct type name
	SrcPkg      string // InboundEvent only: source api package alias
	SrcEvent    string // InboundEvent only: source outbound event name
	HookOptions string // InboundEvent only: rendered sub.Options for Hook.WithOptions, or ""

	// Endpoint subscription options (intermediate.go only).
	ReqClaims  string // sub.RequiredClaims expression, or ""
	TimeBudget string // rendered sub.TimeBudget duration expr, or ""
	Queue      string // "none" | "default" | custom queue name | ""
	Manual     bool   // sub.Manual()
	TagArgs    string // rendered sub.Tag arguments (e.g. `"python"`), or ""

	// Web client shape (client.go webs only): "plain" (ctx, relativeURL), "body" (ctx, relativeURL, body),
	// or "any" (ctx, method, relativeURL, body). Selected from the endpoint's HTTP method.
	WebShape string

	apiPkg    string // when set, bare In/Out field types are qualified with this package alias
	inFields  []fieldDef
	outFields []fieldDef
}

// newFeatureView builds the view for a feature, resolving its In/Out struct fields.
func newFeatureView(svc *service, f feature) *featureView {
	return &featureView{
		Name:       f.name,
		Doc:        f.doc,
		DocComment: docComment(f.doc),
		In:         f.in,
		Out:        f.out,
		inFields:   svc.fieldsOf(f.in),
		outFields:  svc.fieldsOf(f.out),
	}
}

// HasOut reports whether the feature has any output fields.
func (f *featureView) HasOut() bool { return len(f.outFields) > 0 }

// Params renders ", name type" for each input field (leading comma per param).
func (f *featureView) Params() string {
	var b strings.Builder
	for _, x := range f.inFields {
		fmt.Fprintf(&b, ", %s %s", lowerFirst(x.goName), qualifyTypes(x.typ, f.apiPkg))
	}
	return b.String()
}

// Returns renders "name type, " for each output field (named returns, trailing comma per return).
func (f *featureView) Returns() string {
	var b strings.Builder
	for _, x := range f.outFields {
		fmt.Fprintf(&b, "%s %s, ", lowerFirst(x.goName), qualifyTypes(x.typ, f.apiPkg))
	}
	return b.String()
}

// Zeros renders "name, " for each output field (the named returns at their zero values).
func (f *featureView) Zeros() string {
	var b strings.Builder
	for _, x := range f.outFields {
		fmt.Fprintf(&b, "%s, ", lowerFirst(x.goName))
	}
	return b.String()
}

// Dot renders "recv.GoName, " for each output field.
func (f *featureView) Dot(recv string) string {
	var b strings.Builder
	for _, x := range f.outFields {
		fmt.Fprintf(&b, "%s.%s, ", recv, x.goName)
	}
	return b.String()
}

// InArgs renders ", recv.GoName" for each input field, for passing into a handler call.
func (f *featureView) InArgs(recv string) string {
	var b strings.Builder
	for _, x := range f.inFields {
		fmt.Fprintf(&b, ", %s.%s", recv, x.goName)
	}
	return b.String()
}

// InLit renders the In struct literal, e.g. "VerifyCreditIn{CreditScore: creditScore}".
func (f *featureView) InLit() string {
	var b strings.Builder
	b.WriteString(f.In)
	b.WriteString("{")
	for i, x := range f.inFields {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s: %s", x.goName, lowerFirst(x.goName))
	}
	b.WriteString("}")
	return b.String()
}

// emitClient builds the model from the parsed service and renders client.txt.
func emitClient(svc *service, header string) ([]byte, error) {
	err := validateForClient(svc)
	if err != nil {
		return nil, err
	}
	m := buildClientModel(svc, header)
	var buf bytes.Buffer
	err = clientTemplate.ExecuteTemplate(&buf, "client.txt", m)
	if err != nil {
		return nil, err
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n%s", err, numberLines(buf.Bytes()))
	}
	return out, nil
}

// validateForClient reports the first feature that lacks the In/Out type carriers its client methods
// require. Web carries neither and is exempt.
func validateForClient(svc *service) error {
	for _, f := range svc.features {
		switch f.kind {
		case "Function", "Task", "Workflow", "OutboundEvent":
			if f.in == "" || f.out == "" {
				return fmt.Errorf("%s %q: both In and Out must be set in definition.go", f.kind, f.name)
			}
		}
	}
	return nil
}

// buildClientModel projects the parsed service into the client.txt model: the feature views grouped
// by kind and the computed import set.
func buildClientModel(svc *service, header string) *clientModel {
	m := &clientModel{Header: header, Package: svc.apiPkg}
	for _, f := range svc.features {
		fv := newFeatureView(svc, f)
		switch f.kind {
		case "Function":
			m.Funcs = append(m.Funcs, fv)
		case "Web":
			fv.WebShape = webShape(attrString(f.attrs, "Method"))
			m.Webs = append(m.Webs, fv)
		case "Task":
			m.Tasks = append(m.Tasks, fv)
		case "Workflow":
			m.Workflows = append(m.Workflows, fv)
		case "OutboundEvent":
			m.OutboundEvents = append(m.OutboundEvents, fv)
		}
	}

	imports := map[string]bool{}
	need := func(paths ...string) {
		for _, p := range paths {
			imports[p] = true
		}
	}
	if m.NeedResponse() {
		need(impHTTP)
	}
	if m.NeedClient() {
		need(impContext, impService, impPub)
	}
	if m.NeedExecutor() {
		need(impContext, impService, impPub, impWorkflow)
	}
	if m.HasEvent() {
		need(impContext, impIter, impHTTP, impService, impPub, impSub, impHTTPX, impErrors, impDV8)
	}
	if m.HasFunc() {
		need(impContext, impIter, impService, impPub, impHTTPX, impErrors)
	}
	if m.NeedResponse() {
		need(impContext, impService, impPub, impHTTPX, impIter, impReflect)
	}
	if m.HasTask() {
		need(impContext, impService, impPub, impHTTPX, impErrors, impJSON, impWorkflow)
	}
	if m.HasWorkflow() {
		need(impContext, impErrors, impJSON, impWorkflow)
	}
	if m.HasWeb() {
		need(impContext, impIter, impHTTP, impPub, impHTTPX)
	}
	for p := range featureSelectorImports(svc, nil) {
		need(p)
	}
	for p := range imports {
		if isStdlib(p) {
			m.Imports.Std = append(m.Imports.Std, p)
		} else {
			m.Imports.Ext = append(m.Imports.Ext, p)
		}
	}
	sort.Strings(m.Imports.Std)
	sort.Strings(m.Imports.Ext)
	return m
}

// webShape classifies a web endpoint's HTTP method into the client method shape it gets: "any" for the
// ANY method (caller chooses method + body), "body" for methods that carry a request body (POST/PUT/
// PATCH), and "plain" for the rest (GET/HEAD/DELETE/OPTIONS/TRACE/CONNECT), which take neither.
func webShape(method string) string {
	switch strings.ToUpper(method) {
	case "ANY":
		return "any"
	case "POST", "PUT", "PATCH":
		return "body"
	default:
		return "plain"
	}
}

// docComment renders a godoc block, one // line per source line; "" when the doc is empty.
func docComment(doc string) string {
	if doc == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(doc, "\n") {
		b.WriteString("// ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// goGeneratedLineRE matches a Go "Code generated ... DO NOT EDIT." marker line, regardless of which
// tool authored it, so a header inherited from a since-retired generator is stripped before re-emission.
var goGeneratedLineRE = regexp.MustCompile(`(?m)^//\s*Code generated .* DO NOT EDIT\.\s*\n?`)

// existingHeader returns the leading comment block (license header) of an existing generated file,
// with any prior generated-by marker removed, so the generator preserves a header it never authors.
// It returns "" when the file is absent or has no header before the package clause.
func existingHeader(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(data)
	if strings.HasPrefix(s, "package ") {
		return ""
	}
	head, _, found := strings.Cut(s, "\npackage ")
	if !found {
		return ""
	}
	head = goGeneratedLineRE.ReplaceAllString(head, "")
	return strings.TrimRight(head, "\n")
}

// isStdlib reports whether an import path is in the standard library (its first segment has no dot).
func isStdlib(path string) bool {
	seg, _, _ := strings.Cut(path, "/")
	return !strings.Contains(seg, ".")
}

// numberLines prefixes each line with its number, for gofmt error diagnostics.
func numberLines(src []byte) string {
	var b strings.Builder
	for i, line := range strings.Split(string(src), "\n") {
		fmt.Fprintf(&b, "%4d\t%s\n", i+1, line)
	}
	return b.String()
}
