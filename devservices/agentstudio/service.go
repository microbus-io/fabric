package agentstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/coreservices/control/controlapi"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/openapi"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/connector"

	bespa "github.com/microbus-io/bespa"
	"github.com/microbus-io/bespa/chart"
	"github.com/microbus-io/bespa/mermaid"
	"github.com/microbus-io/bespa/widget"

	"github.com/microbus-io/fabric/devservices/agentstudio/agentstudioapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ *workflow.FlowOutcome
	_ agentstudioapi.Client
)

// wf is the widget factory used to construct page widgets. It combines the
// default bespa factories with the optional mermaid and chart factories.
var wf = struct {
	bespa.DefaultFactory
	mermaid.MermaidFactory
	chart.ChartFactory
}{}

// listLimit is the maximum number of flows fetched per page from the foreman.
// Pagination within the table is done in-memory by the bespa Table widget.
const listLimit = 500

/*
Service implements agentstudio.dev which serves a developer UI for inspecting flows.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	foreman foremanapi.Client
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	if svc.Deployment() != connector.LOCAL && svc.Deployment() != connector.TESTING {
		return errors.New("agentstudio runs only in LOCAL deployment", "deployment", svc.Deployment())
	}
	svc.foreman = foremanapi.NewClient(svc)
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

// navBar returns the navigation menu shared by every agentstudio page. It
// implements NavAreaMarker via MainMenu, so the enclosing Page auto-slots it
// into the navigation area regardless of insertion order. Each modality
// (rail / drawer / strip) gets a freshly built set of NavTargets per the
// bespa pattern - widgets are stateful and should not be shared across
// modalities. Icon names are Material Symbols ligatures.
func (svc *Service) navBar(r *http.Request) widget.Widget {
	url := func(path string) string {
		return svc.ExternalizeURL(r.Context(), path)
	}
	rail := wf.NavRail().AddTop(
		wf.NavTarget("Workflows", url("/workflows")).WithIcon("flowchart"),
		wf.NavTarget("Flows", url("/flows")).WithIcon("automation"),
		wf.NavTarget("Dashboard", url("/dashboard")).WithIcon("dashboard"),
	)
	drawer := wf.NavDrawer().AddTop(
		wf.NavTarget("Workflows", url("/workflows")).WithIcon("flowchart"),
		wf.NavTarget("Flows", url("/flows")).WithIcon("automation"),
		wf.NavTarget("Dashboard", url("/dashboard")).WithIcon("dashboard"),
	)
	strip := wf.NavStrip().AddTop(
		wf.NavTarget("Agent studio", url("/flows")).WithIcon("home"),
	)
	return wf.MainMenu().
		WithRail(rail).
		WithVertical(drawer).
		WithHorizontal(strip)
}

/*
ListWorkflows renders an HTML page listing the workflows available in the system.
*/
func (svc *Service) ListWorkflows(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: ListWorkflows
	type wfRow struct {
		host string
		name string
		url  string
		desc string
	}
	var rows []wfRow
	for resp := range controlapi.NewMulticastClient(svc).ForHost("all").OpenAPI(r.Context()) {
		doc, status, callErr := resp.Get()
		if callErr != nil {
			svc.LogWarn(r.Context(), "OpenAPI fetch failed for one peer", "error", callErr)
			continue
		}
		if status != 0 && status != http.StatusOK {
			continue
		}
		if doc == nil {
			continue
		}
		host := doc.Info.Title
		for path, methods := range doc.Paths {
			for _, op := range methods {
				if op == nil || op.XFeatureType != openapi.FeatureWorkflow {
					continue
				}
				desc := op.Description
				if desc == "" {
					desc = op.Summary
				}
				rows = append(rows, wfRow{
					host: host,
					name: op.XName,
					url:  "https:/" + path,
					desc: desc,
				})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].host != rows[j].host {
			return rows[i].host < rows[j].host
		}
		return rows[i].name < rows[j].name
	})

	tbl := wf.Table().
		WithDefaultPageRows(r, 50).
		Add(
			wf.Col("nwx", 28, "left").Add("Workflow"),
			wf.Col("wx", 40, "left").Add("Description"),
		)
	tbl.WithTotalRows(r, len(rows))
	from, to := tbl.DisplayRange(r)
	for _, row := range rows[from:to] {
		detailHref := svc.ExternalizeURL(r.Context(), "/workflows/"+strings.TrimPrefix(row.url, "https://"))
		tbl.Add(wf.Row().Add(
			wf.Collection(
				wf.Link(detailHref).Add(row.name),
				wf.HTMLUnsafe("<br>"),
				wf.TextStyle(row.host).WithColorDeemphasized(),
			),
			wf.Text(row.desc),
		))
	}

	page := wf.Page().Add(
		wf.AppBar("Workflows"),
		svc.navBar(r),
		wf.Toolbar().AddRight(wf.Paginator()),
		tbl,
		wf.Toolbar().AddLeft(wf.PageSizer()),
	)
	return page.Draw(w, r)
}

