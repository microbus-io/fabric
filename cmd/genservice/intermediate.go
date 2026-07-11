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
	"sort"
	"strings"
)

// intermediateModel is the tree handed to intermediate.txt. Unlike client.go, intermediate.go lives in
// the service package and references the api package and resources by import path, so those paths and
// the service package name are part of the model.
type intermediateModel struct {
	Header        string
	Package       string // service package name, e.g. "svc"
	APIPkg        string // api package name, e.g. "svcapi"
	APIPath       string // import path of the api package
	ResourcesPath string // import path of the resources package
	Imports       importSet

	Funcs         []*featureView
	Webs          []*featureView
	Tasks         []*featureView
	Workflows     []*featureView
	InboundEvents []*featureView
	Configs       []*configView
	Metrics       []*metricView
	Tickers       []*tickerView
}

func (m *intermediateModel) HasTask() bool     { return len(m.Tasks) > 0 }
func (m *intermediateModel) HasWorkflow() bool { return len(m.Workflows) > 0 }

// ObservableMetrics returns the metrics measured just-in-time via an OnObserve<Name> callback.
func (m *intermediateModel) ObservableMetrics() []*metricView {
	var out []*metricView
	for _, mt := range m.Metrics {
		if mt.Observable {
			out = append(out, mt)
		}
	}
	return out
}

// HasCompileInputs returns whether the microservice has endpoint input types whose
// validation directives are compiled at startup.
func (m *intermediateModel) HasCompileInputs() bool {
	return len(m.Funcs) > 0 || len(m.Tasks) > 0 || len(m.InboundEvents) > 0
}

// CallbackConfigs returns the configs whose change fires an OnChanged<Name> callback.
func (m *intermediateModel) CallbackConfigs() []*configView {
	var out []*configView
	for _, c := range m.Configs {
		if c.Callback {
			out = append(out, c)
		}
	}
	return out
}

// configView is a configuration property's registration and typed accessors.
type configView struct {
	Name       string
	Doc        string // raw description for cfg.Description
	DocComment string
	Type       string // raw getter type: a scalar (string|int|float64|bool|time.Duration) or an api-package struct
	GoType     string // Type qualified for the service package (svcapi.Policy); equals Type for scalars
	Scalar     bool   // whether Type is one of the supported scalar getter types
	Default    string
	Validation string
	Secret     bool
	Callback   bool
}

// metricView is a metric's registration and recorder method.
type metricView struct {
	Name         string
	Doc          string // raw description for DescribeX
	DocComment   string
	Kind         string // counter | gauge | histogram
	RecorderVerb string // Increment | Record
	RecordFn     string // IncrementCounter | RecordGauge | RecordHistogram
	OTelName     string
	ValueType    string // recorder value parameter type
	ValueArg     string // "value" or "float64(value)"
	Labels       []string
	Buckets      string // rendered []float64 (histogram), or ""
	Observable   bool
}

// tickerView is a recurring operation on a schedule.
type tickerView struct {
	Name       string
	DocComment string
	Interval   string // rendered duration expression
}

