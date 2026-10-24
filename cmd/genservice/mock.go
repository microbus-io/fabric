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
	"maps"
	"sort"
	"strings"
)

// mockModel is the tree handed to mock.txt and mock_test.txt. The mock lives in the service package
// and references the api package by import path, so In/Out field types are qualified and the api path
// is part of the model. Imports is the set for mock.go; TestImports the (largely disjoint) set for
// mock_test.go.
type mockModel struct {
	Header       string
	Package      string // service package name, e.g. "svc"
	APIPkg       string // api package name, e.g. "svcapi"
	TestFuncName string // e.g. "TestSvc_Mock"
	Imports      importSet
	TestImports  importSet
	Methods      []*mockView
}

// mockView is one mockable ToDo method. Kind selects the emission shape: a standard setter/executor
// pair, a web handler pair, or the workflow-graph variant (synthetic subgraph delegating to a typed
// handler).
type mockView struct {
	Name      string // method name, e.g. "Greet" or "OnObserveQueueDepth"
	Marker    string // MARKER comment value, e.g. "Greet" or "QueueDepth"
	Kind      string // standard | web | graph
	Sig       string // standard: full param decl, e.g. "ctx context.Context, flow *workflow.Flow, item string"
	Results   string // standard: full result decl, e.g. "done bool, err error"
	Args      string // standard: executor call args, e.g. "ctx, flow, item"
	LValues   string // standard: assignment LHS when there are outputs, e.g. "done, err"
	RetNoErr  string // standard: output names for the return statement, e.g. "done"
	SingleErr bool   // standard: the result list is just (err error)
	usesFlow  bool   // standard: the method carries a *workflow.Flow param (a task)

	// Graph-only.
	APIPkg      string       // api package alias, for the synthetic task's In/Out references
	KebabRoute  string       // kebab-cased name, for the synthetic mock route
	InType      string       // e.g. "svcapi.MainFlowIn"
	HandlerSig  string       // typed handler func signature
	HandlerArgs string       // ", in.Item, in.Count"
	HandlerOut  string       // "done, err"
	StateWrites []stateWrite // the handler's outputs, written to the flow's state

	// Test-only.
	TestVarDecls []paramDecl // locals to declare before a standard subtest call
	TestArgs     string      // standard subtest call args, e.g. "ctx, nil, item"
	TestLHS      string      // standard subtest assignment LHS, e.g. "_, err"
}

// paramDecl is a name/type pair for a local variable declaration in a subtest body.
type paramDecl struct {
	Name string
	Type string
}

// emitMock renders the service package's mock.go.
func emitMock(svc *service, pkg, apiPath, header string, resolveSource func(string) (*service, error)) ([]byte, error) {
	m, err := buildMockModel(svc, pkg, apiPath, header, resolveSource)
	if err != nil {
		return nil, err
	}
	return renderMock(m, "mock.txt")
}

// emitMockTest renders the service package's mock_test.go.
func emitMockTest(svc *service, pkg, apiPath, header string, resolveSource func(string) (*service, error)) ([]byte, error) {
	m, err := buildMockModel(svc, pkg, apiPath, header, resolveSource)
	if err != nil {
		return nil, err
	}
	return renderMock(m, "mock_test.txt")
}

// renderMock executes the named template against the model and gofmts the result.
func renderMock(m *mockModel, tmpl string) ([]byte, error) {
	var buf bytes.Buffer
	err := clientTemplate.ExecuteTemplate(&buf, tmpl, m)
	if err != nil {
		return nil, err
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n%s", err, numberLines(buf.Bytes()))
	}
	return out, nil
}