/*
WorkflowDetail renders an HTML page with the structure and definition of a single workflow graph.
*/
func (svc *Service) WorkflowDetail(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: WorkflowDetail
	raw := r.PathValue("workflowURL")
	if raw == "" {
		return errors.New("workflowURL is required", http.StatusBadRequest)
	}
	workflowURL := "https://" + raw

	res, err := svc.Request(r.Context(), pub.Method("GET"), pub.URL(workflowURL))
	if err != nil {
		return errors.Trace(err)
	}
	var wrapper struct {
		Graph *workflow.Graph `json:"graph"`
	}
	err = json.NewDecoder(res.Body).Decode(&wrapper)
	if err != nil {
		return errors.Trace(err)
	}
	graph := wrapper.Graph

	overview := wf.Form().Add(
		wf.Field().AddLeft("URL").AddRight(wf.Text(workflowURL)),
	)
	if graph != nil {
		if name := graph.Name(); name != "" {
			overview.Add(wf.Field().AddLeft("Name").AddRight(wf.Text(name)))
		}
		if entry := graph.EntryPoint(); entry != "" {
			overview.Add(wf.Field().AddLeft("Entry point").AddRight(wf.Text(entry)))
		}
	}

	var flowchartBody any = wf.Text("(graph unavailable)")
	if graph != nil {
		mmd, rerr := workflow.NewGraphRenderer(graph).
			WithPrimaryColors(mermaid.PrimaryContainer, mermaid.OnPrimaryContainer).
			WithSecondaryColors(mermaid.SecondaryContainer, mermaid.OnSecondaryContainer).
			WithLinks("task").
			Render()
		if rerr == nil {
			flowchartBody = wf.Mermaid(mmd).WithZoomPan(true).WithHeight("calc(100vh - 120px)")
		}
	}

	if !wf.StateOf(r).Has("workflowTab") {
		wf.StateOf(r).Set("workflowTab", "flowchart")
	}
	tabLabels := wf.TabSwitcher().WithName("workflowTab").AddLeft(
		wf.TabLabel("overview").Add("Overview"),
		wf.TabLabel("flowchart").Add("Flowchart"),
	)
	tabBodies := wf.TabSwitcher().WithName("workflowTab").AddLeft(
		wf.TabBody("overview").Add(overview),
		wf.TabBody("flowchart").Add(flowchartBody),
	)

	taskName := wf.StateOf(r).Get("task")
	embedURL := ""
	if taskName != "" {
		q := url.Values{
			"workflow": {workflowURL},
			"task":     {taskName},
			"_back":    {"^?task="},
		}
		embedURL = "/task-detail?" + q.Encode()
	}
	taskPanel := wf.SidePanel("task").WithWidth("420px").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.TaskDetail(w, r)
		}, r, "GET", embedURL, nil),
	)

	runEmbedURL := ""
	if wf.StateOf(r).Get("run") != "" {
		q := url.Values{
			"workflow": {workflowURL},
			"_back":    {"^?run="},
		}
		runEmbedURL = "/run-workflow?" + q.Encode()
	}
	runModal := wf.Modal("run").WithMinHeight("").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.RunWorkflow(w, r)
		}, r, "GET", runEmbedURL, nil),
	)

	appBar := wf.AppBar(workflowURL).AddBottom(tabLabels)
	// flowsHref := svc.ExternalizeURL(r.Context(), "/flows?workflow="+url.QueryEscape(workflowURL))
	appBar.AddRight(
		// wf.ButtonText("flows").WithHref(flowsHref).Add(wf.Icon("automation"), "Flows"),
		wf.ButtonText("run").WithHref("?run=1").Add(wf.Icon("play_arrow"), "Run"),
	)

	page := wf.Page().Add(
		appBar,
		svc.navBar(r),
		tabBodies,
		taskPanel,
		runModal,
	)
	return page.Draw(w, r)
}

/*
TaskDetail renders an HTML page with the metadata of a single task in a workflow graph.
*/
func (svc *Service) TaskDetail(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: TaskDetail
	workflowURL := r.URL.Query().Get("workflow")
	taskName := r.URL.Query().Get("task")
	if workflowURL == "" || taskName == "" {
		return errors.New("workflow and task are required", http.StatusBadRequest)
	}

	res, err := svc.Request(r.Context(), pub.Method("GET"), pub.URL(workflowURL))
	if err != nil {
		return errors.Trace(err)
	}
	var wrapper struct {
		Graph *workflow.Graph `json:"graph"`
	}
	err = json.NewDecoder(res.Body).Decode(&wrapper)
	if err != nil {
		return errors.Trace(err)
	}
	graph := wrapper.Graph

	form := wf.Form()
	var taskURL string
	if graph != nil {
		taskURL = graph.URLOf(taskName)
	}
	if taskURL == "" || taskURL == workflow.END {
		form.Add(wf.Field().AddLeft("Status").AddRight(wf.Text("(task not registered on the graph)")))
	} else {
		form.Add(wf.Field().AddLeft("URL").AddRight(wf.Text(taskURL)))
		// Resolve description via the hosting microservice's :888/openapi.json.
		rest := strings.TrimPrefix(taskURL, "https://")
		host, _, _ := strings.Cut(rest, "/")
		host, _, _ = strings.Cut(host, ":")
		doc, status, fetchErr := controlapi.NewClient(svc).ForHost(host).OpenAPI(r.Context())
		if fetchErr == nil && (status == 0 || status == http.StatusOK) && doc != nil {
			docPath := "/" + rest
			for _, op := range doc.Paths[docPath] {
				if op == nil {
					continue
				}
				if op.Summary != "" {
					form.Add(wf.Field().AddLeft("Summary").AddRight(wf.Text(op.Summary)))
				}
				if op.Description != "" {
					form.Add(wf.Field().AddLeft("Description").AddRight(wf.Markdown(op.Description)))
				}
				break
			}
		}
	}

	page := wf.Page().Add(
		wf.AppBar(taskName),
		form,
	)
	return page.Draw(w, r)
}