// emitIntermediate renders the service package's intermediate.go. resolveSource parses the api package
// at a given import path, used to recover an inbound event's handler signature from its source. The
// Hostname, Version, and Description consts are referenced from the api package, so definition.go must
// declare all three.
func emitIntermediate(svc *service, pkg, apiPath, resourcesPath, header string, resolveSource func(string) (*service, error)) ([]byte, error) {
	err := validateForClient(svc)
	if err != nil {
		return nil, err
	}
	if !svc.hasVersion {
		return nil, fmt.Errorf("definition.go must declare `const Version`")
	}
	if !svc.hasDescription {
		return nil, fmt.Errorf("definition.go must declare `const Description`")
	}
	m := &intermediateModel{
		Header: header, Package: pkg,
		APIPkg: svc.apiPkg, APIPath: apiPath, ResourcesPath: resourcesPath,
	}
	var srcPaths []string
	for _, f := range svc.features {
		switch f.kind {
		case "Function":
			m.Funcs = append(m.Funcs, endpointView(svc, f))
		case "Web":
			m.Webs = append(m.Webs, endpointView(svc, f))
		case "Task":
			m.Tasks = append(m.Tasks, endpointView(svc, f))
		case "Workflow":
			m.Workflows = append(m.Workflows, endpointView(svc, f))
		case "InboundEvent":
			iv, srcPath, typePaths, err := inboundView(svc, f, resolveSource)
			if err != nil {
				return nil, err
			}
			m.InboundEvents = append(m.InboundEvents, iv)
			srcPaths = append(srcPaths, srcPath)
			srcPaths = append(srcPaths, typePaths...)
		case "Config":
			cv := buildConfig(f)
			cv.GoType = qualifyTypes(cv.Type, svc.apiPkg)
			m.Configs = append(m.Configs, cv)
		case "Metric":
			m.Metrics = append(m.Metrics, buildMetric(svc, f))
		case "Ticker":
			m.Tickers = append(m.Tickers, buildTicker(svc, f))
		}
	}

	imports := map[string]bool{
		impContext: true, impConnector: true,
		apiPath: true, resourcesPath: true,
	}
	// Endpoints drive the http/sub/httpx surface: every doXxx handler and the web ToDo signature take
	// http types and are registered via sub; only functions carry the httpx-based marshalFunction. A
	// config-only or metric-only microservice needs none of these.
	hasEndpoints := len(m.Funcs) > 0 || len(m.Webs) > 0 || m.HasTask() || m.HasWorkflow()
	if hasEndpoints {
		imports[impHTTP] = true
		imports[impSub] = true
	}
	for _, iv := range m.InboundEvents {
		if iv.HookOptions != "" {
			imports[impSub] = true // Hook.WithOptions(sub....) references the sub package
		}
	}
	if len(m.Funcs) > 0 {
		imports[impHTTPX] = true
	}
	// Functions validate decoded inputs; the startup dv8.Compile covers functions and inbound events
	if m.HasCompileInputs() {
		imports[impDV8] = true
	}
	if len(m.Funcs) > 0 || m.HasTask() || m.HasWorkflow() || len(m.CallbackConfigs()) > 0 {
		imports[impErrors] = true
	}
	if m.HasTask() || m.HasWorkflow() {
		imports[impJSON] = true
		imports[impWorkflow] = true
	}
	for _, p := range srcPaths {
		imports[p] = true
	}
	if len(m.Configs) > 0 {
		imports[impCfg] = true
	}
	// A struct-valued config's getter/setter marshal JSON and its validator runs dv8;
	// the validation rule on the raw string, if declared, can only be "json".
	for _, c := range m.Configs {
		if !c.Scalar {
			if c.Validation != "" && c.Validation != "json" {
				return nil, fmt.Errorf("config %s: validation of a struct-valued config must be `json`, not `%s`", c.Name, c.Validation)
			}
			imports[impJSON] = true
			imports[impErrors] = true
			imports[impDV8] = true
		}
	}
	if intermediateNeedsStrconv(m) {
		imports[impStrconv] = true
	}
	// Only the kinds whose handlers are emitted here contribute In/Out field-type imports; outbound and
	// inbound events are excluded (inbound source types are added via inboundView's typePaths).
	for p := range featureSelectorImports(svc, map[string]bool{
		"Function": true, "Web": true, "Task": true, "Workflow": true,
	}) {
		imports[p] = true
	}
	// Config getters, metric recorders, ticker intervals, and time budgets render value-typed fragments;
	// resolve their selectors' packages against the api aliases, the same as In/Out fields.
	for _, c := range m.Configs {
		addResolved(imports, svc.imports, c.Type)
	}
	for _, mt := range m.Metrics {
		addResolved(imports, svc.imports, mt.ValueType)
	}
	for _, tk := range m.Tickers {
		addResolved(imports, svc.imports, tk.Interval)
	}
	for _, group := range [][]*featureView{m.Funcs, m.Webs, m.Tasks, m.Workflows} {
		for _, fv := range group {
			addResolved(imports, svc.imports, fv.TimeBudget)
		}
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

	var buf bytes.Buffer
	err = clientTemplate.ExecuteTemplate(&buf, "intermediate.txt", m)
	if err != nil {
		return nil, err
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n%s", err, numberLines(buf.Bytes()))
	}
	return out, nil
}

// endpointView builds a feature view augmented with the subscription options.
func endpointView(svc *service, f feature) *featureView {
	fv := newFeatureView(svc, f)
	fv.apiPkg = svc.apiPkg
	fv.ReqClaims = attrString(f.attrs, "RequiredClaims")
	tb, ok := f.attrs["TimeBudget"]
	if ok {
		fv.TimeBudget = exprSource(svc.fset, tb)
	}
	fv.Queue = loadBalancingValue(f.attrs["LoadBalancing"])
	fv.Manual = attrBool(f.attrs, "Manual")
	if tags := stringSlice(f.attrs["Tags"]); len(tags) > 0 {
		quoted := make([]string, len(tags))
		for i, t := range tags {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		fv.TagArgs = strings.Join(quoted, ", ")
	}
	return fv
}

// buildConfig builds the view for a config property.
func buildConfig(f feature) *configView {
	typ := carrierTypeName(f.attrs["Value"])
	scalar := typ == "string" || typ == "int" || typ == "float64" || typ == "bool" || typ == "time.Duration"
	return &configView{
		Name:       f.name,
		Doc:        f.doc,
		DocComment: docComment(f.doc),
		Type:       typ,
		Scalar:     scalar,
		Default:    attrString(f.attrs, "Default"),
		Validation: attrString(f.attrs, "Validation"),
		Secret:     attrBool(f.attrs, "Secret"),
		Callback:   attrBool(f.attrs, "Callback"),
	}
}

// buildMetric builds the view for a metric.
func buildMetric(svc *service, f feature) *metricView {
	kind := metricKind(f.attrs["Kind"])
	vt := carrierTypeName(f.attrs["Value"])
	if vt == "" {
		vt = "int"
	}
	valueArg := "float64(value)"
	if vt == "float64" {
		valueArg = "value"
	}
	verb, fn := "Record", "RecordGauge"
	switch kind {
	case "counter":
		verb, fn = "Increment", "IncrementCounter"
	case "histogram":
		verb, fn = "Record", "RecordHistogram"
	}
	buckets := ""
	if kind == "histogram" {
		buckets = exprSource(svc.fset, f.attrs["Buckets"])
	}
	return &metricView{
		Name:         f.name,
		Doc:          f.doc,
		DocComment:   docComment(f.doc),
		Kind:         kind,
		RecorderVerb: verb,
		RecordFn:     fn,
		OTelName:     attrString(f.attrs, "OTelName"),
		ValueType:    vt,
		ValueArg:     valueArg,
		Labels:       stringSlice(f.attrs["Labels"]),
		Buckets:      buckets,
		Observable:   attrBool(f.attrs, "Observable"),
	}
}

// buildTicker builds the view for a ticker.
func buildTicker(svc *service, f feature) *tickerView {
	return &tickerView{
		Name:       f.name,
		DocComment: docComment(f.doc),
		Interval:   exprSource(svc.fset, f.attrs["Interval"]),
	}
}

// intermediateNeedsStrconv reports whether any config getter parses an int/bool/float64.
func intermediateNeedsStrconv(m *intermediateModel) bool {
	for _, c := range m.Configs {
		if c.Type == "int" || c.Type == "bool" || c.Type == "float64" {
			return true
		}
	}
	return false
}

// inboundView resolves an inbound event's handler signature from its source api package and returns
// the view, the source api package's import path, and the import paths of any pkg.Type selectors in
// the source event's fields, resolved against the source package's own aliases.
func inboundView(svc *service, f feature, resolveSource func(string) (*service, error)) (*featureView, string, []string, error) {
	srcPath, ok := svc.imports[f.srcPkg]
	if !ok {
		return nil, "", nil, fmt.Errorf("inbound event %q: unknown source package %q", f.name, f.srcPkg)
	}
	srcSvc, err := resolveSource(srcPath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("inbound event %q: resolving source %s: %w", f.name, srcPath, err)
	}
	var ev *feature
	for i := range srcSvc.features {
		if srcSvc.features[i].kind == "OutboundEvent" && srcSvc.features[i].name == f.srcEvent {
			ev = &srcSvc.features[i]
			break
		}
	}
	if ev == nil {
		return nil, "", nil, fmt.Errorf("inbound event %q: source outbound event %s.%s not found", f.name, f.srcPkg, f.srcEvent)
	}
	inFields := srcSvc.fieldsOf(ev.in)
	outFields := srcSvc.fieldsOf(ev.out)
	typeImports := map[string]bool{}
	for _, fld := range append(append([]fieldDef{}, inFields...), outFields...) {
		addResolved(typeImports, srcSvc.imports, fld.typ)
	}
	var paths []string
	for p := range typeImports {
		paths = append(paths, p)
	}
	return &featureView{
		Name:        f.name,
		Doc:         f.doc,
		DocComment:  docComment(f.doc),
		In:          ev.in,
		SrcPkg:      f.srcPkg,
		SrcEvent:    f.srcEvent,
		HookOptions: hookOptions(svc, f),
		apiPkg:      f.srcPkg, // inbound handler params reference the source api package's types
		inFields:    inFields,
		outFields:   outFields,
	}, srcPath, paths, nil
}

// hookOptions renders the sub.Options for an inbound event's Hook.WithOptions call from its
// RequiredClaims/TimeBudget/LoadBalancing fields, comma-joined, or "" when none are set.
func hookOptions(svc *service, f feature) string {
	var opts []string
	if claims := attrString(f.attrs, "RequiredClaims"); claims != "" {
		opts = append(opts, fmt.Sprintf("sub.RequiredClaims(`%s`)", claims))
	}
	if tb, ok := f.attrs["TimeBudget"]; ok {
		opts = append(opts, fmt.Sprintf("sub.TimeBudget(%s)", exprSource(svc.fset, tb)))
	}
	switch q := loadBalancingValue(f.attrs["LoadBalancing"]); q {
	case "none":
		opts = append(opts, "sub.NoQueue()")
	case "default":
		opts = append(opts, "sub.DefaultQueue()")
	case "":
		// default queue, no option
	default:
		opts = append(opts, fmt.Sprintf("sub.Queue(%q)", q))
	}
	if attrBool(f.attrs, "Manual") {
		opts = append(opts, "sub.Manual()")
	}
	for _, t := range stringSlice(f.attrs["Tags"]) {
		opts = append(opts, fmt.Sprintf("sub.Tag(%q)", t))
	}
	return strings.Join(opts, ", ")
}