// buildMockModel projects the parsed service into the mock model: the mockable methods in ToDo order
// and the two computed import sets.
func buildMockModel(svc *service, pkg, apiPath, header string, resolveSource func(string) (*service, error)) (*mockModel, error) {
	err := validateForClient(svc)
	if err != nil {
		return nil, err
	}
	m := &mockModel{
		Header: header, Package: pkg, APIPkg: svc.apiPkg,
		TestFuncName: "Test" + pascalCase(pkg) + "_Mock",
	}

	// Methods are grouped by kind in the same order the ToDo interface declares them.
	var funcs, webs, tasks, workflows, inbound, tickers, observe, changed []*mockView
	for _, f := range svc.features {
		switch f.kind {
		case "Function":
			funcs = append(funcs, standardMock(f.name, f.name, svc.apiPkg, false, svc.fieldsOf(f.in), svc.fieldsOf(f.out)))
		case "Web":
			webs = append(webs, &mockView{Name: f.name, Marker: f.name, Kind: "web"})
		case "Task":
			tasks = append(tasks, standardMock(f.name, f.name, svc.apiPkg, true, svc.fieldsOf(f.in), svc.fieldsOf(f.out)))
		case "Workflow":
			workflows = append(workflows, graphMock(f.name, svc.apiPkg, svc.fieldsOf(f.in), svc.fieldsOf(f.out)))
		case "InboundEvent":
			iv, _, _, err := inboundView(svc, f, resolveSource)
			if err != nil {
				return nil, err
			}
			inbound = append(inbound, standardMock(f.name, f.name, f.srcPkg, false, iv.inFields, iv.outFields))
		case "Ticker":
			tickers = append(tickers, standardMock(f.name, f.name, "", false, nil, nil))
		case "Metric":
			if attrBool(f.attrs, "Observable") {
				observe = append(observe, standardMock("OnObserve"+f.name, f.name, "", false, nil, nil))
			}
		case "Config":
			if attrBool(f.attrs, "Callback") {
				changed = append(changed, standardMock("OnChanged"+f.name, f.name, "", false, nil, nil))
			}
		}
	}
	m.Methods = append(m.Methods, funcs...)
	m.Methods = append(m.Methods, webs...)
	m.Methods = append(m.Methods, tasks...)
	m.Methods = append(m.Methods, workflows...)
	m.Methods = append(m.Methods, inbound...)
	m.Methods = append(m.Methods, tickers...)
	m.Methods = append(m.Methods, observe...)
	m.Methods = append(m.Methods, changed...)

	aliases, err := mockAliases(svc, apiPath, resolveSource)
	if err != nil {
		return nil, err
	}
	m.Imports = mockImports(m, aliases)
	m.TestImports = mockTestImports(m, aliases)
	return m, nil
}

// standardMock builds the view for a setter/executor pair. method is the Go method name; marker the
// MARKER comment value; apiPkg the alias used to qualify bare In/Out field types ("" for none);
// hasFlow adds a *workflow.Flow parameter (tasks).
func standardMock(method, marker, apiPkg string, hasFlow bool, in, out []fieldDef) *mockView {
	sigParts := []string{"ctx context.Context"}
	execArgs := []string{"ctx"}
	testArgs := []string{"ctx"}
	if hasFlow {
		sigParts = append(sigParts, "flow *workflow.Flow")
		execArgs = append(execArgs, "flow")
		testArgs = append(testArgs, "nil")
	}
	var varDecls []paramDecl
	for _, x := range in {
		n := lowerFirst(x.goName)
		t := qualifyTypes(x.typ, apiPkg)
		sigParts = append(sigParts, n+" "+t)
		execArgs = append(execArgs, n)
		testArgs = append(testArgs, n)
		varDecls = append(varDecls, paramDecl{Name: n, Type: t})
	}
	var resParts, outNames []string
	for _, x := range out {
		n := lowerFirst(x.goName)
		resParts = append(resParts, n+" "+qualifyTypes(x.typ, apiPkg))
		outNames = append(outNames, n)
	}
	mv := &mockView{
		Name:         method,
		Marker:       marker,
		Kind:         "standard",
		Sig:          strings.Join(sigParts, ", "),
		Results:      strings.Join(append(resParts, "err error"), ", "),
		Args:         strings.Join(execArgs, ", "),
		SingleErr:    len(out) == 0,
		usesFlow:     hasFlow,
		TestVarDecls: varDecls,
		TestArgs:     strings.Join(testArgs, ", "),
	}
	if len(out) > 0 {
		mv.LValues = strings.Join(append(append([]string{}, outNames...), "err"), ", ")
		mv.RetNoErr = strings.Join(outNames, ", ")
	}
	lhs := make([]string, 0, len(out)+1)
	for range out {
		lhs = append(lhs, "_")
	}
	mv.TestLHS = strings.Join(append(lhs, "err"), ", ")
	return mv
}