/*
Dashboard renders an HTML page with operator dashboards for flows and workflows.
*/
func (svc *Service) Dashboard(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Dashboard
	window := dashboardWindow(wf.StateOf(r).Get("window"))

	flows, _, err := svc.foreman.List(r.Context(), foremanapi.Query{
		NewerThan: window,
		Limit:     500,
	})
	if err != nil {
		return errors.Trace(err)
	}

	// Bound the History fan-out so a busy environment doesn't blow up the page.
	histFlows := flows
	if len(histFlows) > dashboardHistoryLimit {
		histFlows = histFlows[:dashboardHistoryLimit]
	}
	histories := make([][]foremanapi.FlowStep, len(histFlows))
	jobs := make([]func(ctx context.Context) error, len(histFlows))
	for i := range histFlows {
		fk := histFlows[i].FlowKey
		jobs[i] = func(ctx context.Context) error {
			steps, herr := svc.foreman.History(ctx, fk)
			if herr == nil {
				histories[i] = steps
			}
			return nil
		}
	}
	if len(jobs) > 0 {
		_ = svc.Parallel(r.Context(), jobs...)
	}

	windowDropdown := wf.Dropdown("window", windowKey(window)).WithAutoSubmit(true).WithRequired(true).
		AddOption("8h", "Last 8 hours").
		AddOption("24h", "Last 24 hours")

	statusChart := buildStatusTimelineChart(flows, window)
	errorChart := buildErrorTimelineChart(flows, window)
	heatmapChart := buildTaskHeatmapChart(histories, window)
	boxplotChart := buildTaskBoxplotChart(histories)

	page := wf.Page().Add(
		wf.AppBar("Dashboard"),
		svc.navBar(r),
		wf.Toolbar().AddLeft(windowDropdown).RedrawIfChanged(r, "window"),
		wf.Splitter(1, 1).
			AddToCol(0, statusChart).
			AddToCol(1, errorChart).
			RedrawIfChanged(r, "window"),
		heatmapChart.RedrawIfChanged(r, "window"),
		boxplotChart.RedrawIfChanged(r, "window"),
	)
	return page.Draw(w, r)
}

/*
ListFlows renders an HTML page with a paginated, sortable table of flows.
*/
func (svc *Service) ListFlows(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: ListFlows
	tbl := wf.Table().
		WithDefaultPageRows(r, 25).
		Add(
			wf.Col("wx", 14, "left").Add("Created"),
			wf.Col("n", 8, "left").Add("Created"),
			wf.Col("nwx", 28, "left").Add("Flow"),
			wf.Col("wx", 22, "left").Add("Status"),
			wf.Col("nwx", 10, "right").Add("Duration"),
			wf.Col("x", 22, "left").Add("Error / cancel reason"),
		)

	statusFilter := wf.StateOf(r).Get("status")
	statusDropdown := wf.Dropdown("status", statusFilter).WithAutoSubmit(true).
		AddOption("", "All statuses").
		AddOption(workflow.StatusPending, workflow.StatusPending).
		AddOption(workflow.StatusRunning, workflow.StatusRunning).
		AddOption(workflow.StatusInterrupted, workflow.StatusInterrupted).
		AddOption(workflow.StatusCompleted, workflow.StatusCompleted).
		AddOption(workflow.StatusFailed, workflow.StatusFailed).
		AddOption(workflow.StatusCancelled, workflow.StatusCancelled)

	flows, _, err := svc.foreman.List(r.Context(), foremanapi.Query{
		WorkflowURL: strings.TrimSpace(r.URL.Query().Get("workflow")),
		Status:      statusFilter,
		Search:      strings.TrimSpace(tbl.Query(r)),
		Limit:       listLimit,
	})
	if err != nil {
		return errors.Trace(err)
	}

	tbl.WithTotalRows(r, len(flows))
	from, to := tbl.DisplayRange(r)
	for _, f := range flows[from:to] {
		href := svc.ExternalizeURL(r.Context(), "/flows/"+url.PathEscape(f.FlowKey))
		errCell := f.Error
		if errCell == "" {
			errCell = f.CancelReason
		}
		statusCell := wf.Collection(statusChip(f.Status))
		if f.TaskName != "" && f.Status != workflow.StatusCompleted {
			statusCell.Add(" at ", wf.QuickSearchUnderliner(f.TaskName))
		}
		tbl.Add(wf.Row().Add(
			wf.DateTime(f.CreatedAt),
			wf.DateOrTime(f.CreatedAt),
			wf.Collection(
				wf.Link(href).Add(wf.QuickSearchUnderliner(f.FlowKey)),
				wf.HTMLUnsafe("<br>"),
				wf.TextStyle(f.WorkflowName).WithColorDeemphasized(),
			),
			statusCell,
			wf.Duration(f.Duration()),
			wf.QuickSearchUnderliner(truncate(errCell, 80)),
		))
	}

	page := wf.Page().Add(
		wf.AppBar("Flows"),
		svc.navBar(r),
		wf.Toolbar().
			AddLeft(wf.QuickSearch(), statusDropdown).
			AddRight(wf.Paginator()).
			RedrawIfChanged(r, "status"),
		tbl.RedrawIfChanged(r, "status"),
		wf.Toolbar().AddLeft(wf.PageSizer()).RedrawIfChanged(r, "status"),
	)
	return page.Draw(w, r)
}