// graphMock builds the workflow-graph mock view. The synthetic task delegates to a typed handler
// reconstructed from the workflow's In/Out structs.
func graphMock(name, apiPkg string, in, out []fieldDef) *mockView {
	mv := &mockView{
		Name:       name,
		Marker:     name,
		Kind:       "graph",
		APIPkg:     apiPkg,
		KebabRoute: kebabCase(name),
		InType:     apiPkg + "." + name + "In",
	}
	sig := "func(ctx context.Context, flow *workflow.Flow"
	for _, x := range in {
		sig += ", " + lowerFirst(x.goName) + " " + qualifyTypes(x.typ, apiPkg)
	}
	sig += ") ("
	for _, x := range out {
		sig += lowerFirst(x.goName) + " " + qualifyTypes(x.typ, apiPkg) + ", "
	}
	mv.HandlerSig = sig + "err error)"

	var hargs strings.Builder
	for _, x := range in {
		fmt.Fprintf(&hargs, ", in.%s", x.goName)
	}
	mv.HandlerArgs = hargs.String()

	var outNames []string
	for _, x := range out {
		outNames = append(outNames, lowerFirst(x.goName))
	}
	mv.HandlerOut = strings.Join(append(append([]string{}, outNames...), "err"), ", ")

	mv.StateWrites = stateWrites(out, func(x fieldDef) string { return "&" + lowerFirst(x.goName) })
	return mv
}

// mockAliases maps every package alias that may appear in a qualified In/Out type to its import path:
// the api package's own imports, the api package itself, and (for inbound events) each source package
// and its imports.
func mockAliases(svc *service, apiPath string, resolveSource func(string) (*service, error)) (map[string]string, error) {
	aliases := map[string]string{}
	maps.Copy(aliases, svc.imports)
	aliases[svc.apiPkg] = apiPath
	for _, f := range svc.features {
		if f.kind != "InboundEvent" {
			continue
		}
		srcPath, ok := svc.imports[f.srcPkg]
		if !ok {
			continue
		}
		srcSvc, err := resolveSource(srcPath)
		if err != nil {
			return nil, err
		}
		aliases[f.srcPkg] = srcPath
		maps.Copy(aliases, srcSvc.imports)
	}
	return aliases, nil
}

// mockImports computes the import set for mock.go: a per-kind base plus the paths resolved from the
// qualified type expressions the generated code references.
func mockImports(m *mockModel, aliases map[string]string) importSet {
	imports := map[string]bool{impContext: true, impErrors: true, impConnector: true}
	var hasWeb, hasGraph, hasFlow bool
	for _, mv := range m.Methods {
		switch mv.Kind {
		case "web":
			hasWeb = true
		case "graph":
			hasGraph = true
		}
		if mv.usesFlow {
			hasFlow = true
		}
		addResolved(imports, aliases, mv.Sig, mv.Results, mv.HandlerSig, mv.InType)
	}
	if hasWeb || hasGraph {
		imports[impHTTP] = true
	}
	if hasFlow || hasGraph {
		imports[impWorkflow] = true
	}
	if hasGraph {
		imports[impJSON] = true
		imports[impHTTPX] = true
		imports[impSub] = true
		imports[impUtils] = true
	}
	return splitImports(imports)
}

// mockTestImports computes the import set for mock_test.go.
func mockTestImports(m *mockModel, aliases map[string]string) importSet {
	imports := map[string]bool{impTesting: true, impConnector: true, impTestarossa: true}
	var hasWeb, hasGraph, hasFlow bool
	for _, mv := range m.Methods {
		switch mv.Kind {
		case "web":
			hasWeb = true
		case "graph":
			hasGraph = true
		}
		if mv.usesFlow {
			hasFlow = true
		}
		// Every non-web no-op handler signature spells out context.Context. Add the import explicitly
		// rather than relying on it leaking in via the api package's client.go (which a config-only or
		// web-only microservice does not import).
		if mv.Kind != "web" {
			imports[impContext] = true
		}
		addResolved(imports, aliases, mv.Sig, mv.Results, mv.HandlerSig)
	}
	if hasWeb {
		imports[impHTTP] = true
		imports[impHTTPX] = true
	}
	if hasFlow || hasGraph {
		imports[impWorkflow] = true
	}
	return splitImports(imports)
}

// addResolved adds, for each type expression, the import paths of its pkg.Type selectors resolved
// against aliases. Selectors with no alias entry (context, workflow, http, ...) are left to the
// per-kind base set.
func addResolved(imports map[string]bool, aliases map[string]string, exprs ...string) {
	for _, e := range exprs {
		for _, sel := range selectorsIn(e) {
			if path, ok := aliases[sel]; ok {
				imports[path] = true
			}
		}
	}
}

// splitImports partitions a path set into the std and external groups, each sorted.
func splitImports(imports map[string]bool) importSet {
	var set importSet
	for p := range imports {
		if isStdlib(p) {
			set.Std = append(set.Std, p)
		} else {
			set.Ext = append(set.Ext, p)
		}
	}
	sort.Strings(set.Std)
	sort.Strings(set.Ext)
	return set
}