/*
FlowDetail renders an HTML page with the details, DAG diagram, and step log of a flow.
*/
func (svc *Service) FlowDetail(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: FlowDetail
	flowKey := r.PathValue("flowKey")
	if flowKey == "" {
		return errors.New("flowKey is required", http.StatusBadRequest)
	}

	if wf.StateOf(r).Get("cancel") != "" {
		cancelErr := svc.foreman.Cancel(r.Context(), flowKey, "cancelled from agentstudio")
		if cancelErr != nil {
			svc.LogWarn(r.Context(), "Cancel failed", "flowKey", flowKey, "error", cancelErr)
		}
		wf.StateOf(r).Del("cancel")
	}

	// Capture the poll baseline before reading the snapshot/history that render the
	// graph, so `since` can never be newer than the rendered state.
	initialToken, _, _ := svc.foreman.Fingerprint(r.Context(), flowKey)

	outcome, err := svc.foreman.Snapshot(r.Context(), flowKey)
	if err != nil {
		return errors.Trace(err)
	}
	steps, err := svc.foreman.History(r.Context(), flowKey)
	if err != nil {
		return errors.Trace(err)
	}

	var createdAt, startedAt, updatedAt time.Time
	if len(steps) > 0 {
		createdAt = steps[0].CreatedAt
		// StartedAt of the entry step is when this attempt actually began dispatching.
		// Falls back to CreatedAt for steps that never started.
		if steps[0].HasStarted() && !steps[0].StartedAt.IsZero() {
			startedAt = steps[0].StartedAt
		} else {
			startedAt = steps[0].CreatedAt
		}
		updatedAt = steps[len(steps)-1].UpdatedAt
	}
	stepCount := countSteps(steps)

	overview := wf.Form().Add(
		wf.Field().AddLeft("Flow key").AddRight(wf.Text(flowKey)),
		wf.Field().AddLeft("Status").AddRight(statusChip(statusOf(outcome))),
	)
	if errStr := errorOf(outcome); errStr != "" {
		overview.Add(wf.Field().AddLeft("Error").AddRight(wf.Text(errStr)))
	}
	if reason := cancelReasonOf(outcome); reason != "" {
		overview.Add(wf.Field().AddLeft("Reason").AddRight(wf.Text(reason)))
	}
	overview.Add(
		wf.Field().AddLeft("Steps").AddRight(wf.Text(fmt.Sprintf("%d", stepCount))),
		wf.Field().AddLeft("Created").AddRight(wf.DateTime(createdAt)),
		wf.Field().AddLeft("Duration").AddRight(wf.Duration(updatedAt.Sub(startedAt))),
	)

	mmd, mmdErr := workflow.NewFlowRenderer(steps).
		WithPrimaryColors(mermaid.PrimaryContainer, mermaid.OnPrimaryContainer).
		WithSecondaryColors(mermaid.SecondaryContainer, mermaid.OnSecondaryContainer).
		WithErrorColors(mermaid.ErrorContainer, mermaid.OnErrorContainer).
		WithAttentionColors(mermaid.TertiaryContainer, mermaid.OnTertiaryContainer).
		WithLinks("step").
		Render()
	if mmdErr != nil {
		mmd = "flowchart TD\n  err[\"(history unavailable)\"]"
	}
	mmdWidget := wf.Mermaid(mmd).WithZoomPan(true).WithHeight("calc(100vh - 120px)").RedrawIfChanged(r, "flowrefresh")
	// Seed flowstopped so a flow opened in a terminal state renders the bar
	// already hidden — Tag.When(false) emits a stable empty placeholder, no
	// polling kicks off, no first-poll flicker.
	if !isLiveStatus(statusOf(outcome)) {
		wf.StateOf(r).Set("flowstopped", "1")
	}
	pollURL := svc.ExternalizeURL(r.Context(), "/poll-flow") +
		"?flow=" + url.QueryEscape(flowKey) +
		"&since=" + url.QueryEscape(initialToken)
	progressBar := wf.Progress().
		WithMax(1).
		WithValue(-1).
		WithRefreshURL(pollURL).
		WithRefreshInterval(250*time.Millisecond).
		WithWidth("100%").
		HideIfEq(r, "flowstopped", "1").
		RedrawIfChanged(r, "flowstopped")

	// Input/Output tabs need the first and last step's state, which History
	// intentionally omits. Fetch via foreman.Step (one or two extra round
	// trips). Skip silently for an empty flow.
	var inputForm, outputForm any
	if len(steps) > 0 {
		firstStep, err := svc.foreman.Step(r.Context(), steps[0].StepKey)
		if err == nil && firstStep != nil {
			inputForm = renderStateForm(r, "expFlowIn", firstStep.State, nil)
		}
		lastKey := steps[len(steps)-1].StepKey
		lastStep := firstStep
		if lastKey != steps[0].StepKey {
			lastStep, err = svc.foreman.Step(r.Context(), lastKey)
			if err != nil {
				lastStep = nil
			}
		}
		if lastStep != nil {
			merged := make(map[string]any, len(lastStep.State)+len(lastStep.Changes))
			for k, v := range lastStep.State {
				merged[k] = v
			}
			for k, v := range lastStep.Changes {
				merged[k] = v
			}
			outputForm = renderStateForm(r, "expFlowOut", merged, nil)
		}
	}

	logTbl := wf.Table().
		WithName("log").
		WithDefaultPageRows(r, 50).
		Add(
			wf.Col("nwx", 8, "right").Add("T+"),
			wf.Col("nwx", 10, "left").Add("Status"),
			wf.Col("nwx", 28, "left").Add("Task"),
			wf.Col("x", 24, "left").Add("Error"),
		)
	flat := flattenSteps(steps, 0)
	var origin time.Time
	if len(steps) > 0 {
		origin = steps[0].CreatedAt
	}
	logTbl.WithTotalRows(r, len(flat))
	from, to := logTbl.DisplayRange(r)
	for _, fs := range flat[from:to] {
		taskCell := wf.Link("?step=" + fs.step.StepKey).Add(strings.Repeat("    ", fs.depth) + fs.step.TaskName)
		logTbl.Add(wf.Row().Add(
			formatDelta(fs.step.UpdatedAt, origin),
			statusChip(fs.step.Status),
			taskCell,
			truncate(fs.step.Error, 80),
		))
	}

	// Seed the flow-tab state so the labels switcher (no WithSelected) and the
	// bodies switcher agree on the initial render. Default to the DAG tab.
	if !wf.StateOf(r).Has("flowTab") {
		wf.StateOf(r).Set("flowTab", "dag")
	}
	tabLabels := wf.TabSwitcher().WithName("flowTab").AddLeft(
		wf.TabLabel("overview").Add("Overview"),
		wf.TabLabel("dag").Add("DAG"),
		wf.TabLabel("log").Add("Log"),
	)
	tabBodies := wf.TabSwitcher().WithName("flowTab").AddLeft(
		wf.TabBody("overview").Add(overview),
		wf.TabBody("dag").Add(mmdWidget, progressBar),
		wf.TabBody("log").Add(
			logTbl,
			wf.Toolbar().AddLeft(wf.Paginator().ForTable("log")).AddRight(wf.PageSizer().ForTable("log")),
		),
	)
	if inputForm != nil {
		tabLabels.AddLeft(wf.TabLabel("input").Add("Input"))
		tabBodies.AddLeft(wf.TabBody("input").Add(inputForm))
	}
	if outputForm != nil {
		tabLabels.AddLeft(wf.TabLabel("output").Add("Output"))
		tabBodies.AddLeft(wf.TabBody("output").Add(outputForm))
	}

	stepKey := wf.StateOf(r).Get("step")
	embedURL := ""
	if stepKey != "" {
		q := url.Values{
			"_back": {"^?step="},
			"flow":  {flowKey},
		}
		embedURL = "/steps/" + url.PathEscape(stepKey) + "?" + q.Encode()
	}
	stepModal := wf.SidePanel("step").WithWidth("420px").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("stepKey", stepKey)
			_ = svc.StepDetail(w, r)
		}, r, "GET", embedURL, nil),
	)

	continueEmbedURL := ""
	if wf.StateOf(r).Get("continue") != "" {
		q := url.Values{
			"flow":  {flowKey},
			"_back": {"^?continue="},
		}
		continueEmbedURL = "/continue-flow?" + q.Encode()
	}
	continueModal := wf.Modal("continue").WithMinHeight("").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.ContinueFlow(w, r)
		}, r, "GET", continueEmbedURL, nil),
	)

	resumeEmbedURL := ""
	if wf.StateOf(r).Get("resume") != "" {
		q := url.Values{
			"flow":  {flowKey},
			"_back": {"^?resume="},
		}
		resumeEmbedURL = "/resume-flow?" + q.Encode()
	}
	resumeModal := wf.Modal("resume").WithMinHeight("").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.ResumeFlow(w, r)
		}, r, "GET", resumeEmbedURL, nil),
	)

	// The flow-level fork re-runs the whole flow as a new self-contained flow by
	// forking from its entry step (the original is immutable). The DAG's per-step
	// fork affordance handles forking from any later step.
	forkEmbedURL := ""
	if wf.StateOf(r).Get("fork") != "" && len(steps) > 0 {
		q := url.Values{
			"step":  {steps[0].StepKey},
			"flow":  {flowKey},
			"_back": {"^?fork="},
		}
		forkEmbedURL = "/fork-from-step?" + q.Encode()
	}
	forkModal := wf.Modal("fork").WithMinHeight("").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.ForkFromStep(w, r)
		}, r, "GET", forkEmbedURL, nil),
	)

	// Render every status-dependent button unconditionally and gate visibility
	// with HideIf so each button keeps a stable data-id across renders. With
	// RedrawIfChanged(r, "flowrefresh") the poll-driven page navigation swaps
	// just the buttons whose visibility flipped, not the whole appbar.
	cancellable := func(status string) bool {
		switch status {
		case workflow.StatusCreated, workflow.StatusPending, workflow.StatusRunning, workflow.StatusInterrupted:
			return true
		}
		return false
	}
	currentStatus := statusOf(outcome)
	appBar := wf.AppBar("Flow " + flowKey).AddBottom(tabLabels)
	appBar.AddRight(wf.ButtonText("resume").
		WithHref("?resume=1").
		Add(wf.Icon("play_arrow"), "Resume").
		HideIf(currentStatus != workflow.StatusInterrupted).
		RedrawIfChanged(r, "flowrefresh"))
	appBar.AddRight(wf.ButtonText("cancel").
		WithHref("?cancel=1").
		Add(wf.Icon("cancel"), "Cancel").
		HideIf(!cancellable(currentStatus)).
		RedrawIfChanged(r, "flowrefresh"))
	appBar.AddRight(wf.ButtonText("continue").
		WithHref("?continue=1").
		WithDisabled(!svc.threadIsContinuable(r, flowKey)).
		Add(wf.Icon("play_arrow"), "Continue").
		HideIf(cancellable(currentStatus)).
		RedrawIfChanged(r, "flowrefresh"))
	appBar.AddRight(wf.ButtonText("fork").
		WithHref("?fork=1").
		Add(wf.Icon("fork_right"), "Fork").
		HideIf(!isForkableStatus(currentStatus)).
		RedrawIfChanged(r, "flowrefresh"))

	page := wf.Page().Add(
		appBar,
		svc.navBar(r),
		tabBodies,
		stepModal,
		continueModal,
		resumeModal,
		forkModal,
	)
	return page.Draw(w, r)
}

/*
StepDetail renders an HTML page with the details of one execution step.
*/
func (svc *Service) StepDetail(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: StepDetail
	stepKey := r.PathValue("stepKey")
	if stepKey == "" {
		return errors.New("stepKey is required", http.StatusBadRequest)
	}
	step, err := svc.foreman.Step(r.Context(), stepKey)
	if err != nil {
		return errors.Trace(err)
	}

	// Overview tab: high-level metadata. Errors render inline under Status; the
	// Interrupt payload still gets its own tab because it is a structured map.
	overview := wf.Form().Add(
		wf.Field().AddLeft("Step key").AddRight(wf.Text(step.StepKey)),
		wf.Field().AddLeft("Status").AddRight(statusChip(step.Status)),
	)
	if step.Error != "" {
		overview.Add(wf.Field().AddLeft("Error").AddRight(wf.Text(step.Error)))
	}
	overview.Add(
		wf.Field().AddLeft("Depth").AddRight(wf.Text(fmt.Sprintf("%d", step.StepDepth))),
		wf.Field().AddLeft("Created").AddRight(wf.DateTime(step.CreatedAt)),
	)
	if step.Attempt > 0 {
		overview.Add(wf.Field().AddLeft("Attempt").AddRight(wf.Text(fmt.Sprintf("%d", step.Attempt))))
	}
	if step.HasStarted() && !step.StartedAt.IsZero() && !step.UpdatedAt.IsZero() {
		overview.Add(wf.Field().AddLeft("Duration").AddRight(
			wf.Text(strings.TrimPrefix(formatDelta(step.UpdatedAt, step.StartedAt), "+")),
		))
	}

	// Input tab: state snapshot as the task saw it on entry. One row per key.
	inputForm := wf.Form()
	inputKeys := make([]string, 0, len(step.State))
	for k := range step.State {
		inputKeys = append(inputKeys, k)
	}
	sort.Strings(inputKeys)
	for _, k := range inputKeys {
		inputForm.Add(stateField(r, stateKeyFor("expIn", k), k, jsonValueString(step.State[k], true), func(s string) any {
			return wf.Text(s)
		}))
	}

	// Output tab: merged final state (state + changes). Keys carried over from
	// the input render dimmed; keys actually set/mutated by this step render in
	// regular color. Only shown when the step has produced at least one change
	// or reached a status where its output is meaningful.
	terminal := func(status string) bool {
		switch status {
		case workflow.StatusCompleted, workflow.StatusFailed, workflow.StatusCancelled, workflow.StatusInterrupted:
			return true
		}
		return false
	}
	showOutput := len(step.Changes) > 0 || terminal(step.Status)
	var outputForm any
	if showOutput {
		form := wf.Form()
		outKeys := mergedSortedKeys(step.State, step.Changes)
		for _, k := range outKeys {
			val, inChanges := step.Changes[k]
			if !inChanges {
				val = step.State[k]
			}
			styled := func(s string) any { return wf.Text(s) }
			if !inChanges {
				styled = func(s string) any { return wf.TextStyle(s).WithColorDeemphasized() }
			}
			form.Add(stateField(r, stateKeyFor("expOut", k), k, jsonValueString(val, true), styled))
		}
		outputForm = form
	}

	defaultTab := "input"
	if showOutput {
		defaultTab = "output"
	}

	// Two TabSwitchers sharing the "stepTab" state: one in the AppBar with only
	// labels, one in the page body with only the matching bodies. WithSelected
	// is set on the body switcher (where the bodies live) since that's the one
	// that owns the default-selected key.
	var interruptForm any
	if len(step.InterruptPayload) > 0 {
		f := wf.Form()
		intrKeys := make([]string, 0, len(step.InterruptPayload))
		for k := range step.InterruptPayload {
			intrKeys = append(intrKeys, k)
		}
		sort.Strings(intrKeys)
		for _, k := range intrKeys {
			f.Add(stateField(r, stateKeyFor("expIntr", k), k, jsonValueString(step.InterruptPayload[k], true), func(s string) any {
				return wf.Text(s)
			}))
		}
		interruptForm = f
	}

	// The labels switcher and the bodies switcher each pick the "first added"
	// when the state variable is unset, which makes them disagree on the
	// initial render. Seed the state to defaultTab so both render the same
	// selected tab on the first paint.
	if !wf.StateOf(r).Has("stepTab") {
		wf.StateOf(r).Set("stepTab", defaultTab)
	}
	tabLabels := wf.TabSwitcher().WithName("stepTab").AddLeft(
		wf.TabLabel("overview").Add("Overview"),
		wf.TabLabel("input").Add("Input"),
	)
	tabBodies := wf.TabSwitcher().WithName("stepTab").AddLeft(
		wf.TabBody("overview").Add(overview),
		wf.TabBody("input").Add(inputForm),
	)
	if showOutput {
		tabLabels.AddLeft(wf.TabLabel("output").Add("Output"))
		tabBodies.AddLeft(wf.TabBody("output").Add(outputForm))
	}
	if interruptForm != nil {
		tabLabels.AddLeft(wf.TabLabel("interrupt").Add("Interrupt"))
		tabBodies.AddLeft(wf.TabBody("interrupt").Add(interruptForm))
	}

	appBar := wf.AppBar(step.TaskName).AddBottom(tabLabels)
	if isForkableStatus(step.Status) {
		appBar.AddRight(wf.ButtonText("fork").
			WithHref("?fork=1").
			Add(wf.Icon("fork_right"), "Fork"))
	}

	parentFlowKey := strings.TrimSpace(r.URL.Query().Get("flow"))
	forkEmbedURL := ""
	if wf.StateOf(r).Get("fork") != "" {
		q := url.Values{
			"step":  {stepKey},
			"flow":  {parentFlowKey},
			"_back": {"^?fork="},
		}
		forkEmbedURL = "/fork-from-step?" + q.Encode()
	}
	forkModal := wf.Modal("fork").WithMinHeight("").Add(
		wf.EmbedHandler(func(w http.ResponseWriter, r *http.Request) {
			_ = svc.ForkFromStep(w, r)
		}, r, "GET", forkEmbedURL, nil),
	)

	page := wf.Page().Add(
		appBar,
		tabBodies,
		forkModal,
	)
	return page.Draw(w, r)
}

/*
Assets serves the bespa CSS and JavaScript assets at /bespa/.
*/
func (svc *Service) Assets(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Assets
	widget.AssetRegistry.ServeHTTP(w, r)
	return nil
}

/*
RunWorkflow renders a form to create and start a workflow with an initial state and FlowOptions.
*/
func (svc *Service) RunWorkflow(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: RunWorkflow
	state := wf.StateOf(r)
	workflowURL := strings.TrimSpace(state.Get("workflow"))
	if workflowURL == "" {
		return errors.New("workflow is required", http.StatusBadRequest)
	}

	stateField := wf.InputText("state", "").
		WithRows(15).
		WithWidth("100%").
		WithTrimSpaces(false)

	runButton := wf.ButtonFilled("submit").Add("Run")
	cancelButton := wf.ButtonText("").WithHref("^?run=").Add("Cancel")

	form := wf.Form().Add(
		wf.Field().AddLeft("Initial state").AddRight(stateField),
		cancelButton,
		runButton,
	)

	var errMsg string
	if form.ReadyToCommit(r) {
		initialState, parseErr := parseStateRaw(stateField.Value(r))
		if parseErr != nil {
			errMsg = parseErr.Error()
		} else {
			flowKey, createErr := svc.foreman.Create(r.Context(), workflowURL, initialState, nil)
			if createErr == nil {
				wf.Redirect(w, r, "~/"+Hostname+"/flows/"+url.PathEscape(flowKey))
				return nil
			}
			errMsg = createErr.Error()
		}
	}

	var errBanner any
	if errMsg != "" {
		errBanner = wf.Snackbar().Add(errMsg).RedrawIf(true)
	}
	page := wf.Page().Add(
		wf.AppBar("Run workflow"),
		errBanner,
		form,
	)
	return page.Draw(w, r)
}

/*
ContinueFlow renders a form to continue a completed flow's thread with additional state.
*/
func (svc *Service) ContinueFlow(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: ContinueFlow
	state := wf.StateOf(r)
	threadKey := strings.TrimSpace(state.Get("flow"))
	if threadKey == "" {
		return errors.New("flow is required", http.StatusBadRequest)
	}

	stateField := wf.InputText("state", "").
		WithRows(15).
		WithWidth("100%").
		WithTrimSpaces(false)

	continueButton := wf.ButtonFilled("submit").Add("Continue")
	cancelButton := wf.ButtonText("").WithHref("^?continue=").Add("Cancel")

	form := wf.Form().Add(
		wf.Field().AddLeft("Additional state").AddRight(stateField),
		cancelButton,
		continueButton,
	)

	var errMsg string
	if form.ReadyToCommit(r) {
		additionalState, parseErr := parseStateRaw(stateField.Value(r))
		if parseErr != nil {
			errMsg = parseErr.Error()
		} else {
			newFlowKey, contErr := svc.foreman.Continue(r.Context(), threadKey, additionalState)
			if contErr == nil {
				wf.Redirect(w, r, "~/"+Hostname+"/flows/"+url.PathEscape(newFlowKey))
				return nil
			}
			errMsg = contErr.Error()
		}
	}

	var errBanner any
	if errMsg != "" {
		errBanner = wf.TextStyle(errMsg).WithColorError()
	}
	page := wf.Page().Add(
		wf.AppBar("Continue flow"),
		errBanner,
		form,
	)
	return page.Draw(w, r)
}

/*
PollFlow returns a JSON status payload driving the FlowDetail live-update progress bar.
*/
func (svc *Service) PollFlow(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: PollFlow
	flowKey := strings.TrimSpace(r.URL.Query().Get("flow"))
	if flowKey == "" {
		return errors.New("flow is required", http.StatusBadRequest)
	}
	since := strings.TrimSpace(r.URL.Query().Get("since"))

	deadline := time.Now().Add(pollFlowMaxWait)
	var (
		token  string
		status string
		stop   bool
	)
	for {
		fp, st, fpErr := svc.foreman.Fingerprint(r.Context(), flowKey)
		if fpErr != nil {
			token = ""
			status = ""
			stop = true
			break
		}
		token = fp
		status = st
		stop = !isLiveStatus(status)
		if token != since || stop || time.Now().After(deadline) {
			break
		}
		select {
		case <-time.After(pollFlowTick):
		case <-r.Context().Done():
			return nil
		}
	}

	value := -1
	if stop {
		value = 0
	}
	resp := struct {
		Value  int    `json:"value"`
		Stop   bool   `json:"stop"`
		Action string `json:"action,omitempty"`
	}{
		Value: value,
		Stop:  stop,
	}
	parts := []string{}
	if token != since {
		parts = append(parts, "flowrefresh="+url.QueryEscape(token))
	}
	if stop {
		parts = append(parts, "flowstopped=1")
	}
	if len(parts) > 0 {
		resp.Action = "?" + strings.Join(parts, "&")
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

const (
	pollFlowMaxWait = 15 * time.Second
	pollFlowTick    = 1 * time.Second
)

/*
ForkFromStep renders a form to fork a terminal flow from a specific recorded step into a new flow with optional state overrides.
*/
func (svc *Service) ForkFromStep(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: ForkFromStep
	state := wf.StateOf(r)
	stepKey := strings.TrimSpace(state.Get("step"))
	if stepKey == "" {
		return errors.New("step is required", http.StatusBadRequest)
	}

	stateField := wf.InputText("state", "").
		WithRows(15).
		WithWidth("100%").
		WithTrimSpaces(false)

	forkButton := wf.ButtonFilled("submit").Add("Fork")
	cancelButton := wf.ButtonText("").WithHref("^?fork=").Add("Cancel")

	form := wf.Form().Add(
		wf.Field().AddLeft("State overrides").AddRight(stateField),
		cancelButton,
		forkButton,
	)

	var errMsg string
	if form.ReadyToCommit(r) {
		overrides, parseErr := parseStateRaw(stateField.Value(r))
		if parseErr != nil {
			errMsg = parseErr.Error()
		} else {
			newFlowKey, forkErr := svc.foreman.Fork(r.Context(), stepKey, overrides)
			if forkErr == nil {
				wf.Redirect(w, r, "~/"+Hostname+"/flows/"+url.PathEscape(newFlowKey))
				return nil
			}
			errMsg = forkErr.Error()
		}
	}

	var errBanner any
	if errMsg != "" {
		errBanner = wf.TextStyle(errMsg).WithColorError()
	}
	page := wf.Page().Add(
		wf.AppBar("Fork from step"),
		errBanner,
		form,
	)
	return page.Draw(w, r)
}

/*
ResumeFlow renders a form to resume an interrupted flow with a resume payload.
*/
func (svc *Service) ResumeFlow(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: ResumeFlow
	state := wf.StateOf(r)
	flowKey := strings.TrimSpace(state.Get("flow"))
	if flowKey == "" {
		return errors.New("flow is required", http.StatusBadRequest)
	}

	stateField := wf.InputText("state", "").
		WithRows(15).
		WithWidth("100%").
		WithTrimSpaces(false)

	resumeButton := wf.ButtonFilled("submit").Add("Resume")
	cancelButton := wf.ButtonText("").WithHref("^?resume=").Add("Cancel")

	form := wf.Form().Add(
		wf.Field().AddLeft("Resume payload").AddRight(stateField),
		cancelButton,
		resumeButton,
	)

	var errMsg string
	if form.ReadyToCommit(r) {
		payload, parseErr := parseStateRaw(stateField.Value(r))
		if parseErr != nil {
			errMsg = parseErr.Error()
		} else {
			resumeErr := svc.foreman.Resume(r.Context(), flowKey, payload)
			if resumeErr == nil {
				wf.Redirect(w, r, "~/"+Hostname+"/flows/"+url.PathEscape(flowKey))
				return nil
			}
			errMsg = resumeErr.Error()
		}
	}

	var errBanner any
	if errMsg != "" {
		errBanner = wf.TextStyle(errMsg).WithColorError()
	}
	page := wf.Page().Add(
		wf.AppBar("Resume flow"),
		errBanner,
		form,
	)
	return page.Draw(w, r)
}
